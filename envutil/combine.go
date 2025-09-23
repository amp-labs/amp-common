package envutil

import (
	"errors"

	"github.com/amp-labs/amp-common/tuple"
)

// Combine2 combines 2 distinct Readers into a single Reader that contains a
// tuple.Tuple2 with the values of the original Readers. If any of the original
// Readers has an error or is missing, the resulting Reader will also have an
// error or be missing.
func Combine2[A any, B any](
	first Reader[A],
	second Reader[B],
) Reader[tuple.Tuple2[A, B]] {
	key := first.key + "+" + second.key

	if first.err != nil || second.err != nil {
		return Reader[tuple.Tuple2[A, B]]{
			key:     key,
			present: false,
			err:     errors.Join(first.err, second.err),
		}
	}

	if !first.present || !second.present {
		return Reader[tuple.Tuple2[A, B]]{
			key:     key,
			present: false,
		}
	}

	return Reader[tuple.Tuple2[A, B]]{
		present: true,
		key:     key,
		value:   tuple.NewTuple2[A, B](first.value, second.value),
	}
}

// Combine3 combines 3 distinct Readers into a single Reader that contains a
// tuple.Tuple3 with the values of the original Readers. If any of the original
// Readers has an error or is missing, the resulting Reader will also have an
// error or be missing.
func Combine3[A any, B any, C any](
	first Reader[A],
	second Reader[B],
	third Reader[C],
) Reader[tuple.Tuple3[A, B, C]] {
	key := first.key + "+" + second.key + "+" + third.key

	if first.err != nil || second.err != nil || third.err != nil {
		return Reader[tuple.Tuple3[A, B, C]]{
			key:     key,
			present: false,
			err:     errors.Join(first.err, second.err, third.err),
		}
	}

	if !first.present || !second.present || !third.present {
		return Reader[tuple.Tuple3[A, B, C]]{
			key:     key,
			present: false,
		}
	}

	return Reader[tuple.Tuple3[A, B, C]]{
		present: true,
		key:     key,
		value:   tuple.NewTuple3[A, B, C](first.value, second.value, third.value),
	}
}

// Combine4 combines 4 distinct Readers into a single Reader that contains a
// tuple.Tuple4 with the values of the original Readers. If any of the original
// Readers has an error or is missing, the resulting Reader will also have an
// error or be missing.
func Combine4[A any, B any, C any, D any](
	first Reader[A],
	second Reader[B],
	third Reader[C],
	fourth Reader[D],
) Reader[tuple.Tuple4[A, B, C, D]] {
	key := first.key + "+" + second.key + "+" + third.key + "+" + fourth.key

	if first.err != nil || second.err != nil || third.err != nil || fourth.err != nil {
		return Reader[tuple.Tuple4[A, B, C, D]]{
			key:     key,
			present: false,
			err:     errors.Join(first.err, second.err, third.err, fourth.err),
		}
	}

	if !first.present || !second.present || !third.present || !fourth.present {
		return Reader[tuple.Tuple4[A, B, C, D]]{
			key:     key,
			present: false,
		}
	}

	return Reader[tuple.Tuple4[A, B, C, D]]{
		present: true,
		key:     key,
		value:   tuple.NewTuple4(first.value, second.value, third.value, fourth.value),
	}
}

// Combine5 combines 5 distinct Readers into a single Reader that contains a
// tuple.Tuple5 with the values of the original Readers. If any of the original
// Readers has an error or is missing, the resulting Reader will also have an
// error or be missing.
func Combine5[A any, B any, C any, D any, E any](
	first Reader[A],
	second Reader[B],
	third Reader[C],
	fourth Reader[D],
	fifth Reader[E],
) Reader[tuple.Tuple5[A, B, C, D, E]] {
	key := first.key + "+" + second.key + "+" + third.key + "+" + fourth.key + "+" + fifth.key

	if first.err != nil || second.err != nil || third.err != nil || fourth.err != nil {
		return Reader[tuple.Tuple5[A, B, C, D, E]]{
			key:     key,
			present: false,
			err:     errors.Join(first.err, second.err, third.err, fourth.err),
		}
	}

	if !first.present || !second.present || !third.present || !fourth.present {
		return Reader[tuple.Tuple5[A, B, C, D, E]]{
			key:     key,
			present: false,
		}
	}

	return Reader[tuple.Tuple5[A, B, C, D, E]]{
		present: true,
		key:     key,
		value:   tuple.NewTuple5(first.value, second.value, third.value, fourth.value, fifth.value),
	}
}

// Combine6 combines 5 distinct Readers into a single Reader that contains a
// tuple.Tuple6 with the values of the original Readers. If any of the original
// Readers has an error or is missing, the resulting Reader will also have an
// error or be missing.
func Combine6[A any, B any, C any, D any, E any, F any](
	first Reader[A],
	second Reader[B],
	third Reader[C],
	fourth Reader[D],
	fifth Reader[E],
	sixth Reader[F],
) Reader[tuple.Tuple6[A, B, C, D, E, F]] {
	key := first.key + "+" + second.key + "+" + third.key + "+" + fourth.key + "+" + fifth.key

	if first.err != nil || second.err != nil || third.err != nil || fourth.err != nil {
		return Reader[tuple.Tuple6[A, B, C, D, E, F]]{
			key:     key,
			present: false,
			err:     errors.Join(first.err, second.err, third.err, fourth.err),
		}
	}

	if !first.present || !second.present || !third.present || !fourth.present {
		return Reader[tuple.Tuple6[A, B, C, D, E, F]]{
			key:     key,
			present: false,
		}
	}

	return Reader[tuple.Tuple6[A, B, C, D, E, F]]{
		present: true,
		key:     key,
		value:   tuple.NewTuple6(first.value, second.value, third.value, fourth.value, fifth.value, sixth.value),
	}
}

// Split2 splits a Reader of a tuple.Tuple2 into 2 distinct Readers. If the original
// Reader has an error or is missing, the resulting Readers will also have an
// error or be missing.
func Split2[A any, B any](
	value Reader[tuple.Tuple2[A, B]],
) (Reader[A], Reader[B]) {
	if !value.present {
		return Reader[A]{key: value.key},
			Reader[B]{key: value.key}
	}

	if value.err != nil {
		return Reader[A]{key: value.key, err: value.err},
			Reader[B]{key: value.key, err: value.err}
	}

	return Reader[A]{key: value.key, present: true, value: value.value.First()},
		Reader[B]{key: value.key, present: true, value: value.value.Second()}
}

// Split3 splits a Reader of a tuple.Tuple3 into 3 distinct Readers. If the original
// Reader has an error or is missing, the resulting Readers will also have an
// error or be missing.
func Split3[A any, B any, C any](
	value Reader[tuple.Tuple3[A, B, C]],
) (Reader[A], Reader[B], Reader[C]) {
	if !value.present {
		return Reader[A]{key: value.key},
			Reader[B]{key: value.key},
			Reader[C]{key: value.key}
	}

	if value.err != nil {
		return Reader[A]{key: value.key, err: value.err},
			Reader[B]{key: value.key, err: value.err},
			Reader[C]{key: value.key, err: value.err}
	}

	return Reader[A]{key: value.key, present: true, value: value.value.First()},
		Reader[B]{key: value.key, present: true, value: value.value.Second()},
		Reader[C]{key: value.key, present: true, value: value.value.Third()}
}

// Split4 splits a Reader of a tuple.Tuple4 into 4 distinct Readers. If the original
// Reader has an error or is missing, the resulting Readers will also have an
// error or be missing.
func Split4[A any, B any, C any, D any](
	value Reader[tuple.Tuple4[A, B, C, D]],
) (Reader[A], Reader[B], Reader[C], Reader[D]) {
	if !value.present {
		return Reader[A]{key: value.key},
			Reader[B]{key: value.key},
			Reader[C]{key: value.key},
			Reader[D]{key: value.key}
	}

	if value.err != nil {
		return Reader[A]{key: value.key, err: value.err},
			Reader[B]{key: value.key, err: value.err},
			Reader[C]{key: value.key, err: value.err},
			Reader[D]{key: value.key, err: value.err}
	}

	return Reader[A]{key: value.key, present: true, value: value.value.First()},
		Reader[B]{key: value.key, present: true, value: value.value.Second()},
		Reader[C]{key: value.key, present: true, value: value.value.Third()},
		Reader[D]{key: value.key, present: true, value: value.value.Fourth()}
}

func Split5[A any, B any, C any, D any, E any](
	value Reader[tuple.Tuple5[A, B, C, D, E]],
) (Reader[A], Reader[B], Reader[C], Reader[D], Reader[E]) {
	if !value.present {
		return Reader[A]{key: value.key},
			Reader[B]{key: value.key},
			Reader[C]{key: value.key},
			Reader[D]{key: value.key},
			Reader[E]{key: value.key}
	}

	if value.err != nil {
		return Reader[A]{key: value.key, err: value.err},
			Reader[B]{key: value.key, err: value.err},
			Reader[C]{key: value.key, err: value.err},
			Reader[D]{key: value.key, err: value.err},
			Reader[E]{key: value.key, err: value.err}
	}

	return Reader[A]{key: value.key, present: true, value: value.value.First()},
		Reader[B]{key: value.key, present: true, value: value.value.Second()},
		Reader[C]{key: value.key, present: true, value: value.value.Third()},
		Reader[D]{key: value.key, present: true, value: value.value.Fourth()},
		Reader[E]{key: value.key, present: true, value: value.value.Fifth()}
}

func Split6[A any, B any, C any, D any, E any, F any](
	value Reader[tuple.Tuple6[A, B, C, D, E, F]],
) (Reader[A], Reader[B], Reader[C], Reader[D], Reader[E], Reader[F]) {
	if !value.present {
		return Reader[A]{key: value.key},
			Reader[B]{key: value.key},
			Reader[C]{key: value.key},
			Reader[D]{key: value.key},
			Reader[E]{key: value.key},
			Reader[F]{key: value.key}
	}

	if value.err != nil {
		return Reader[A]{key: value.key, err: value.err},
			Reader[B]{key: value.key, err: value.err},
			Reader[C]{key: value.key, err: value.err},
			Reader[D]{key: value.key, err: value.err},
			Reader[E]{key: value.key, err: value.err},
			Reader[F]{key: value.key, err: value.err}
	}

	return Reader[A]{key: value.key, present: true, value: value.value.First()},
		Reader[B]{key: value.key, present: true, value: value.value.Second()},
		Reader[C]{key: value.key, present: true, value: value.value.Third()},
		Reader[D]{key: value.key, present: true, value: value.value.Fourth()},
		Reader[E]{key: value.key, present: true, value: value.value.Fifth()},
		Reader[F]{key: value.key, present: true, value: value.value.Sixth()}
}
