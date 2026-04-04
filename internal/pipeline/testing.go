package pipeline

import (
	"log/slog"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/artifact"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
)

// NewCompressionStageForTest exposes compression stage construction for external tests.
func NewCompressionStageForTest(cfg *config.Compression, store artifact.Store, cat catalog.Catalog) Stage {
	return newCompressionStageFromDeps(cfg, store, cat)
}

// NewImageStageForTest exposes image stage construction for external tests.
func NewImageStageForTest(cfg *config.Image, store artifact.Store, cat catalog.Catalog) Stage {
	return newImageStageFromDeps(cfg, store, cat)
}

// NormalizeEncodingsForTest exposes compression encoding normalization for external tests.
func NormalizeEncodingsForTest(values collectionx.List[string]) collectionx.List[string] {
	return normalizeEncodings(values)
}

// NormalizeImageFormatsForTest exposes image format normalization for external tests.
func NormalizeImageFormatsForTest(values collectionx.List[string]) collectionx.List[string] {
	return normalizeImageFormats(values)
}

// NormalizeRequestStringsForTest exposes request string normalization for external tests.
func NormalizeRequestStringsForTest(values collectionx.List[string]) collectionx.List[string] {
	return normalizeRequestStrings(values)
}

// NormalizeRequestIntsForTest exposes request integer normalization for external tests.
func NormalizeRequestIntsForTest(values collectionx.List[int]) collectionx.List[int] {
	return normalizeRequestInts(values)
}

// NewServiceForTest exposes service construction for external tests.
func NewServiceForTest(cfg *config.Compression, logger *slog.Logger, cat catalog.Catalog, queueSize int) *Service {
	return newServiceState(cfg, logger, cat, nil, nil, queueSize)
}

// PendingCountForTest exposes pending queue cardinality for external tests.
func PendingCountForTest(s *Service) int {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	return s.pending.Len()
}

// QueuedCountForTest exposes queued request count for external tests.
func QueuedCountForTest(s *Service) int {
	return len(s.tasks)
}

// CleanupRemovedForTest exposes cleanup execution for external tests.
func CleanupRemovedForTest(s *Service, now time.Time) int {
	return s.cleanupArtifacts(now).removed
}
