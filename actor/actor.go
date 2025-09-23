package actor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

	"github.com/amp-labs/amp-common/logger"
	"github.com/amp-labs/amp-common/try"
	"github.com/amp-labs/amp-common/utils"
)

const (
	actorMetricsTickerTime  = 10 * time.Second
	actorPanicReturnTimeout = 5 * time.Second
)

var (
	ErrDeadActor  = errors.New("actor is dead")
	ErrActorPanic = errors.New("panic in actor")
)

type Actor[Request, Response any] struct {
	factory func(ref *Ref[Request, Response]) Processor[Request, Response]
}

func New[Request, Response any](
	processorFactory func(ref *Ref[Request, Response]) Processor[Request, Response],
) *Actor[Request, Response] {
	return &Actor[Request, Response]{
		factory: processorFactory,
	}
}

func createMessageChannel[Request, Response any](size int) chan Message[Request, Response] {
	if size == 0 {
		return make(chan Message[Request, Response])
	} else {
		return make(chan Message[Request, Response], size)
	}
}

func getPanicErr(name string, err any) error {
	if e, ok := err.(error); ok {
		return fmt.Errorf("%w %s: %w", ErrActorPanic, name, e)
	}

	return fmt.Errorf("%w %s: %v", ErrActorPanic, name, err)
}

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
	utils.CloseChannelIgnorePanic(msg.ResponseChan)
}

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

func (a *Actor[Request, Response]) Run(ctx context.Context, name string, depth int) *Ref[Request, Response] {
	ref := &Ref[Request, Response]{
		inbox: createMessageChannel[Request, Response](depth),
		name:  name,
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

				// Due to a race this might already be closed.
				// If it is, we don't want to panic.
				utils.CloseChannelIgnorePanic(ref.inbox)

				ref.dead = true
			case <-ticker.C:
				actorIdle.WithLabelValues(subsystem, name).Dec()
				actorBusy.WithLabelValues(subsystem, name).Inc()

				wasBusy = true

				if depth > 0 {
					enqueuedMessages.WithLabelValues(subsystem, name).Set(float64(len(ref.inbox)))
				}
			case msg, ok := <-ref.inbox:
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

type Ref[Request, Response any] struct {
	wg    sync.WaitGroup
	inbox chan Message[Request, Response]
	dead  bool
	name  string
}

func (r *Ref[Request, Response]) Name() string {
	return r.name
}

func (r *Ref[Request, Response]) Alive() bool {
	return !r.dead
}

func (r *Ref[Request, Response]) Stop() {
	if r.dead {
		return
	}

	utils.CloseChannelIgnorePanic(r.inbox)
	r.dead = true
}

func (r *Ref[Request, Response]) Wait() {
	r.wg.Wait()
}

func (r *Ref[Request, Response]) submit(ctx context.Context, message Message[Request, Response]) error {
	if r.dead {
		return ErrDeadActor
	}

	subsystem := logger.GetSubsystem(ctx)

	submitCount.WithLabelValues(subsystem, r.name).Inc()

	begin := time.Now()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case r.inbox <- message:
		break
	}

	end := time.Now()

	submitTime.WithLabelValues(subsystem, r.name).Observe(end.Sub(begin).Seconds())

	return nil
}

func (r *Ref[Request, Response]) Publish(message Message[Request, Response]) {
	if err := r.submit(context.Background(), message); err != nil {
		slog.Error("Publish: error publishing actor message", "actor", r.name, "error", err)
	}
}

func (r *Ref[Request, Response]) PublishCtx(ctx context.Context, message Message[Request, Response]) {
	if err := r.submit(ctx, message); err != nil {
		slog.Error("PublishCtx: error publishing actor message", "actor", r.name, "error", err)
	}
}

func (r *Ref[Request, Response]) Send(request Request) {
	if err := r.submit(context.Background(), Message[Request, Response]{
		Request: request,
	}); err != nil {
		slog.Error("Send: error sending actor message", "actor", r.name, "error", err)
	}
}

func (r *Ref[Request, Response]) SendCtx(ctx context.Context, request Request) {
	if err := r.submit(ctx, Message[Request, Response]{
		Request: request,
	}); err != nil {
		slog.Error("SendCtx: error sending actor message", "actor", r.name, "error", err)
	}
}

func (r *Ref[Request, Response]) Request(request Request) (Response, error) { //nolint:ireturn
	if r.dead {
		var zero Response

		return zero, ErrDeadActor
	}

	ch := make(chan try.Try[Response])

	if err := r.submit(context.Background(), Message[Request, Response]{
		Request:      request,
		ResponseChan: ch,
	}); err != nil {
		utils.CloseChannelIgnorePanic(ch)

		var zero Response

		return zero, err
	}

	start := time.Now()

	val := <-ch

	end := time.Now()

	subsystem := logger.GetSubsystem(context.Background())

	receiveTime.WithLabelValues(subsystem, r.name).Observe(end.Sub(start).Seconds())

	return val.Get()
}

func (r *Ref[Request, Response]) RequestCtx(ctx context.Context, request Request) (Response, error) { //nolint:ireturn
	if r.dead {
		var zero Response

		return zero, ErrDeadActor
	}

	msgChan := make(chan try.Try[Response])

	if err := r.submit(ctx, Message[Request, Response]{
		Request:      request,
		ResponseChan: msgChan,
	}); err != nil {
		utils.CloseChannelIgnorePanic(msgChan)

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
