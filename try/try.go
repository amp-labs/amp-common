package try

type Try[A any] struct {
	Value A
	Error error
}

func (t Try[A]) IsSuccess() bool {
	return t.Error == nil
}

func (t Try[A]) IsFailure() bool {
	return t.Error != nil
}

func (t Try[A]) Get() (A, error) { //nolint:ireturn
	if t.IsFailure() {
		var zero A

		return zero, t.Error
	} else {
		return t.Value, nil
	}
}

func (t Try[A]) GetOrElse(defaultValue A) A { //nolint:ireturn
	if t.IsSuccess() {
		return t.Value
	} else {
		return defaultValue
	}
}

func Map[A, B any](t Try[A], f func(A) (B, error)) Try[B] {
	if t.IsSuccess() {
		val, err := f(t.Value)

		return Try[B]{Value: val, Error: err}
	} else {
		return Try[B]{Error: t.Error}
	}
}
