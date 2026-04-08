package resolver_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/resolver"
)

type recordedMetric struct {
	name  string
	value float64
	attrs map[string]any
}

type recordingObservability struct {
	counters   []recordedMetric
	histograms []recordedMetric
}

func (r *recordingObservability) Logger() *slog.Logger {
	return slog.Default()
}

func (r *recordingObservability) StartSpan(
	ctx context.Context,
	_ string,
	_ ...observabilityx.Attribute,
) (context.Context, observabilityx.Span) {
	return ctx, recordingSpan{}
}

func (r *recordingObservability) AddCounter(
	_ context.Context,
	name string,
	value int64,
	attrs ...observabilityx.Attribute,
) {
	r.counters = append(r.counters, recordedMetric{
		name:  name,
		value: float64(value),
		attrs: attrsToMap(attrs),
	})
}

func (r *recordingObservability) RecordHistogram(
	_ context.Context,
	name string,
	value float64,
	attrs ...observabilityx.Attribute,
) {
	r.histograms = append(r.histograms, recordedMetric{
		name:  name,
		value: value,
		attrs: attrsToMap(attrs),
	})
}

type recordingSpan struct{}

func (recordingSpan) End() {}

func (recordingSpan) RecordError(error) {}

func (recordingSpan) SetAttributes(...observabilityx.Attribute) {}

func TestResolverMetricsRecordFallbackResolution(t *testing.T) {
	obs := &recordingObservability{}
	sourcePath, cat, _ := newResolverFixture(t, "index.html", "text/html; charset=utf-8", []byte("<html>origin</html>"), spaAssetsConfig())
	assetResolver := resolver.NewResolverWithObservabilityForTest(spaAssetsConfig(), cat, slog.New(slog.DiscardHandler), obs)

	result, err := assetResolver.Resolve(resolver.Request{Path: "docs"})
	if err != nil {
		t.Fatal(err)
	}
	if result.FilePath != sourcePath {
		t.Fatalf("expected fallback path %q, got %q", sourcePath, result.FilePath)
	}

	assertCounterMetric(t, obs.counters, "resolver_resolutions_total", 1, "result", "fallback_asset")
	assertHistogramMetric(t, obs.histograms, "resolver_resolution_duration_seconds", "result", "fallback_asset")
}

func TestResolverMetricsRecordGenerationRequests(t *testing.T) {
	obs := &recordingObservability{}
	sourcePath := writeAssetForMetricTest(t, "hero.png", "image/png")
	cat := catalog.NewInMemoryCatalog()
	upsertTestAsset(t, cat, "hero.png", sourcePath, "image/png")

	assetResolver := resolver.NewResolverWithObservabilityForTest(baseAssetsConfig(), cat, slog.New(slog.DiscardHandler), obs)
	result, err := assetResolver.Resolve(resolver.Request{Path: "hero.png", Width: 640, Format: "jpeg"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Variant != nil {
		t.Fatalf("expected original asset result, got variant %#v", result.Variant)
	}

	assertCounterMetric(t, obs.counters, "resolver_resolutions_total", 1, "result", "asset")
	assertCounterMetric(t, obs.counters, "resolver_generation_requests_total", 1, "kind", "image_width")
	assertCounterMetric(t, obs.counters, "resolver_generation_requests_total", 1, "kind", "image_format")
}

func TestResolverMetricsRecordNotFound(t *testing.T) {
	obs := &recordingObservability{}
	assetResolver := resolver.NewResolverWithObservabilityForTest(&config.Assets{Entry: "index.html"}, catalog.NewInMemoryCatalog(), slog.New(slog.DiscardHandler), obs)

	_, err := assetResolver.Resolve(resolver.Request{Path: "missing.txt"})
	if err == nil {
		t.Fatal("expected not found error")
	}

	assertCounterMetric(t, obs.counters, "resolver_resolutions_total", 1, "result", "not_found")
	assertHistogramMetric(t, obs.histograms, "resolver_resolution_duration_seconds", "result", "not_found")
}

func writeAssetForMetricTest(t *testing.T, assetPath, mediaType string) string {
	t.Helper()

	sourcePath := testFilePath(t, assetPath)
	writeTestFile(t, sourcePath, []byte("origin"))
	return sourcePath
}

func testFilePath(t *testing.T, assetPath string) string {
	t.Helper()
	return t.TempDir() + "/" + assetPath
}

func attrsToMap(attrs []observabilityx.Attribute) map[string]any {
	values := make(map[string]any, len(attrs))
	for _, attr := range attrs {
		values[attr.Key] = attr.Value
	}
	return values
}

func assertCounterMetric(t *testing.T, metrics []recordedMetric, name string, wantValue float64, key string, want any) {
	t.Helper()

	for _, metric := range metrics {
		if metric.name != name || metric.value != wantValue {
			continue
		}
		if got := metric.attrs[key]; got == want {
			return
		}
	}
	t.Fatalf("expected counter %s with %s=%v", name, key, want)
}

func assertHistogramMetric(t *testing.T, metrics []recordedMetric, name string, key string, want any) {
	t.Helper()

	for _, metric := range metrics {
		if metric.name != name {
			continue
		}
		if got := metric.attrs[key]; got == want {
			return
		}
	}
	t.Fatalf("expected histogram %s with %s=%v", name, key, want)
}
