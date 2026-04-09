package assetcache_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/DaiYuANg/arcgo/observabilityx"
)

type recordingObservability struct {
	counters map[string]int64
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
	return recordingCounter{name: spec.Name, metrics: &r.counters}
}

func (r *recordingObservability) UpDownCounter(observabilityx.UpDownCounterSpec) observabilityx.UpDownCounter {
	return noopUpDownCounter{}
}

func (r *recordingObservability) Histogram(observabilityx.HistogramSpec) observabilityx.Histogram {
	return noopHistogram{}
}

func (r *recordingObservability) Gauge(observabilityx.GaugeSpec) observabilityx.Gauge {
	return noopGauge{}
}

type recordingCounter struct {
	name    string
	metrics *map[string]int64
}

func (r recordingCounter) Add(_ context.Context, value int64, _ ...observabilityx.Attribute) {
	if *r.metrics == nil {
		*r.metrics = map[string]int64{}
	}
	(*r.metrics)[r.name] += value
}

type noopUpDownCounter struct{}

func (noopUpDownCounter) Add(context.Context, int64, ...observabilityx.Attribute) {}

type noopHistogram struct{}

func (noopHistogram) Record(context.Context, float64, ...observabilityx.Attribute) {}

type noopGauge struct{}

func (noopGauge) Set(context.Context, float64, ...observabilityx.Attribute) {}

type recordingSpan struct{}

func (recordingSpan) End() {}

func (recordingSpan) RecordError(error) {}

func (recordingSpan) SetAttributes(...observabilityx.Attribute) {}

func assertCounterValue(t *testing.T, obs *recordingObservability, name string, want int64) {
	t.Helper()

	got := obs.counters[name]
	if got != want {
		t.Fatalf("expected counter %s=%d, got %d", name, want, got)
	}
}
