package envutil

// Option is a function which modifies a Reader. It's used by
// functions like String and Bool so that the caller can easily
// provide defaults, missing errors, fallbacks, and validation.
type Option[T any] func(Reader[T]) Reader[T]

// Default allows you to provide a default value for the Reader.
func Default[T any](dfl T) Option[T] {
	return func(rdr Reader[T]) Reader[T] {
		return rdr.WithDefault(dfl)
	}
}

// IfMissing allows you to provide an error to return if the
// Reader is missing a value.
func IfMissing[T any](err error) Option[T] {
	return func(rdr Reader[T]) Reader[T] {
		return rdr.WithErrorIfMissing(err)
	}
}

// Fallback allows you to provide a fallback Reader to use if
// the Reader is missing a value.
func Fallback[T any](f Reader[T]) Option[T] {
	return func(rdr Reader[T]) Reader[T] {
		return rdr.WithFallback(f)
	}
}

// Validate allows you to provide a validation function to run
// on the Reader's value. If the validation function returns an
// error, the Reader will return that error.
func Validate[T any](f func(T) error) Option[T] {
	return func(rdr Reader[T]) Reader[T] {
		return rdr.Map(func(val T) (T, error) {
			err := f(val)

			return val, err
		})
	}
}
