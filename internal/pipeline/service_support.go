package pipeline

import (
	"cmp"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
)

func requestKey(request Request) string {
	assetPath := strings.TrimSpace(request.AssetPath)
	encodings := normalizeRequestStrings(request.PreferredEncodings)
	formats := normalizeRequestStrings(request.PreferredFormats)
	widths := normalizeRequestInts(request.PreferredWidths)
	return assetPath + "|e=" + encodings.Join(",") + "|f=" + formats.Join(",") + "|w=" + joinInts(widths)
}

func normalizeRequestStrings(values collectionx.List[string]) collectionx.List[string] {
	if values.IsEmpty() {
		return collectionx.NewList[string]()
	}

	normalized := collectionx.FilterMapList(values, func(_ int, value string) (string, bool) {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "" {
			return "", false
		}
		return normalized, true
	})
	if normalized.IsEmpty() {
		return normalized
	}

	normalized.Sort(strings.Compare)
	return collectionx.NewList(collectionx.NewOrderedSet(normalized.Values()...).Values()...)
}

func normalizeRequestInts(values collectionx.List[int]) collectionx.List[int] {
	if values.IsEmpty() {
		return collectionx.NewList[int]()
	}

	normalized := collectionx.FilterMapList(values, func(_ int, value int) (int, bool) {
		if value <= 0 {
			return 0, false
		}
		return value, true
	})
	if normalized.IsEmpty() {
		return normalized
	}

	normalized.Sort(cmp.Compare[int])
	return collectionx.NewList(collectionx.NewOrderedSet(normalized.Values()...).Values()...)
}

func joinInts(values collectionx.List[int]) string {
	return collectionx.MapList(values, func(_ int, value int) string {
		return strconv.Itoa(value)
	}).Join(",")
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

	result := s.prepareCleanupResult(files)
	remaining := s.removeExpiredCleanupFiles(files, now, &result)
	s.enforceCleanupCacheLimit(remaining, &result)
	return result
}

func (s *Service) prepareCleanupResult(files []cleanupFile) cleanupResult {
	result := cleanupResult{scanned: len(files)}
	for index := range files {
		file := &files[index]
		file.namespace = cleanupNamespace(file.path, s.cfg.CacheDir)
		file.lastUsed = s.effectiveLastUsed(file.path, file.modTime)
		result.totalBytes += file.size
	}
	return result
}

func (s *Service) removeExpiredCleanupFiles(files []cleanupFile, now time.Time, result *cleanupResult) []cleanupFile {
	remaining := files[:0]
	for _, file := range files {
		if s.shouldRemoveExpiredFile(file, now) && s.removeCleanupFile(file) {
			recordExpiredCleanupRemoval(result, file.size)
			continue
		}
		remaining = append(remaining, file)
	}
	return remaining
}

func (s *Service) shouldRemoveExpiredFile(file cleanupFile, now time.Time) bool {
	maxAge := s.cleanupMaxAgeForNamespace(file.namespace)
	return maxAge > 0 && now.Sub(file.lastUsed) > maxAge
}

func recordExpiredCleanupRemoval(result *cleanupResult, size int64) {
	result.removed++
	result.removedTTL++
	result.removedBytes += size
	result.removedTTLBytes += size
	result.totalBytes -= size
}

func (s *Service) enforceCleanupCacheLimit(files []cleanupFile, result *cleanupResult) {
	if s.cleanupMaxCacheBytes <= 0 || result.totalBytes <= s.cleanupMaxCacheBytes {
		return
	}

	for _, file := range sortCleanupFilesByLastUsed(files) {
		if result.totalBytes <= s.cleanupMaxCacheBytes {
			return
		}
		if !s.removeCleanupFile(file) {
			continue
		}
		recordSizeCleanupRemoval(result, file.size)
	}
}

func sortCleanupFilesByLastUsed(files []cleanupFile) []cleanupFile {
	return collectionx.NewList(files...).Sort(func(left, right cleanupFile) int {
		return left.lastUsed.Compare(right.lastUsed)
	}).Values()
}

func recordSizeCleanupRemoval(result *cleanupResult, size int64) {
	result.removed++
	result.removedSize++
	result.removedBytes += size
	result.removedSizeBytes += size
	result.totalBytes -= size
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

func cleanupNamespace(path, root string) string {
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
	if maxAge, ok := s.cleanupNamespaceMaxAge.Get(namespace); ok && maxAge > 0 {
		return maxAge
	}
	return s.cleanupDefaultMaxAge
}

func (s *Service) effectiveLastUsed(path string, modTime time.Time) time.Time {
	s.hitMu.Lock()
	lastHit, ok := s.variantHits.Get(path)
	s.hitMu.Unlock()
	if ok && lastHit.After(modTime) {
		return lastHit
	}
	return modTime
}

func (s *Service) clearVariantHit(path string) {
	s.hitMu.Lock()
	s.variantHits.Delete(path)
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
			return fmt.Errorf("read cleanup file info: %w", err)
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
		return nil, fmt.Errorf("walk cleanup directory: %w", err)
	}
	return files, nil
}
