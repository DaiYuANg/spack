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
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"golang.org/x/sync/singleflight"
)

type Service struct {
	cfg     *config.Compression
	logger  *slog.Logger
	catalog catalog.Catalog
	metrics *Metrics
	stages  []Stage
	bus     eventx.BusRuntime

	tasks     chan Request
	wg        sync.WaitGroup
	sf        singleflight.Group
	pending   collectionx.Set[string]
	pendingMu sync.Mutex

	cleanupMu   sync.Mutex
	cleanupStop chan struct{}
	cleanupDone chan struct{}

	hitMu       sync.Mutex
	variantHits collectionx.Map[string, time.Time]

	cleanupDefaultMaxAge   time.Duration
	cleanupNamespaceMaxAge collectionx.Map[string, time.Duration]
	cleanupMaxCacheBytes   int64

	variantServedUnsubscribe func()
}

func newServiceFromDeps(
	cfg *config.Compression,
	logger *slog.Logger,
	cat catalog.Catalog,
	metrics *Metrics,
	stages []Stage,
	bus eventx.BusRuntime,
) *Service {
	workers := max(cfg.Workers, 1)
	queueSize := resolveQueueSize(cfg, workers)

	svc := newServiceState(cfg, logger, cat, metrics, stages, bus, queueSize)
	svc.initializeMetrics(queueSize)
	return svc
}

func (s *Service) Enqueue(request Request) {
	if !s.cfg.PipelineEnabled() || s.cfg.NormalizedMode() != config.CompressionModeLazy {
		return
	}
	if strings.TrimSpace(request.AssetPath) == "" {
		return
	}

	key := requestKey(request)
	s.pendingMu.Lock()
	if s.pending.Contains(key) {
		s.pendingMu.Unlock()
		if s.metrics != nil {
			s.metrics.EnqueueDeduplicatedTotal.Inc()
		}
		return
	}

	select {
	case s.tasks <- request:
		s.pending.Add(key)
		s.pendingMu.Unlock()
		s.updateQueueLengthMetric()
	default:
		s.pendingMu.Unlock()
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
	s.hitMu.Lock()
	s.variantHits.Set(path, hitAt)
	s.hitMu.Unlock()
}

func (s *Service) Warm(ctx context.Context) error {
	if !s.cfg.PipelineEnabled() || s.cfg.NormalizedMode() != config.CompressionModeWarmup {
		return nil
	}

	s.catalog.AllAssets().Range(func(_ int, asset *catalog.Asset) bool {
		select {
		case <-ctx.Done():
			return false
		default:
		}

		s.process(ctx, Request{AssetPath: asset.Path})
		return true
	})
	if ctx.Err() != nil {
		return fmt.Errorf("warm pipeline: %w", ctx.Err())
	}
	return nil
}

func (s *Service) process(ctx context.Context, request Request) {
	asset, ok := s.catalog.FindAsset(request.AssetPath)
	if !ok {
		return
	}

	for _, stage := range s.stages {
		for _, task := range stage.Plan(asset, request) {
			if variant := s.executeStageTask(stage, asset, task); variant != nil {
				s.upsertStageVariant(ctx, stage, asset, variant)
			}
		}
	}
}

func (s *Service) finishRequest(key string) {
	s.pendingMu.Lock()
	s.pending.Remove(key)
	s.pendingMu.Unlock()
}
