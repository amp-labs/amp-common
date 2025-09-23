package utils

// CloseChannelIgnorePanic closes a channel like normal.
// However, if the channel has already been closed,
// it will suppress the resulting panic.
func CloseChannelIgnorePanic[T any](ch chan<- T) {
	if ch == nil {
		return
	}

	defer func() {
		// Recover from panic if the channel is already closed
		_ = recover()
	}()

	close(ch)
}
