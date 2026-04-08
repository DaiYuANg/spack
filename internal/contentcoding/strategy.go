package contentcoding

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
	"github.com/samber/lo"
)

type Options struct {
	BrotliQuality int
	GzipLevel     int
	ZstdLevel     int
}

type Strategy interface {
	Name() string
	Suffix() string
	Compress(raw []byte) ([]byte, error)
}

type Registry struct {
	strategies collectionx.Map[string, Strategy]
}

func NewRegistry(opts Options, enabled collectionx.List[string]) Registry {
	all := collectionx.NewMapWithCapacity[string, Strategy](3)
	newBuiltinStrategies(opts).Each(func(_ int, strategy Strategy) {
		all.Set(strategy.Name(), strategy)
	})

	enabled = NormalizeNames(enabled)
	if enabled.IsEmpty() {
		enabled = DefaultNames()
	}

	strategies := collectionx.NewMapWithCapacity[string, Strategy](enabled.Len())
	enabled.Each(func(_ int, name string) {
		if strategy, ok := all.Get(name); ok {
			strategies.Set(name, strategy)
		}
	})
	return Registry{strategies: strategies}
}

func (r Registry) Lookup(name string) (Strategy, bool) {
	return r.strategies.Get(name)
}

func DefaultNames() collectionx.List[string] {
	return collectionx.NewList("br", "zstd", "gzip")
}

func IsSupported(name string) bool {
	return lo.Contains(DefaultNames().Values(), strings.ToLower(strings.TrimSpace(name)))
}

func ParseNames(raw string) collectionx.List[string] {
	if strings.TrimSpace(raw) == "" {
		return collectionx.NewList[string]()
	}
	return NormalizeNames(collectionx.NewList(strings.Split(raw, ",")...))
}

func ResolveNames(raw string) collectionx.List[string] {
	names := ParseNames(raw)
	if names.IsEmpty() {
		return DefaultNames()
	}
	return names
}

func NormalizeNames(values collectionx.List[string]) collectionx.List[string] {
	if values.IsEmpty() {
		return collectionx.NewList[string]()
	}

	normalized := lo.FilterMap(values.Values(), func(raw string, _ int) (string, bool) {
		name := strings.ToLower(strings.TrimSpace(raw))
		return name, IsSupported(name)
	})
	if len(normalized) == 0 {
		return collectionx.NewList[string]()
	}
	return collectionx.NewList(collectionx.NewOrderedSet(normalized...).Values()...)
}

func newBuiltinStrategies(opts Options) collectionx.List[Strategy] {
	return collectionx.NewList[Strategy](
		NewBrotliStrategy(opts.BrotliQuality),
		NewZstdStrategy(opts.ZstdLevel),
		NewGzipStrategy(opts.GzipLevel),
	)
}

type BrotliStrategy struct {
	quality int
}

func NewBrotliStrategy(quality int) Strategy {
	return BrotliStrategy{quality: quality}
}

func (s BrotliStrategy) Name() string {
	return "br"
}

func (s BrotliStrategy) Suffix() string {
	return ".br"
}

func (s BrotliStrategy) Compress(raw []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := brotli.NewWriterLevel(&buf, clampBrotliQuality(s.quality))
	if _, err := writer.Write(raw); err != nil {
		return nil, fmt.Errorf("write brotli payload: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close brotli writer: %w", err)
	}
	return buf.Bytes(), nil
}

type ZstdStrategy struct {
	level int
}

func NewZstdStrategy(level int) Strategy {
	return ZstdStrategy{level: level}
}

func (s ZstdStrategy) Name() string {
	return "zstd"
}

func (s ZstdStrategy) Suffix() string {
	return ".zst"
}

func (s ZstdStrategy) Compress(raw []byte) ([]byte, error) {
	encoder, err := zstd.NewWriter(nil,
		zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(clampZstdLevel(s.level))),
		zstd.WithEncoderConcurrency(1),
	)
	if err != nil {
		return nil, fmt.Errorf("create zstd encoder: %w", err)
	}
	defer func() {
		_ = encoder.Close()
	}()

	return encoder.EncodeAll(raw, nil), nil
}

type GzipStrategy struct {
	level int
}

func NewGzipStrategy(level int) Strategy {
	return GzipStrategy{level: level}
}

func (s GzipStrategy) Name() string {
	return "gzip"
}

func (s GzipStrategy) Suffix() string {
	return ".gz"
}

func (s GzipStrategy) Compress(raw []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, clampGzipLevel(s.level))
	if err != nil {
		return nil, fmt.Errorf("create gzip writer: %w", err)
	}
	if _, err := writer.Write(raw); err != nil {
		return nil, fmt.Errorf("write gzip payload: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close gzip writer: %w", err)
	}
	return buf.Bytes(), nil
}

func clampGzipLevel(level int) int {
	if level < gzip.BestSpeed || level > gzip.BestCompression {
		return gzip.DefaultCompression
	}
	return level
}

func clampBrotliQuality(q int) int {
	if q < 0 {
		return 0
	}
	if q > 11 {
		return 11
	}
	return q
}

func clampZstdLevel(level int) int {
	if level < 0 {
		return 0
	}
	if level > 22 {
		return 22
	}
	return level
}
