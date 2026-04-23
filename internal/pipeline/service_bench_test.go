package pipeline_test

import (
	"fmt"
	"log/slog"
	"testing"

	"github.com/arcgolabs/collectionx"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/pipeline"
)

func BenchmarkServiceEnqueueUnique(b *testing.B) {
	svc := pipeline.NewServiceForTest(&config.Compression{
		Enable: true,
		Mode:   config.CompressionModeLazy,
	}, slog.New(slog.DiscardHandler), catalog.NewInMemoryCatalog(), 1)

	requests := make([]pipeline.Request, 1024)
	for i := range requests {
		requests[i] = pipeline.Request{
			AssetPath:          fmt.Sprintf("asset-%d.js", i),
			PreferredEncodings: collectionx.NewList("br", "gzip"),
			PreferredFormats:   collectionx.NewList("jpeg", "png"),
			PreferredWidths:    collectionx.NewList(640, 1280),
		}
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := range b.N {
		req := requests[i%len(requests)]
		svc.Enqueue(req)

		queued := pipeline.DequeueRequestForTest(svc)
		pipeline.FinishRequestForTest(svc, queued)
	}
}

func BenchmarkServiceEnqueueDeduplicated(b *testing.B) {
	svc := pipeline.NewServiceForTest(&config.Compression{
		Enable: true,
		Mode:   config.CompressionModeLazy,
	}, slog.New(slog.DiscardHandler), catalog.NewInMemoryCatalog(), 1)

	req := pipeline.Request{
		AssetPath:          "hero.png",
		PreferredEncodings: collectionx.NewList("br", "gzip"),
		PreferredFormats:   collectionx.NewList("jpeg", "png"),
		PreferredWidths:    collectionx.NewList(640, 1280),
	}
	svc.Enqueue(req)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		svc.Enqueue(req)
	}
}
