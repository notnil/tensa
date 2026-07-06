package concurx

// Drain consumes available elements from the provided channel until no additional
// element is immediately available (i.e., the channel is drained).
//
// The function returns the last value read from the channel. If no value is available,
// it returns the zero value for type T.
//
// This helper is useful in scenarios where you want to non-blockingly clear any residual
// data from a channel without waiting indefinitely for new values. It is more efficient
// than time-based approaches as it doesn't introduce artificial delays.
func Drain[T any](ch <-chan T) T {
	var v T
	for {
		select {
		case val := <-ch:
			v = val
		default:
			return v
		}
	}
}
