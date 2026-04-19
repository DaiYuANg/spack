package task

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/daiyuang/spack/internal/source"
)

const sourceRescanDebounce = 300 * time.Millisecond

func (r *sourceRescanRuntime) startSourceWatcher() {
	if r == nil || r.watchCancel != nil {
		return
	}

	watchCtx, cancel := context.WithCancel(context.Background())
	changes, err := r.scanner.Watch(watchCtx)
	if errors.Is(err, source.ErrWatchUnsupported) {
		cancel()
		return
	}
	if err != nil {
		cancel()
		r.logger.Warn("Task source rescan watcher unavailable", slog.String("err", err.Error()))
		return
	}

	r.watchCancel = cancel
	r.watchDone = make(chan struct{})
	go r.watchSourceChanges(watchCtx, changes)
	r.logger.Info("Task source rescan watcher enabled", slog.Duration("debounce", sourceRescanDebounce))
}

func (r *sourceRescanRuntime) watchSourceChanges(ctx context.Context, changes <-chan source.ChangeEvent) {
	defer close(r.watchDone)

	timer := time.NewTimer(sourceRescanDebounce)
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	defer timer.Stop()
	pending := false

	for {
		select {
		case <-ctx.Done():
			return
		case change, ok := <-changes:
			if !ok {
				return
			}
			r.logger.Debug("Source change detected",
				slog.String("path", change.Path),
				slog.String("op", change.Op),
			)
			pending = true
			resetTimer(timer, sourceRescanDebounce)
		case <-timer.C:
			if pending {
				pending = false
				runSourceRescan(context.Background(), r)
			}
		}
	}
}

func stopSourceRescanRuntime(ctx context.Context, runtime *sourceRescanRuntime) error {
	if runtime == nil || runtime.watchCancel == nil {
		return nil
	}

	runtime.watchCancel()
	select {
	case <-runtime.watchDone:
		runtime.watchCancel = nil
		runtime.watchDone = nil
		return nil
	case <-ctx.Done():
		return fmt.Errorf("stop source watcher: %w", ctx.Err())
	}
}

func resetTimer(timer *time.Timer, delay time.Duration) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(delay)
}
