package actor

import (
	"log/slog"

	"github.com/amp-labs/amp-common/try"
)

type Processor[Request, Response any] interface {
	Process(msg Message[Request, Response])
}

type processor[Request, Response any] struct {
	process func(Message[Request, Response])
}

func (p *processor[Request, Response]) Process(msg Message[Request, Response]) {
	p.process(msg)
}

func NewProcessor[Request, Response any](processorFunc func(Message[Request, Response])) Processor[Request, Response] {
	return &processor[Request, Response]{
		process: processorFunc,
	}
}

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
