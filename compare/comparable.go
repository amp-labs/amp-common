package compare

type Comparable[T any] interface {
	Equals(other T) bool
}

func Equals[T any](a Comparable[T], b T) bool {
	return a.Equals(b)
}
