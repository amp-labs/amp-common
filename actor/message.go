package actor

import "github.com/amp-labs/amp-common/try"

// Message represents a message sent to an actor, containing a request and an optional response channel.
// If ResponseChan is nil, the message is fire-and-forget. If provided, the actor will send the
// response (or error) to this channel after processing.
type Message[Request, Response any] struct {
	// Request is the data to be processed by the actor.
	Request Request
	// ResponseChan is an optional channel for receiving the response.
	// If nil, no response is expected (fire-and-forget).
	ResponseChan chan try.Try[Response]
}
