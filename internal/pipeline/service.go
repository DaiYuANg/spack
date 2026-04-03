package pipeline

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"go.uber.org/fx"
	"golang.org/x/sync/singleflight"
)

type serviceIn struct {
	fx.In
	Lifecycle fx.Lifecycle
	Config    *config.Compression
	Logger    *slog.Logger
	Catalog   catalog.Catalog
	Metrics   *Metrics
	Stages    []Stage `group:"pipeline_stage"`
}

type Service struct {
	cfg     *config.Compression
	logger  *slog.Logger
	catalog catalog.Catalog
	metrics *Metrics
	stages  []Stage

	tasks     chan Request
	wg        sync.WaitGroup
	sf        singleflight.Group
	pending   map[string]struct{}
	pendingMu sync.Mutex

	cleanupMu   sync.Mutex
	cleanupStop chan struct{}
	cleanupDone chan struct{}

	hitMu       sync.Mutex
	variantHits map[string]time.Time

	cleanupDefaultMaxAge   time.Duration
	cleanupNamespaceMaxAge map[string]time.Duration
	cleanupMaxCacheBytes   int64
}

func newService(in serviceIn) *Service {
	workers := in.Config.Workers
	if workers < 1 {
		workers = 1
	}
	queueSize := in.Config.QueueCapacity()
	if queueSize < 1 {
		queueSize = workers * 64
	}

	svc := &Service{
		cfg:                    in.Config,
		logger:                 in.Logger,
		catalog:                in.Catalog,
		metrics:                in.Metrics,
		stages:                 in.Stages,
		tasks:                  make(chan Request, queueSize),
		pending:                make(map[string]struct{}, queueSize),
		variantHits:            make(map[string]time.Time, queueSize),
		cleanupDefaultMaxAge:   in.Config.ParsedMaxAge(),
		cleanupNamespaceMaxAge: in.Config.NamespaceMaxAges(),
		cleanupMaxCacheBytes:   in.Config.MaxCacheBytes,
	}
	if svc.metrics != nil {
		svc.metrics.QueueCapacity.Set(float64(queueSize))
		svc.metrics.QueueLength.Set(0)
	}

	in.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if !in.Config.PipelineEnabled() {
				in.Logger.Info("Pipeline disabled")
				return nil
			}
			if strings.TrimSpace(in.Config.CacheDir) == "" {
				return nil
			}
			if err := os.MkdirAll(in.Config.CacheDir, 0o755); err != nil {
				return err
			}
			for workerID := 0; workerID < workers; workerID++ {
				svc.wg.Add(1)
				go func() {
					defer svc.wg.Done()
					for request := range svc.tasks {
						key := requestKey(request)
						svc.updateQueueLengthMetric()
						svc.process(request)
						svc.finishRequest(key)
					}
				}()
			}
			in.Logger.Info("Pipeline workers started",
				slog.Int("workers", workers),
				slog.Int("queue_size", queueSize),
				slog.String("mode", in.Config.NormalizedMode()),
				slog.String("cache_dir", in.Config.CacheDir),
			)

			if svc.cleanupDefaultMaxAge > 0 || len(svc.cleanupNamespaceMaxAge) > 0 || svc.cleanupMaxCacheBytes > 0 {
				interval := svc.cfg.ParsedCleanupInterval()
				svc.cleanupStop = make(chan struct{})
				svc.cleanupDone = make(chan struct{})
				go svc.cleanupLoop(interval)
				in.Logger.Info("Pipeline cleanup enabled",
					slog.String("interval", interval.String()),
					slog.String("max_age", svc.cleanupDefaultMaxAge.String()),
					slog.String("encoding_max_age", svc.cleanupNamespaceMaxAge["encoding"].String()),
					slog.String("image_max_age", svc.cleanupNamespaceMaxAge["image"].String()),
					slog.Int64("max_cache_bytes", svc.cleanupMaxCacheBytes),
				)
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if svc.cleanupStop != nil {
				close(svc.cleanupStop)
				select {
				case <-svc.cleanupDone:
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			close(svc.tasks)
			done := make(chan struct{})
			go func() {
				svc.wg.Wait()
				close(done)
			}()
			select {
			case <-done:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	})

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
	if _, exists := s.pending[key]; exists {
		s.pendingMu.Unlock()
		if s.metrics != nil {
			s.metrics.EnqueueDeduplicatedTotal.Inc()
		}
		return
	}

	select {
	case s.tasks <- request:
		s.pending[key] = struct{}{}
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
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	s.hitMu.Lock()
	s.variantHits[path] = time.Now()
	s.hitMu.Unlock()
}

func (s *Service) Warm(ctx context.Context) error {
	if !s.cfg.PipelineEnabled() || s.cfg.NormalizedMode() != config.CompressionModeWarmup {
		return nil
	}

	for _, asset := range s.catalog.AllAssets() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		s.process(Request{AssetPath: asset.Path})
	}
	return nil
}

func (s *Service) process(request Request) {
	asset, ok := s.catalog.FindAsset(request.AssetPath)
	if !ok {
		return
	}

	for _, stage := range s.stages {
		tasks := stage.Plan(asset, request)
		for _, task := range tasks {
			key := stage.Name() + "|" + asset.Path + "|" + asset.SourceHash + "|" + task.Encoding + "|" + task.Format + "|" + strconv.Itoa(task.Width)
			variantValue, err, _ := s.sf.Do(key, func() (interface{}, error) {
				return stage.Execute(task, asset)
			})
			if err != nil {
				s.logger.Error("Pipeline stage failed",
					slog.String("stage", stage.Name()),
					slog.String("asset", asset.Path),
					slog.String("err", err.Error()),
				)
				continue
			}

			variant, ok := variantValue.(*catalog.Variant)
			if !ok || variant == nil {
				continue
			}
			if err := s.catalog.UpsertVariant(variant); err != nil {
				s.logger.Error("Catalog variant upsert failed",
					slog.String("stage", stage.Name()),
					slog.String("asset", asset.Path),
					slog.String("err", err.Error()),
				)
			}
		}
	}
}

func (s *Service) finishRequest(key string) {
	s.pendingMu.Lock()
	delete(s.pending, key)
	s.pendingMu.Unlock()
}

func requestKey(request Request) string {
	assetPath := strings.TrimSpace(request.AssetPath)
	encodings := normalizeRequestStrings(request.PreferredEncodings)
	formats := normalizeRequestStrings(request.PreferredFormats)
	widths := normalizeRequestInts(request.PreferredWidths)
	return assetPath + "|e=" + strings.Join(encodings, ",") + "|f=" + strings.Join(formats, ",") + "|w=" + joinInts(widths)
}

func normalizeRequestStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	sort.Strings(out)
	return out
}

func normalizeRequestInts(values []int) []int {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[int]struct{}, len(values))
	out := make([]int, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Ints(out)
	return out
}

func joinInts(values []int) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.Itoa(value))
	}
	return strings.Join(parts, ",")
}

type cleanupFile struct {
	path      string
	namespace string
	size      int64
	modTime   time.Time
	lastUsed  time.Time
}

type cleanupResult struct {
	scanned          int
	removed          int
	removedBytes     int64
	totalBytes       int64
	removedTTL       int
	removedSize      int
	removedTTLBytes  int64
	removedSizeBytes int64
}

func (s *Service) cleanupLoop(interval time.Duration) {
	defer close(s.cleanupDone)
	s.cleanupOnce()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-s.cleanupStop:
			return
		case <-ticker.C:
			s.cleanupOnce()
		}
	}
}

func (s *Service) cleanupOnce() {
	result := s.cleanupArtifacts(time.Now())
	if s.metrics != nil {
		s.metrics.CleanupRunsTotal.Inc()
		if result.removedTTL > 0 {
			s.metrics.CleanupRemovedTotal.WithLabelValues("ttl").Add(float64(result.removedTTL))
			s.metrics.CleanupRemovedBytesTotal.WithLabelValues("ttl").Add(float64(result.removedTTLBytes))
		}
		if result.removedSize > 0 {
			s.metrics.CleanupRemovedTotal.WithLabelValues("size").Add(float64(result.removedSize))
			s.metrics.CleanupRemovedBytesTotal.WithLabelValues("size").Add(float64(result.removedSizeBytes))
		}
	}
	if result.removed > 0 {
		s.logger.Info("Pipeline cache cleanup completed",
			slog.Int("scanned", result.scanned),
			slog.Int("removed", result.removed),
			slog.Int("removed_ttl", result.removedTTL),
			slog.Int("removed_size", result.removedSize),
			slog.Int64("removed_bytes", result.removedBytes),
			slog.Int64("remaining_bytes", result.totalBytes),
		)
	}
}

func (s *Service) cleanupArtifacts(now time.Time) cleanupResult {
	if strings.TrimSpace(s.cfg.CacheDir) == "" {
		return cleanupResult{}
	}

	s.cleanupMu.Lock()
	defer s.cleanupMu.Unlock()

	files, err := collectCleanupFiles(s.cfg.CacheDir)
	if err != nil {
		s.logger.Error("Pipeline cache scan failed", slog.String("err", err.Error()))
		return cleanupResult{}
	}

	result := cleanupResult{scanned: len(files)}
	for index := range files {
		file := &files[index]
		file.namespace = cleanupNamespace(file.path, s.cfg.CacheDir)
		file.lastUsed = s.effectiveLastUsed(file.path, file.modTime)
		result.totalBytes += file.size
	}

	remaining := files[:0]
	for _, file := range files {
		maxAge := s.cleanupMaxAgeForNamespace(file.namespace)
		if maxAge > 0 && now.Sub(file.lastUsed) > maxAge {
			if s.removeCleanupFile(file) {
				result.removed++
				result.removedTTL++
				result.removedBytes += file.size
				result.removedTTLBytes += file.size
				result.totalBytes -= file.size
				continue
			}
		}
		remaining = append(remaining, file)
	}

	if s.cleanupMaxCacheBytes > 0 && result.totalBytes > s.cleanupMaxCacheBytes {
		sort.Slice(remaining, func(i, j int) bool {
			return remaining[i].lastUsed.Before(remaining[j].lastUsed)
		})

		for _, file := range remaining {
			if result.totalBytes <= s.cleanupMaxCacheBytes {
				break
			}
			if !s.removeCleanupFile(file) {
				continue
			}
			result.removed++
			result.removedSize++
			result.removedBytes += file.size
			result.removedSizeBytes += file.size
			result.totalBytes -= file.size
		}
	}

	return result
}

func (s *Service) removeCleanupFile(file cleanupFile) bool {
	if err := os.Remove(file.path); err != nil {
		if !os.IsNotExist(err) {
			s.logger.Debug("Pipeline cache cleanup remove failed",
				slog.String("path", file.path),
				slog.String("err", err.Error()),
			)
			return false
		}
	}
	s.catalog.DeleteVariantByArtifactPath(file.path)
	s.clearVariantHit(file.path)
	return true
}

func cleanupNamespace(path string, root string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return ""
	}
	normalized := filepath.ToSlash(relative)
	parts := strings.Split(normalized, "/")
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

func (s *Service) cleanupMaxAgeForNamespace(namespace string) time.Duration {
	if maxAge, ok := s.cleanupNamespaceMaxAge[namespace]; ok && maxAge > 0 {
		return maxAge
	}
	return s.cleanupDefaultMaxAge
}

func (s *Service) effectiveLastUsed(path string, modTime time.Time) time.Time {
	s.hitMu.Lock()
	lastHit, ok := s.variantHits[path]
	s.hitMu.Unlock()
	if ok && lastHit.After(modTime) {
		return lastHit
	}
	return modTime
}

func (s *Service) clearVariantHit(path string) {
	s.hitMu.Lock()
	delete(s.variantHits, path)
	s.hitMu.Unlock()
}

func (s *Service) updateQueueLengthMetric() {
	if s.metrics != nil {
		s.metrics.QueueLength.Set(float64(len(s.tasks)))
	}
}

func collectCleanupFiles(root string) ([]cleanupFile, error) {
	files := make([]cleanupFile, 0, 64)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}
		files = append(files, cleanupFile{
			path:    path,
			size:    info.Size(),
			modTime: info.ModTime(),
		})
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return files, nil
}
