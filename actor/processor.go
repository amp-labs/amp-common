package actor

import (
	"log/slog"

	"github.com/amp-labs/amp-common/try"
)

// Processor defines the interface for processing messages within an actor.
// Implementations must handle the message, optionally sending responses via the message's ResponseChan.
type Processor[Request, Response any] interface {
	Process(msg Message[Request, Response])
}

// processor is a simple implementation of Processor that wraps a function.
type processor[Request, Response any] struct {
	process func(Message[Request, Response])
}

func (p *processor[Request, Response]) Process(msg Message[Request, Response]) {
	p.process(msg)
}

// NewProcessor creates a Processor from a function that processes messages.
// The function is responsible for handling response channels if present in the message.
func NewProcessor[Request, Response any](processorFunc func(Message[Request, Response])) Processor[Request, Response] {
	return &processor[Request, Response]{
		process: processorFunc,
	}
}

// SimpleProcessor creates a Processor from a simple request-response function.
// It automatically handles response channel management: sending the result or error
// to the response channel if present, or logging errors for fire-and-forget messages.
func SimpleProcessor[Request, Response any](f func(req Request) (Response, error)) Processor[Request, Response] {
	processorFunc := func(msg Message[Request, Response]) {
		resp, err := f(msg.Request)
		if err != nil && msg.ResponseChan == nil { //nolint:nestif
			slog.Error("error processing message", "error", err)
		}

		if msg.ResponseChan != nil {
			msg.ResponseChan <- try.Try[Response]{
				Value: resp,
				Error: err,
			}

			close(msg.ResponseChan)
		}
	}

	return NewProcessor(processorFunc)
}
