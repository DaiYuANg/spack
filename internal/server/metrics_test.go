package server_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/daiyuang/spack/internal/server"
	"github.com/gofiber/fiber/v3"
)

func TestMetricsMiddlewareRecordsAssetDeliveryMetrics(t *testing.T) {
	obs := &recordingObservability{}
	app := fiber.New()
	app.Use(server.MetricsMiddlewareForTest(obs))
	app.Get("/", func(c fiber.Ctx) error {
		server.SetAssetDeliveryForTest(c, "memory_cache_hit")
		return c.SendStatus(fiber.StatusNoContent)
	})

	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer closeBody(t, response)

	assertMetricCount(t, obs.counters, "http_requests_total", 1)
	assertMetricCount(t, obs.histograms, "http_request_duration_seconds", 1)
	assertMetricCount(t, obs.counters, "http_asset_delivery_total", 1)
	assertMetricCount(t, obs.histograms, "http_asset_delivery_duration_seconds", 1)

	requestCounter := findMetric(t, obs.counters, "http_requests_total")
	assertAttrAbsent(t, requestCounter.attrs, "delivery")

	deliveryCounter := findMetric(t, obs.counters, "http_asset_delivery_total")
	assertAttrValue(t, deliveryCounter.attrs, "delivery", "memory_cache_hit")
	assertAttrValue(t, deliveryCounter.attrs, "status", "204")
}

func TestMetricsMiddlewareSkipsAssetDeliveryMetricsWithoutDelivery(t *testing.T) {
	obs := &recordingObservability{}
	app := fiber.New()
	app.Use(server.MetricsMiddlewareForTest(obs))
	app.Get("/healthz", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/healthz", http.NoBody)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer closeBody(t, response)

	assertMetricCount(t, obs.counters, "http_requests_total", 1)
	assertMetricCount(t, obs.histograms, "http_request_duration_seconds", 1)
	assertMetricCount(t, obs.counters, "http_asset_delivery_total", 0)
	assertMetricCount(t, obs.histograms, "http_asset_delivery_duration_seconds", 0)
}

type recordedMetric struct {
	name  string
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
	_ int64,
	attrs ...observabilityx.Attribute,
) {
	r.counters = append(r.counters, recordedMetric{
		name:  name,
		attrs: attrsToMap(attrs),
	})
}

func (r *recordingObservability) RecordHistogram(
	_ context.Context,
	name string,
	_ float64,
	attrs ...observabilityx.Attribute,
) {
	r.histograms = append(r.histograms, recordedMetric{
		name:  name,
		attrs: attrsToMap(attrs),
	})
}

type recordingSpan struct{}

func (recordingSpan) End() {}

func (recordingSpan) RecordError(error) {}

func (recordingSpan) SetAttributes(...observabilityx.Attribute) {}

func attrsToMap(attrs []observabilityx.Attribute) map[string]any {
	values := make(map[string]any, len(attrs))
	for _, attr := range attrs {
		values[attr.Key] = attr.Value
	}
	return values
}

func assertMetricCount(t *testing.T, metrics []recordedMetric, name string, want int) {
	t.Helper()

	count := 0
	for _, metric := range metrics {
		if metric.name == name {
			count++
		}
	}
	if count != want {
		t.Fatalf("expected %d %s metrics, got %d", want, name, count)
	}
}

func findMetric(t *testing.T, metrics []recordedMetric, name string) recordedMetric {
	t.Helper()

	for _, metric := range metrics {
		if metric.name == name {
			return metric
		}
	}
	t.Fatalf("metric %s not found", name)
	return recordedMetric{}
}

func assertAttrValue(t *testing.T, attrs map[string]any, key string, want any) {
	t.Helper()

	got, ok := attrs[key]
	if !ok {
		t.Fatalf("expected attr %s to be present", key)
	}
	if got != want {
		t.Fatalf("expected attr %s=%v, got %v", key, want, got)
	}
}

func assertAttrAbsent(t *testing.T, attrs map[string]any, key string) {
	t.Helper()

	if _, ok := attrs[key]; ok {
		t.Fatalf("expected attr %s to be absent", key)
	}
}

func closeBody(t *testing.T, response *http.Response) {
	t.Helper()

	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}
}
