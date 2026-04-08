package pipeline

import (
	"context"
	"log/slog"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/eventx"
	"github.com/daiyuang/spack/internal/artifact"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/contentcoding"
)

// NewCompressionStageForTest exposes compression stage construction for external tests.
func NewCompressionStageForTest(cfg *config.Compression, store artifact.Store, cat catalog.Catalog) Stage {
	return newCompressionStage(
		cfg,
		contentcoding.NewRegistry(contentcoding.Options{
			BrotliQuality: cfg.BrotliQuality,
			GzipLevel:     cfg.GzipLevel,
			ZstdLevel:     cfg.ZstdLevel,
		}, cfg.NormalizedEncodings()),
		store,
		cat,
	)
}

// NewImageStageForTest exposes image stage construction for external tests.
func NewImageStageForTest(cfg *config.Image, store artifact.Store, cat catalog.Catalog) Stage {
	return newImageStage(cfg, store, cat)
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
	return newServiceState(cfg, logger, cat, nil, nil, nil, nil, queueSize)
}

// NewServiceWithBusForTest exposes service construction with an event bus for external tests.
func NewServiceWithBusForTest(
	cfg *config.Compression,
	logger *slog.Logger,
	cat catalog.Catalog,
	bus eventx.BusRuntime,
	queueSize int,
) *Service {
	return newServiceState(cfg, logger, cat, nil, nil, bus, nil, queueSize)
}

type testStage struct {
	name string
}

func (s testStage) Name() string {
	return s.name
}

func (testStage) Plan(_ *catalog.Asset, _ Request) []Task {
	return nil
}

func (testStage) Execute(_ Task, _ *catalog.Asset) (*catalog.Variant, error) {
	return nil, ErrVariantSkipped
}

// PendingCountForTest exposes pending queue cardinality for external tests.
func PendingCountForTest(s *Service) int {
	return s.pending.Len()
}

// QueuedCountForTest exposes queued request count for external tests.
func QueuedCountForTest(s *Service) int {
	return len(s.tasks)
}

// CleanupRemovedForTest exposes cleanup execution for external tests.
func CleanupRemovedForTest(s *Service, now time.Time) int {
	return s.cleanupArtifacts(context.Background(), now).removed
}

// SubscribeVariantServedForTest exposes event subscription for external tests.
func SubscribeVariantServedForTest(s *Service) error {
	return s.subscribeVariantServed()
}

// UpsertStageVariantForTest exposes catalog upsert and side effects for external tests.
func UpsertStageVariantForTest(s *Service, stageName string, asset *catalog.Asset, variant *catalog.Variant) {
	s.upsertStageVariant(context.Background(), testStage{name: stageName}, asset, variant)
}
