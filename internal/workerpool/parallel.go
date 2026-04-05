package workerpool

import (
	"context"
	"fmt"
	"sync"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/panjf2000/ants/v2"
)

// RunList executes list items with the shared worker pool when available.
// It falls back to serial execution when pool is nil.
func RunList[T any](
	ctx context.Context,
	pool *ants.Pool,
	values collectionx.List[T],
	run func(context.Context, T) error,
) error {
	if run == nil || values.IsEmpty() {
		return contextErr(ctx)
	}
	if pool == nil {
		return runListSerial(ctx, values, run)
	}
	return runListParallel(ctx, pool, values, run)
}

func runListParallel[T any](
	ctx context.Context,
	pool *ants.Pool,
	values collectionx.List[T],
	run func(context.Context, T) error,
) error {
	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	state := newRunState(cancel)
	scheduleRunList(workerCtx, pool, values, run, state)
	state.Wait()
	return pickRunErr(ctx, state.Err())
}

func runListSerial[T any](
	ctx context.Context,
	values collectionx.List[T],
	run func(context.Context, T) error,
) error {
	var runErr error
	values.Range(func(_ int, value T) bool {
		runErr = run(ctx, value)
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
		return submitRunTask(ctx, pool, value, run, state)
	})
}

func submitRunTask[T any](
	ctx context.Context,
	pool *ants.Pool,
	value T,
	run func(context.Context, T) error,
	state *runState,
) bool {
	state.wg.Add(1)
	if err := pool.Submit(func() {
		defer state.wg.Done()
		if contextErr(ctx) != nil {
			return
		}
		state.SetErr(run(ctx, value))
	}); err != nil {
		state.wg.Done()
		state.SetErr(fmt.Errorf("submit worker pool task: %w", err))
		return false
	}
	return true
}

func contextErr(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("context done: %w", ctx.Err())
	default:
		return nil
	}
}
