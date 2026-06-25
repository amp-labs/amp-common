// Package actor provides an implementation of the actor model for concurrent message processing.
// Actors are concurrent entities that process messages sequentially through a mailbox (inbox channel).
// Each actor can handle requests and optionally return responses, with built-in panic recovery and
// Prometheus metrics integration for monitoring.
package actor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/amp-labs/amp-common/channels"
	"github.com/amp-labs/amp-common/logger"
	"github.com/amp-labs/amp-common/try"
)

const (
	// actorMetricsTickerTime is the interval at which actor metrics are updated.
	actorMetricsTickerTime = 10 * time.Second
	// actorPanicReturnTimeout is the maximum time to wait when returning panic errors to callers.
	actorPanicReturnTimeout = 5 * time.Second
)

var (
	// ErrDeadActor is returned when attempting to interact with a stopped actor.
	ErrDeadActor = errors.New("actor is dead")
	// ErrActorPanic is returned when an actor's processor panics during message processing.
	ErrActorPanic = errors.New("panic in actor")
)

// Actor is a concurrent entity that processes messages of type Request and produces responses of type Response.
// Actors are created using New and started with Run. Messages are processed sequentially through a mailbox.
type Actor[Request, Response any] struct {
	factory func(ref *Ref[Request, Response]) Processor[Request, Response]
}

// New creates a new Actor with the given processor factory function.
// The factory is called when the actor is started via Run, receiving a reference to the actor
// which can be used to interact with other actors or itself.
func New[Request, Response any](
	processorFactory func(ref *Ref[Request, Response]) Processor[Request, Response],
) *Actor[Request, Response] {
	return &Actor[Request, Response]{
		factory: processorFactory,
	}
}

// getPanicErr wraps a panic value into an error, preserving the original error if possible.
func getPanicErr(name string, err any) error {
	if e, ok := err.(error); ok {
		return fmt.Errorf("%w %s: %w", ErrActorPanic, name, e)
	}

	return fmt.Errorf("%w %s: %v", ErrActorPanic, name, err)
}

// informCallerOfPanic attempts to send a panic error to a message's response channel if one exists.
// It uses a timeout to avoid blocking indefinitely if the caller has stopped listening.
func informCallerOfPanic[Request, Response any](
	ctx context.Context,
	name string,
	msg Message[Request, Response],
	err any,
) {
	if msg.ResponseChan == nil {
		return
	}

	timer := time.NewTimer(actorPanicReturnTimeout)

	defer func() {
		// Ignore this panic, it means that the channel was closed,
		// which is perfectly understandable and valid. No need to
		// take further action.
		_ = recover()

		// Stop the timer to prevent resource leaks.
		timer.Stop()
	}()

	rsp := try.Try[Response]{
		Error: getPanicErr(name, err),
	}

	// We wait for 1 of the following to happen:
	select {
	case <-ctx.Done():
		// Context is done, do not send the error to the channel.
	case msg.ResponseChan <- rsp: // might panic
		// Successfully sent the error to the channel.
	case <-timer.C: // Timed out waiting to send the error to the channel.
	}

	// Close the channel
	channels.CloseChannelIgnorePanic(msg.ResponseChan)
}

// runProcessor executes the processor's Process method with panic recovery.
// If a panic occurs, it logs the error with stack trace, updates metrics, and notifies the caller.
func (a *Actor[Request, Response]) runProcessor(
	ctx context.Context,
	proc Processor[Request, Response],
	msg Message[Request, Response],
	name string,
) {
	defer func() {
		if err := recover(); err != nil {
			log := logger.Get(logger.WithSlackNotification(ctx))
			subsystem := logger.GetSubsystem(ctx)

			actorPanic.WithLabelValues(subsystem, name).Inc()

			log.Error("actor recovered from panic",
				"actor", name,
				"request", msg.Request,
				"error", err,
				"stack", string(debug.Stack()))

			informCallerOfPanic(ctx, name, msg, err)
		}
	}()

	proc.Process(msg)
}

// Run starts the actor and returns a reference that can be used to send messages to it.
// The name parameter is used for logging and metrics. The depth parameter specifies the mailbox
// buffer size (0 for unbuffered). Messages are delivered in FIFO order. The actor runs until the
// context is canceled or Stop is called on the returned reference.
func (a *Actor[Request, Response]) Run(ctx context.Context, name string, depth int) *Ref[Request, Response] {
	w, r, count := channels.Create[Message[Request, Response]](depth)

	return a.run(ctx, name, w, r, count, depth > 0)
}

// RunPriority starts the actor with a priority-ordered inbox instead of a FIFO one and returns a
// reference for sending messages. Messages submitted with a higher Weight are processed first;
// messages of equal Weight are processed in submission (FIFO) order. Use SendWithWeight,
// SendCtxWithWeight, RequestWithWeight, or RequestCtxWithWeight to set the weight, or set the Weight
// field directly when using Publish/PublishCtx.
//
// maxSize bounds the inbox: when maxSize <= 0 the inbox is unbounded (heap-backed, mirroring
// channels.InfiniteChan), so submitting never blocks on a full mailbox. When maxSize > 0, once that
// many messages are queued, submitting blocks until the actor drains one — applying backpressure
// (SendCtx/RequestCtx honor their context while blocked). Use Run instead for bounded FIFO delivery.
//
// Stopping via Stop drains and processes any queued messages in priority order before the run loop
// exits; canceling ctx stops immediately and discards queued messages.
func (a *Actor[Request, Response]) RunPriority(ctx context.Context, name string, maxSize int) *Ref[Request, Response] {
	w, r, count := channels.CreatePriority(ctx, maxSize, func(x, y Message[Request, Response]) bool {
		return x.Weight > y.Weight
	})

	return a.run(ctx, name, w, r, count, true)
}

// run wires up a Ref around the given inbox channels and starts the actor's processing goroutine.
// It backs both Run (FIFO inbox) and RunPriority (heap inbox). trackQueue controls whether the
// enqueued-messages gauge is reported (meaningful only when the inbox actually buffers).
func (a *Actor[Request, Response]) run(
	ctx context.Context,
	name string,
	w chan<- Message[Request, Response],
	r <-chan Message[Request, Response],
	count func() int,
	trackQueue bool,
) *Ref[Request, Response] {
	ref := &Ref[Request, Response]{
		inboxRead:  r,
		inboxWrite: w,
		getCount:   count,
		name:       name,
	}

	ref.wg.Add(1)

	proc := a.factory(ref)

	ticker := time.NewTicker(actorMetricsTickerTime)

	subsystem := logger.GetSubsystem(ctx)

	processedMessages.WithLabelValues(subsystem, name).Add(0)
	enqueuedMessages.WithLabelValues(subsystem, name).Set(0)
	actorPanic.WithLabelValues(subsystem, name).Add(0)
	aliveActors.WithLabelValues(subsystem, name).Inc()
	actorIdle.WithLabelValues(subsystem, name).Add(0)
	actorBusy.WithLabelValues(subsystem, name).Add(0)

	go func() {
		wasBusy := false

		actorStarted.Inc()

		defer ref.wg.Done()
		defer ticker.Stop()
		defer aliveActors.WithLabelValues(subsystem, name).Dec()
		defer actorStopped.Inc()
		defer func() {
			if wasBusy {
				actorBusy.WithLabelValues(subsystem, name).Dec()
			}
		}()

		for {
			actorIdle.WithLabelValues(subsystem, name).Inc()

			if wasBusy {
				actorBusy.WithLabelValues(subsystem, name).Dec()
			}

			select {
			case <-ctx.Done():
				actorIdle.WithLabelValues(subsystem, name).Dec()
				actorBusy.WithLabelValues(subsystem, name).Inc()

				wasBusy = true

				// Flip dead before closing the inbox so concurrent senders
				// see Alive() == false instead of racing with the close.
				// CloseChannelIgnorePanic recovers if Stop() got here first.
				ref.dead.Store(true)
				channels.CloseChannelIgnorePanic(ref.inboxWrite)
			case <-ticker.C:
				actorIdle.WithLabelValues(subsystem, name).Dec()
				actorBusy.WithLabelValues(subsystem, name).Inc()

				wasBusy = true

				if trackQueue {
					enqueuedMessages.WithLabelValues(subsystem, name).Set(float64(ref.getCount()))
				}
			case msg, ok := <-ref.inboxRead:
				actorIdle.WithLabelValues(subsystem, name).Dec()
				actorBusy.WithLabelValues(subsystem, name).Inc()

				wasBusy = true

				if !ok {
					return
				}

				start := time.Now()

				a.runProcessor(ctx, proc, msg, name)

				end := time.Now()

				processedMessages.WithLabelValues(subsystem, name).Inc()
				processingTime.WithLabelValues(subsystem, name).Observe(end.Sub(start).Seconds())
			}
		}
	}()

	return ref
}

// Ref is a reference to a running actor. It provides methods to send messages,
// make requests, and control the actor's lifecycle.
type Ref[Request, Response any] struct {
	wg         sync.WaitGroup
	inboxRead  <-chan Message[Request, Response]
	inboxWrite chan<- Message[Request, Response]
	getCount   func() int
	dead       atomic.Bool
	name       string
}

// Name returns the actor's name.
func (r *Ref[Request, Response]) Name() string {
	return r.name
}

// Alive returns true if the actor is still running.
func (r *Ref[Request, Response]) Alive() bool {
	return !r.dead.Load()
}

// Stop signals the actor to shut down by closing its inbox channel.
// It is safe to call multiple times.
func (r *Ref[Request, Response]) Stop() {
	if !r.dead.CompareAndSwap(false, true) {
		return
	}

	channels.CloseChannelIgnorePanic(r.inboxWrite)
}

// Wait blocks until the actor has fully stopped processing messages.
func (r *Ref[Request, Response]) Wait() {
	r.wg.Wait()
}

// submit is an internal method that sends a message to the actor's inbox,
// tracking submission metrics and respecting context cancellation.
//
// The deferred recover catches "send on closed channel" when the run loop
// (or Stop) closes inboxWrite concurrently with this send: a sender can
// pass the dead-check, then have the inbox closed before the send goroutine
// completes. Recovering here lets callers see ErrDeadActor instead of
// crashing the goroutine.
func (r *Ref[Request, Response]) submit(ctx context.Context, message Message[Request, Response]) (err error) {
	if r.dead.Load() {
		return ErrDeadActor
	}

	defer func() {
		if rec := recover(); rec != nil {
			err = ErrDeadActor
		}
	}()

	subsystem := logger.GetSubsystem(ctx)

	submitCount.WithLabelValues(subsystem, r.name).Inc()

	begin := time.Now()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case r.inboxWrite <- message:
		break
	}

	end := time.Now()

	submitTime.WithLabelValues(subsystem, r.name).Observe(end.Sub(begin).Seconds())

	return nil
}

// Publish sends a complete message to the actor without waiting for a response.
// Errors are logged but not returned. Uses context.Background().
func (r *Ref[Request, Response]) Publish(message Message[Request, Response]) {
	err := r.submit(context.Background(), message)
	if err != nil {
		slog.Error("Publish: error publishing actor message", "actor", r.name, "error", err)
	}
}

// PublishCtx sends a complete message to the actor without waiting for a response.
// Errors are logged but not returned. Respects the provided context for cancellation.
func (r *Ref[Request, Response]) PublishCtx(ctx context.Context, message Message[Request, Response]) {
	err := r.submit(ctx, message)
	if err != nil {
		slog.Error("PublishCtx: error publishing actor message", "actor", r.name, "error", err)
	}
}

// Send sends a request to the actor without waiting for a response.
// This is a fire-and-forget operation. Errors are logged but not returned.
// Uses context.Background().
func (r *Ref[Request, Response]) Send(request Request) {
	r.SendWithWeight(request, 0)
}

// SendWithWeight is like Send but assigns the message the given priority weight.
// The weight is honored only when the actor was started with RunPriority (higher
// weight is processed first; equal weights keep FIFO order); Run-started actors
// ignore it. Uses context.Background().
func (r *Ref[Request, Response]) SendWithWeight(request Request, weight int) {
	err := r.submit(context.Background(), Message[Request, Response]{
		Request: request,
		Weight:  weight,
	})
	if err != nil {
		slog.Error("Send: error sending actor message", "actor", r.name, "error", err)
	}
}

// SendCtx sends a request to the actor without waiting for a response.
// This is a fire-and-forget operation. Errors are logged but not returned.
// Respects the provided context for cancellation.
func (r *Ref[Request, Response]) SendCtx(ctx context.Context, request Request) {
	r.SendCtxWithWeight(ctx, request, 0)
}

// SendCtxWithWeight is like SendCtx but assigns the message the given priority
// weight, honored only when the actor was started with RunPriority (higher
// weight first; equal weights keep FIFO order). Respects the provided context.
func (r *Ref[Request, Response]) SendCtxWithWeight(ctx context.Context, request Request, weight int) {
	err := r.submit(ctx, Message[Request, Response]{
		Request: request,
		Weight:  weight,
	})
	if err != nil {
		slog.Error("SendCtx: error sending actor message", "actor", r.name, "error", err)
	}
}

// Request sends a request to the actor and blocks until a response is received.
// Uses context.Background(). Returns ErrDeadActor if the actor is stopped.
func (r *Ref[Request, Response]) Request(request Request) (Response, error) { //nolint:ireturn
	return r.RequestWithWeight(request, 0)
}

// RequestWithWeight is like Request but assigns the message the given priority
// weight, honored only when the actor was started with RunPriority (higher
// weight first; equal weights keep FIFO order). Uses context.Background().
func (r *Ref[Request, Response]) RequestWithWeight(request Request, weight int) (Response, error) { //nolint:ireturn
	if r.dead.Load() {
		var zero Response

		return zero, ErrDeadActor
	}

	responseChan := make(chan try.Try[Response])

	err := r.submit(context.Background(), Message[Request, Response]{
		Request:      request,
		ResponseChan: responseChan,
		Weight:       weight,
	})
	if err != nil {
		channels.CloseChannelIgnorePanic(responseChan)

		var zero Response

		return zero, err
	}

	start := time.Now()

	val := <-responseChan

	end := time.Now()

	subsystem := logger.GetSubsystem(context.Background())

	receiveTime.WithLabelValues(subsystem, r.name).Observe(end.Sub(start).Seconds())

	return val.Get()
}

// RequestCtx sends a request to the actor and blocks until a response is received or the context is canceled.
// Returns ErrDeadActor if the actor is stopped, or context error if context is canceled.
func (r *Ref[Request, Response]) RequestCtx(ctx context.Context, request Request) (Response, error) { //nolint:ireturn
	return r.RequestCtxWithWeight(ctx, request, 0)
}

// RequestCtxWithWeight is like RequestCtx but assigns the message the given
// priority weight, honored only when the actor was started with RunPriority
// (higher weight first; equal weights keep FIFO order). Respects the context.
func (r *Ref[Request, Response]) RequestCtxWithWeight( //nolint:ireturn
	ctx context.Context,
	request Request,
	weight int,
) (Response, error) {
	if r.dead.Load() {
		var zero Response

		return zero, ErrDeadActor
	}

	msgChan := make(chan try.Try[Response])

	err := r.submit(ctx, Message[Request, Response]{
		Request:      request,
		ResponseChan: msgChan,
		Weight:       weight,
	})
	if err != nil {
		channels.CloseChannelIgnorePanic(msgChan)

		var zero Response

		return zero, err
	}

	start := time.Now()

	select {
	case <-ctx.Done():
		var zero Response

		return zero, ctx.Err()
	case val := <-msgChan:
		end := time.Now()

		subsystem := logger.GetSubsystem(ctx)

		receiveTime.WithLabelValues(subsystem, r.name).Observe(end.Sub(start).Seconds())

		return val.Get()
	}
}
