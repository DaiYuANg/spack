package asyncx_test

import (
	"context"
	"log/slog"
	"sync"
	"testing"

	"github.com/DaiYuANg/arcgo/observabilityx"
)

type recordedMetric struct {
	name  string
	attrs map[string]any
}

type recordingObservability struct {
	mu         sync.Mutex
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

func (r *recordingObservability) Counter(spec observabilityx.CounterSpec) observabilityx.Counter {
	return recordingCounter{name: spec.Name, metrics: &r.counters, mu: &r.mu}
}

func (r *recordingObservability) UpDownCounter(observabilityx.UpDownCounterSpec) observabilityx.UpDownCounter {
	return noopUpDownCounter{}
}

func (r *recordingObservability) Histogram(spec observabilityx.HistogramSpec) observabilityx.Histogram {
	return recordingHistogram{name: spec.Name, metrics: &r.histograms, mu: &r.mu}
}

func (r *recordingObservability) Gauge(observabilityx.GaugeSpec) observabilityx.Gauge {
	return noopGauge{}
}

type recordingCounter struct {
	name    string
	metrics *[]recordedMetric
	mu      *sync.Mutex
}

func (r recordingCounter) Add(_ context.Context, _ int64, attrs ...observabilityx.Attribute) {
	r.mu.Lock()
	defer r.mu.Unlock()
	*r.metrics = append(*r.metrics, recordedMetric{
		name:  r.name,
		attrs: recordedAttrs(attrs),
	})
}

type recordingHistogram struct {
	name    string
	metrics *[]recordedMetric
	mu      *sync.Mutex
}

func (r recordingHistogram) Record(_ context.Context, _ float64, attrs ...observabilityx.Attribute) {
	r.mu.Lock()
	defer r.mu.Unlock()
	*r.metrics = append(*r.metrics, recordedMetric{
		name:  r.name,
		attrs: recordedAttrs(attrs),
	})
}

type noopUpDownCounter struct{}

func (noopUpDownCounter) Add(context.Context, int64, ...observabilityx.Attribute) {}

type noopGauge struct{}

func (noopGauge) Set(context.Context, float64, ...observabilityx.Attribute) {}

type recordingSpan struct{}

func (recordingSpan) End() {}

func (recordingSpan) RecordError(error) {}

func (recordingSpan) SetAttributes(...observabilityx.Attribute) {}

func recordedAttrs(attrs []observabilityx.Attribute) map[string]any {
	values := make(map[string]any, len(attrs))
	for _, attr := range attrs {
		values[attr.Key] = attr.Value
	}
	return values
}

func assertRecordedMetricCount(t *testing.T, metrics []recordedMetric, name string, want int) {
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

func assertRecordedMetric(t *testing.T, metrics []recordedMetric, name string, want map[string]any) {
	t.Helper()

	for _, metric := range metrics {
		if metric.name != name {
			continue
		}
		matched := true
		for key, value := range want {
			if metric.attrs[key] != value {
				matched = false
				break
			}
		}
		if matched {
			return
		}
	}
	t.Fatalf("metric %s with attrs %v not found", name, want)
}
