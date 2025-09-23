package actor

import "github.com/amp-labs/amp-common/try"

type Message[Request, Response any] struct {
	Request      Request
	ResponseChan chan try.Try[Response]
}
