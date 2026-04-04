package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/eventx"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
)

func resolveQueueSize(cfg *config.Compression, workers int) int {
	queueSize := cfg.QueueCapacity()
	if queueSize < 1 {
		return workers * 64
	}
	return queueSize
}

func newServiceState(
	cfg *config.Compression,
	logger *slog.Logger,
	cat catalog.Catalog,
	metrics *Metrics,
	stages []Stage,
	bus eventx.BusRuntime,
	queueSize int,
) *Service {
	return &Service{
		cfg:                    cfg,
		logger:                 logger,
		catalog:                cat,
		metrics:                metrics,
		stages:                 stages,
		bus:                    bus,
		tasks:                  make(chan Request, queueSize),
		pending:                collectionx.NewSetWithCapacity[string](queueSize),
		variantHits:            collectionx.NewMapWithCapacity[string, time.Time](queueSize),
		cleanupDefaultMaxAge:   cfg.ParsedMaxAge(),
		cleanupNamespaceMaxAge: cfg.NamespaceMaxAges(),
		cleanupMaxCacheBytes:   cfg.MaxCacheBytes,
	}
}

func (s *Service) initializeMetrics(queueSize int) {
	if s.metrics == nil {
		return
	}
	s.metrics.QueueCapacity.Set(float64(queueSize))
	s.metrics.QueueLength.Set(0)
}

func (s *Service) start(ctx context.Context, workers, queueSize int) error {
	if !s.cfg.PipelineEnabled() {
		s.logger.Info("Pipeline disabled")
		return nil
	}
	if err := s.subscribeVariantServed(); err != nil {
		return fmt.Errorf("subscribe variant served events: %w", err)
	}
	if strings.TrimSpace(s.cfg.CacheDir) == "" {
		return nil
	}
	if err := os.MkdirAll(s.cfg.CacheDir, 0o750); err != nil {
		return fmt.Errorf("create pipeline cache directory: %w", err)
	}

	s.startWorkers(ctx, workers)
	s.logWorkersStarted(workers, queueSize)
	s.startCleanupIfNeeded(ctx)
	return nil
}

func (s *Service) startWorkers(ctx context.Context, workers int) {
	for range workers {
		s.wg.Go(func() {
			for request := range s.tasks {
				key := requestKey(request)
				s.updateQueueLengthMetric()
				s.process(ctx, request)
				s.finishRequest(key)
			}
		})
	}
}

func (s *Service) logWorkersStarted(workers, queueSize int) {
	s.logger.Info("Pipeline workers started",
		slog.Int("workers", workers),
		slog.Int("queue_size", queueSize),
		slog.String("mode", s.cfg.NormalizedMode()),
		slog.String("cache_dir", s.cfg.CacheDir),
	)
}

func (s *Service) startCleanupIfNeeded(ctx context.Context) {
	if !s.cleanupEnabled() {
		return
	}

	interval := s.cfg.ParsedCleanupInterval()
	s.cleanupStop = make(chan struct{})
	s.cleanupDone = make(chan struct{})
	go s.cleanupLoop(ctx, interval)
	s.logger.Info("Pipeline cleanup enabled",
		slog.String("interval", interval.String()),
		slog.String("max_age", s.cleanupDefaultMaxAge.String()),
		slog.String("encoding_max_age", s.cleanupNamespaceMaxAge.GetOrDefault("encoding", 0).String()),
		slog.String("image_max_age", s.cleanupNamespaceMaxAge.GetOrDefault("image", 0).String()),
		slog.Int64("max_cache_bytes", s.cleanupMaxCacheBytes),
	)
}

func (s *Service) cleanupEnabled() bool {
	return s.cleanupDefaultMaxAge > 0 || s.cleanupNamespaceMaxAge.Len() > 0 || s.cleanupMaxCacheBytes > 0
}

func (s *Service) stop(ctx context.Context) error {
	s.unsubscribeVariantServed()
	if err := s.stopCleanup(ctx); err != nil {
		return err
	}
	return s.stopWorkers(ctx)
}

func (s *Service) stopCleanup(ctx context.Context) error {
	if s.cleanupStop == nil {
		return nil
	}

	close(s.cleanupStop)
	select {
	case <-s.cleanupDone:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait for cleanup shutdown: %w", ctx.Err())
	}
}

func (s *Service) stopWorkers(ctx context.Context) error {
	close(s.tasks)
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait for worker shutdown: %w", ctx.Err())
	}
}

func (s *Service) executeStageTask(stage Stage, asset *catalog.Asset, task Task) *catalog.Variant {
	key := buildStageTaskKey(stage, asset, task)
	variantValue, err, _ := s.sf.Do(key, func() (any, error) {
		return stage.Execute(task, asset)
	})
	if err != nil {
		if IsVariantSkipped(err) {
			return nil
		}
		s.logStageFailure(stage, asset, err)
		return nil
	}

	variant, ok := variantValue.(*catalog.Variant)
	if !ok || variant == nil {
		return nil
	}
	return variant
}

func buildStageTaskKey(stage Stage, asset *catalog.Asset, task Task) string {
	return stage.Name() + "|" + asset.Path + "|" + asset.SourceHash + "|" + task.Encoding + "|" + task.Format + "|" + strconv.Itoa(task.Width)
}

func (s *Service) logStageFailure(stage Stage, asset *catalog.Asset, err error) {
	s.logger.Error("Pipeline stage failed",
		slog.String("stage", stage.Name()),
		slog.String("asset", asset.Path),
		slog.String("err", err.Error()),
	)
}

func (s *Service) upsertStageVariant(ctx context.Context, stage Stage, asset *catalog.Asset, variant *catalog.Variant) {
	if err := s.catalog.UpsertVariant(variant); err != nil {
		s.logger.Error("Catalog variant upsert failed",
			slog.String("stage", stage.Name()),
			slog.String("asset", asset.Path),
			slog.String("err", err.Error()),
		)
		return
	}
	s.publishVariantGenerated(ctx, stage.Name(), variant)
}
