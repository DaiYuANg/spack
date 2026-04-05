package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/eventx"
	"github.com/daiyuang/spack/internal/cachepolicy"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/workerpool"
	"github.com/panjf2000/ants/v2"
	"golang.org/x/sync/singleflight"
)

type Service struct {
	cfg     *config.Compression
	logger  *slog.Logger
	catalog catalog.Catalog
	metrics *Metrics
	stages  collectionx.List[Stage]
	bus     eventx.BusRuntime

	tasks   chan Request
	wg      sync.WaitGroup
	sf      singleflight.Group
	pending collectionx.ConcurrentSet[string]

	cleanupMu   sync.Mutex
	cleanupStop chan struct{}
	cleanupDone chan struct{}

	variantHits collectionx.ConcurrentMap[string, time.Time]
	warmPool    *ants.Pool

	artifactPolicy cachepolicy.ArtifactPolicy

	variantServedUnsubscribe func()
}

func (s *Service) Enqueue(request Request) {
	if !s.cfg.PipelineEnabled() || s.cfg.NormalizedMode() != config.CompressionModeLazy {
		return
	}
	if strings.TrimSpace(request.AssetPath) == "" {
		return
	}

	key := requestKey(request)
	if !s.pending.AddIfAbsent(key) {
		if s.metrics != nil {
			s.metrics.EnqueueDeduplicatedTotal.Inc()
		}
		return
	}

	select {
	case s.tasks <- request:
		s.updateQueueLengthMetric()
	default:
		s.pending.Remove(key)
		if s.metrics != nil {
			s.metrics.EnqueueDroppedTotal.Inc()
		}
		s.logger.Debug("Pipeline queue full",
			slog.String("asset", request.AssetPath),
			slog.Int("queue_len", len(s.tasks)),
			slog.Int("queue_cap", cap(s.tasks)),
		)
	}
}

func (s *Service) MarkVariantHit(path string) {
	s.markVariantHitAt(path, time.Now())
}

func (s *Service) markVariantHitAt(path string, hitAt time.Time) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	s.variantHits.Set(path, hitAt)
}

func (s *Service) Warm(ctx context.Context) error {
	if !s.cfg.PipelineEnabled() || s.cfg.NormalizedMode() != config.CompressionModeWarmup {
		return nil
	}

	err := workerpool.RunList[*catalog.Asset](ctx, s.warmPool, s.catalog.AllAssets(), func(ctx context.Context, asset *catalog.Asset) error {
		s.process(ctx, Request{AssetPath: asset.Path})
		return nil
	})
	if err != nil {
		return fmt.Errorf("warm pipeline: %w", err)
	}
	return nil
}

func (s *Service) process(ctx context.Context, request Request) {
	asset, ok := s.catalog.FindAsset(request.AssetPath)
	if !ok {
		return
	}
	if ctx.Err() != nil {
		return
	}

	s.stages.Range(func(_ int, stage Stage) bool {
		for _, task := range stage.Plan(asset, request) {
			if variant := s.executeStageTask(stage, asset, task); variant != nil {
				s.upsertStageVariant(ctx, stage, asset, variant)
			}
		}
		return true
	})
}

func (s *Service) finishRequest(key string) {
	s.pending.Remove(key)
}
