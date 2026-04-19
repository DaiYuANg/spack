package metrics

import (
	"context"
	"sync"

	"github.com/DaiYuANg/arcgo/dix"
	"github.com/DaiYuANg/arcgo/observabilityx"
)

// DeferredObserver lets dix register an observer before the DI metrics backend exists.
type DeferredObserver struct {
	mu       sync.Mutex
	opts     []Option
	observer dix.Observer
	pending  []func(dix.Observer)
}

// NewDeferredObserver buffers dix events until a metrics backend is resolved.
func NewDeferredObserver(opts ...Option) *DeferredObserver {
	return &DeferredObserver{
		opts: opts,
	}
}

// Attach wires the real metrics observer and replays events captured before DI resolved metrics.
func (o *DeferredObserver) Attach(obs observabilityx.Observability) {
	if o == nil || obs == nil {
		return
	}

	o.mu.Lock()
	if o.observer == nil {
		o.observer = NewObserver(obs, o.opts...)
	}
	observer := o.observer
	pending := o.pending
	o.pending = nil
	o.mu.Unlock()

	for _, replay := range pending {
		replay(observer)
	}
}

func (o *DeferredObserver) emit(replay func(dix.Observer)) {
	if o == nil || replay == nil {
		return
	}

	o.mu.Lock()
	observer := o.observer
	if observer == nil {
		o.pending = append(o.pending, replay)
		o.mu.Unlock()
		return
	}
	o.mu.Unlock()

	replay(observer)
}

func (o *DeferredObserver) OnBuild(ctx context.Context, event dix.BuildEvent) {
	o.emit(func(observer dix.Observer) {
		observer.OnBuild(ctx, event)
	})
}

func (o *DeferredObserver) OnStart(ctx context.Context, event dix.StartEvent) {
	o.emit(func(observer dix.Observer) {
		observer.OnStart(ctx, event)
	})
}

func (o *DeferredObserver) OnStop(ctx context.Context, event dix.StopEvent) {
	o.emit(func(observer dix.Observer) {
		observer.OnStop(ctx, event)
	})
}

func (o *DeferredObserver) OnHealthCheck(ctx context.Context, event dix.HealthCheckEvent) {
	o.emit(func(observer dix.Observer) {
		observer.OnHealthCheck(ctx, event)
	})
}

func (o *DeferredObserver) OnStateTransition(ctx context.Context, event dix.StateTransitionEvent) {
	o.emit(func(observer dix.Observer) {
		observer.OnStateTransition(ctx, event)
	})
}
