package workerpool_test

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/workerpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNewSettingsUsesNormalizedAsyncWorkers(t *testing.T) {
	settings := workerpool.NewSettingsForTest(&config.Async{Workers: 5})
	if settings.Size != 5 {
		t.Fatalf("expected settings size 5, got %d", settings.Size)
	}
}

func TestNewPoolUsesConfiguredSize(t *testing.T) {
	pool, err := workerpool.NewPoolForTest(&workerpool.Settings{Size: 3})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cleanupErr := pool.ReleaseTimeout(3 * time.Second); cleanupErr != nil {
			t.Fatalf("release worker pool: %v", cleanupErr)
		}
	})

	if got := pool.Cap(); got != 3 {
		t.Fatalf("expected ants pool cap 3, got %d", got)
	}
}

func TestRunListFallsBackToSerialWithoutPool(t *testing.T) {
	values := collectionx.NewList(1, 2, 3)
	visited := collectionx.NewList[int]()

	err := workerpool.RunListForTest[int](context.Background(), nil, nil, "test_serial", values, func(_ context.Context, value int) error {
		visited.Add(value)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(visited.Values(), []int{1, 2, 3}) {
		t.Fatalf("expected serial visit order [1 2 3], got %v", visited.Values())
	}
}

func TestRunListUsesPool(t *testing.T) {
	pool, err := workerpool.NewPoolForTest(&workerpool.Settings{Size: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cleanupErr := pool.ReleaseTimeout(3 * time.Second); cleanupErr != nil {
			t.Fatalf("release worker pool: %v", cleanupErr)
		}
	})

	values := collectionx.NewList(1, 2, 3, 4)
	visited := collectionx.NewConcurrentSet[int]()

	err = workerpool.RunListForTest[int](context.Background(), nil, pool, "test_parallel", values, func(_ context.Context, value int) error {
		visited.Add(value)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	got := visited.Values()
	slices.Sort(got)
	if !slices.Equal(got, []int{1, 2, 3, 4}) {
		t.Fatalf("expected all values visited once, got %v", got)
	}
}

func TestRunListRecordsMetrics(t *testing.T) {
	values := collectionx.NewList(1, 2, 3)
	obs := &recordingObservability{}

	err := workerpool.RunListForTest[int](context.Background(), obs, nil, "asset_cache_warm", values, func(_ context.Context, value int) error {
		_ = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	assertRecordedMetricCount(t, obs.counters, "workerpool_batch_items_total", 1)
	assertRecordedMetricCount(t, obs.counters, "workerpool_batch_runs_total", 1)
	assertRecordedMetricCount(t, obs.histograms, "workerpool_batch_duration_seconds", 1)
	assertRecordedMetricCount(t, obs.counters, "workerpool_task_runs_total", 3)
	assertRecordedMetricCount(t, obs.histograms, "workerpool_task_duration_seconds", 3)
	assertRecordedMetricCount(t, obs.counters, "workerpool_task_submissions_total", 0)

	assertRecordedMetric(t, obs.counters, "workerpool_batch_runs_total", map[string]any{
		"workload": "asset_cache_warm",
		"mode":     "serial",
		"result":   "ok",
	})
	assertRecordedMetric(t, obs.counters, "workerpool_batch_items_total", map[string]any{
		"workload": "asset_cache_warm",
		"mode":     "serial",
	})
}

func TestRunListRecordsParallelSubmissionMetrics(t *testing.T) {
	pool, err := workerpool.NewPoolForTest(&workerpool.Settings{Size: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cleanupErr := pool.ReleaseTimeout(3 * time.Second); cleanupErr != nil {
			t.Fatalf("release worker pool: %v", cleanupErr)
		}
	})

	values := collectionx.NewList(1, 2, 3)
	obs := &recordingObservability{}
	err = workerpool.RunListForTest[int](context.Background(), obs, pool, "pipeline_warm", values, func(_ context.Context, value int) error {
		_ = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	assertRecordedMetricCount(t, obs.counters, "workerpool_task_submissions_total", 3)
	assertRecordedMetric(t, obs.counters, "workerpool_batch_runs_total", map[string]any{
		"workload": "pipeline_warm",
		"mode":     "parallel",
		"result":   "ok",
	})
	assertRecordedMetric(t, obs.counters, "workerpool_task_submissions_total", map[string]any{
		"workload": "pipeline_warm",
		"result":   "submitted",
	})
}

func TestRuntimeMetricsExposePoolState(t *testing.T) {
	pool, err := workerpool.NewPoolForTest(&workerpool.Settings{Size: 3})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cleanupErr := pool.ReleaseTimeout(3 * time.Second); cleanupErr != nil {
			t.Fatalf("release worker pool: %v", cleanupErr)
		}
	})

	registry := prometheus.NewRegistry()
	metrics := workerpool.NewRuntimeMetricsForTest(&workerpool.Settings{Size: 3}, pool)
	for _, collector := range metrics.Collectors() {
		registry.MustRegister(collector)
	}

	expected := strings.NewReader(`
# HELP spack_workerpool_capacity_current Current shared worker pool capacity
# TYPE spack_workerpool_capacity_current gauge
spack_workerpool_capacity_current 3
# HELP spack_workerpool_free_current Current number of free workers in the shared worker pool
# TYPE spack_workerpool_free_current gauge
spack_workerpool_free_current 3
# HELP spack_workerpool_running_current Current number of running worker pool goroutines
# TYPE spack_workerpool_running_current gauge
spack_workerpool_running_current 0
# HELP spack_workerpool_waiting_current Current number of waiting tasks in the shared worker pool
# TYPE spack_workerpool_waiting_current gauge
spack_workerpool_waiting_current 0
`)
	if err := testutil.GatherAndCompare(registry, expected,
		"spack_workerpool_capacity_current",
		"spack_workerpool_free_current",
		"spack_workerpool_running_current",
		"spack_workerpool_waiting_current",
	); err != nil {
		t.Fatal(err)
	}
}

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
