package actor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type empty struct{}

func TestActorPanic(t *testing.T) {
	t.Parallel()

	// Create a new actor with a panic handler
	act := New[empty, empty](func(ref *Ref[empty, empty]) Processor[empty, empty] {
		return NewProcessor[empty, empty](func(m Message[empty, empty]) {
			panic("test panic")
		})
	})

	ref := act.Run(t.Context(), "test", 1)

	_, err := ref.RequestCtx(t.Context(), empty{})

	require.Error(t, err)
	require.ErrorContains(t, err, "test panic")
}
