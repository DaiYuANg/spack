package workerpool

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/panjf2000/ants/v2"
)

var (
	workerpoolBatchItemsTotalSpec = observabilityx.NewCounterSpec(
		"workerpool_batch_items_total",
		observabilityx.WithDescription("Total number of items submitted to workerpool batch runs."),
		observabilityx.WithLabelKeys("workload", "mode"),
	)
	workerpoolBatchRunsTotalSpec = observabilityx.NewCounterSpec(
		"workerpool_batch_runs_total",
		observabilityx.WithDescription("Total number of workerpool batch executions."),
		observabilityx.WithLabelKeys("workload", "mode", "result"),
	)
	workerpoolBatchDurationSpec = observabilityx.NewHistogramSpec(
		"workerpool_batch_duration_seconds",
		observabilityx.WithDescription("Workerpool batch execution duration in seconds."),
		observabilityx.WithUnit("s"),
		observabilityx.WithLabelKeys("workload", "mode", "result"),
	)
	workerpoolTaskRunsTotalSpec = observabilityx.NewCounterSpec(
		"workerpool_task_runs_total",
		observabilityx.WithDescription("Total number of individual workerpool task executions."),
		observabilityx.WithLabelKeys("workload", "mode", "result"),
	)
	workerpoolTaskDurationSpec = observabilityx.NewHistogramSpec(
		"workerpool_task_duration_seconds",
		observabilityx.WithDescription("Individual workerpool task execution duration in seconds."),
		observabilityx.WithUnit("s"),
		observabilityx.WithLabelKeys("workload", "mode", "result"),
	)
	workerpoolTaskSubmissionsTotalSpec = observabilityx.NewCounterSpec(
		"workerpool_task_submissions_total",
		observabilityx.WithDescription("Total number of workerpool task submission attempts."),
		observabilityx.WithLabelKeys("workload", "result"),
	)
)

// RunList executes list items with the shared worker pool when available.
// It falls back to serial execution when pool is nil.
func RunList[T any](
	ctx context.Context,
	obs observabilityx.Observability,
	pool *ants.Pool,
	workload string,
	values collectionx.List[T],
	run func(context.Context, T) error,
) error {
	obs = observabilityx.Normalize(obs, nil)
	workload = normalizeWorkload(workload)
	mode := runMode(pool)
	startedAt := time.Now()

	if run == nil || values.IsEmpty() {
		err := contextErr(ctx)
		recordBatchRunMetrics(ctx, obs, workload, mode, startedAt, err)
		return err
	}

	recordBatchItems(ctx, obs, workload, mode, values.Len())

	var err error
	if pool == nil {
		err = runListSerial[T](ctx, obs, workload, mode, values, run)
	} else {
		err = runListParallel[T](ctx, obs, workload, mode, pool, values, run)
	}
	recordBatchRunMetrics(ctx, obs, workload, mode, startedAt, err)
	return err
}

func runListParallel[T any](
	ctx context.Context,
	obs observabilityx.Observability,
	workload string,
	mode string,
	pool *ants.Pool,
	values collectionx.List[T],
	run func(context.Context, T) error,
) error {
	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	state := newRunState(cancel)
	scheduleRunList[T](workerCtx, obs, workload, mode, pool, values, run, state)
	state.Wait()
	return pickRunErr(ctx, state.Err())
}

func runListSerial[T any](
	ctx context.Context,
	obs observabilityx.Observability,
	workload string,
	mode string,
	values collectionx.List[T],
	run func(context.Context, T) error,
) error {
	var runErr error
	values.Range(func(_ int, value T) bool {
		startedAt := time.Now()
		runErr = run(ctx, value)
		recordTaskRunMetrics(ctx, obs, workload, mode, startedAt, runErr)
		return runErr == nil
	})
	return pickRunErr(ctx, runErr)
}

func pickRunErr(ctx context.Context, runErr error) error {
	if runErr != nil {
		return runErr
	}
	return contextErr(ctx)
}

type runState struct {
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	errOnce sync.Once
	runErr  error
}

func newRunState(cancel context.CancelFunc) *runState {
	return &runState{cancel: cancel}
}

func (s *runState) SetErr(err error) {
	if err == nil {
		return
	}
	s.errOnce.Do(func() {
		s.runErr = err
		if s.cancel != nil {
			s.cancel()
		}
	})
}

func (s *runState) Wait() {
	s.wg.Wait()
}

func (s *runState) Err() error {
	if s == nil {
		return nil
	}
	return s.runErr
}

func scheduleRunList[T any](
	ctx context.Context,
	obs observabilityx.Observability,
	workload string,
	mode string,
	pool *ants.Pool,
	values collectionx.List[T],
	run func(context.Context, T) error,
	state *runState,
) {
	values.Range(func(_ int, value T) bool {
		if err := contextErr(ctx); err != nil {
			state.SetErr(err)
			return false
		}
		return submitRunTask(ctx, obs, workload, mode, pool, value, run, state)
	})
}

func submitRunTask[T any](
	ctx context.Context,
	obs observabilityx.Observability,
	workload string,
	mode string,
	pool *ants.Pool,
	value T,
	run func(context.Context, T) error,
	state *runState,
) bool {
	state.wg.Add(1)
	if err := pool.Submit(func() {
		defer state.wg.Done()
		startedAt := time.Now()
		runErr := contextErr(ctx)
		if runErr == nil {
			runErr = run(ctx, value)
		}
		recordTaskRunMetrics(ctx, obs, workload, mode, startedAt, runErr)
		state.SetErr(runErr)
	}); err != nil {
		state.wg.Done()
		recordTaskSubmission(ctx, obs, workload, false)
		state.SetErr(fmt.Errorf("submit worker pool task: %w", err))
		return false
	}
	recordTaskSubmission(ctx, obs, workload, true)
	return true
}

func normalizeWorkload(workload string) string {
	workload = strings.TrimSpace(workload)
	if workload == "" {
		return "unknown"
	}
	return workload
}

func runMode(pool *ants.Pool) string {
	if pool == nil {
		return "serial"
	}
	return "parallel"
}

func workerpoolAttrs(workload, mode string) []observabilityx.Attribute {
	return []observabilityx.Attribute{
		observabilityx.String("workload", normalizeWorkload(workload)),
		observabilityx.String("mode", strings.TrimSpace(mode)),
	}
}

func recordBatchItems(ctx context.Context, obs observabilityx.Observability, workload, mode string, count int) {
	if count <= 0 {
		return
	}
	obs.Counter(workerpoolBatchItemsTotalSpec).Add(ctx, int64(count), workerpoolAttrs(workload, mode)...)
}

func recordBatchRunMetrics(
	ctx context.Context,
	obs observabilityx.Observability,
	workload string,
	mode string,
	startedAt time.Time,
	err error,
) {
	attrs := append(workerpoolAttrs(workload, mode), observabilityx.String("result", metricResult(err)))
	obs.Counter(workerpoolBatchRunsTotalSpec).Add(ctx, 1, attrs...)
	obs.Histogram(workerpoolBatchDurationSpec).Record(ctx, time.Since(startedAt).Seconds(), attrs...)
}

func recordTaskRunMetrics(
	ctx context.Context,
	obs observabilityx.Observability,
	workload string,
	mode string,
	startedAt time.Time,
	err error,
) {
	attrs := append(workerpoolAttrs(workload, mode), observabilityx.String("result", metricResult(err)))
	obs.Counter(workerpoolTaskRunsTotalSpec).Add(ctx, 1, attrs...)
	obs.Histogram(workerpoolTaskDurationSpec).Record(ctx, time.Since(startedAt).Seconds(), attrs...)
}

func recordTaskSubmission(ctx context.Context, obs observabilityx.Observability, workload string, submitted bool) {
	result := "rejected"
	if submitted {
		result = "submitted"
	}
	obs.Counter(workerpoolTaskSubmissionsTotalSpec).Add(ctx, 1,
		observabilityx.String("workload", normalizeWorkload(workload)),
		observabilityx.String("result", result),
	)
}

func metricResult(err error) string {
	if err == nil {
		return "ok"
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "cancelled"
	}
	return "error"
}

func contextErr(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("context done: %w", ctx.Err())
	default:
		return nil
	}
}
