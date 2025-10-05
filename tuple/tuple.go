//nolint:ireturn

// Package tuple provides generic tuple types for grouping multiple values together.
// Tuples are immutable value types with type-safe accessors.
package tuple

// NewTuple2 creates a new 2-element tuple.
func NewTuple2[A, B any](first A, second B) Tuple2[A, B] {
	return Tuple2[A, B]{
		first:  first,
		second: second,
	}
}

// Tuple2 is a type that represents a pair of values.
type Tuple2[A any, B any] struct {
	first  A
	second B
}

// First returns the first element of the tuple.
func (t Tuple2[A, B]) First() A { //nolint:ireturn
	return t.first
}

// Second returns the second element of the tuple.
func (t Tuple2[A, B]) Second() B { //nolint:ireturn
	return t.second
}

// NewTuple3 creates a new 3-element tuple.
func NewTuple3[A, B, C any](first A, second B, third C) Tuple3[A, B, C] {
	return Tuple3[A, B, C]{
		first:  first,
		second: second,
		third:  third,
	}
}

// Tuple3 is a type that represents a triple of values.
type Tuple3[A any, B any, C any] struct {
	first  A
	second B
	third  C
}

// First returns the first element of the tuple.
func (t Tuple3[A, B, C]) First() A { //nolint:ireturn
	return t.first
}

// Second returns the second element of the tuple.
func (t Tuple3[A, B, C]) Second() B { //nolint:ireturn
	return t.second
}

// Third returns the third element of the tuple.
func (t Tuple3[A, B, C]) Third() C { //nolint:ireturn
	return t.third
}

// NewTuple4 creates a new 4-element tuple.
func NewTuple4[A, B, C, D any](first A, second B, third C, fourth D) Tuple4[A, B, C, D] {
	return Tuple4[A, B, C, D]{
		first:  first,
		second: second,
		third:  third,
		fourth: fourth,
	}
}

// Tuple4 is a type that represents a quadruple of values.
type Tuple4[A any, B any, C any, D any] struct {
	first  A
	second B
	third  C
	fourth D
}

// First returns the first element of the tuple.
func (t Tuple4[A, B, C, D]) First() A { //nolint:ireturn
	return t.first
}

// Second returns the second element of the tuple.
func (t Tuple4[A, B, C, D]) Second() B { //nolint:ireturn
	return t.second
}

// Third returns the third element of the tuple.
func (t Tuple4[A, B, C, D]) Third() C { //nolint:ireturn
	return t.third
}

// Fourth returns the fourth element of the tuple.
func (t Tuple4[A, B, C, D]) Fourth() D { //nolint:ireturn
	return t.fourth
}

// NewTuple5 creates a new 5-element tuple.
func NewTuple5[A, B, C, D, E any](first A, second B, third C, fourth D, fifth E) Tuple5[A, B, C, D, E] {
	return Tuple5[A, B, C, D, E]{
		first:  first,
		second: second,
		third:  third,
		fourth: fourth,
		fifth:  fifth,
	}
}

// Tuple5 is a type that represents a quintuple of values.
type Tuple5[A any, B any, C any, D any, E any] struct {
	first  A
	second B
	third  C
	fourth D
	fifth  E
}

// First returns the first element of the tuple.
func (t Tuple5[A, B, C, D, E]) First() A { //nolint:ireturn
	return t.first
}

// Second returns the second element of the tuple.
func (t Tuple5[A, B, C, D, E]) Second() B { //nolint:ireturn
	return t.second
}

// Third returns the third element of the tuple.
func (t Tuple5[A, B, C, D, E]) Third() C { //nolint:ireturn
	return t.third
}

// Fourth returns the fourth element of the tuple.
func (t Tuple5[A, B, C, D, E]) Fourth() D { //nolint:ireturn
	return t.fourth
}

// Fifth returns the fifth element of the tuple.
func (t Tuple5[A, B, C, D, E]) Fifth() E { //nolint:ireturn
	return t.fifth
}

// NewTuple6 creates a new 6-element tuple.
func NewTuple6[A, B, C, D, E, F any](first A, second B, third C, fourth D, fifth E, sixth F) Tuple6[A, B, C, D, E, F] {
	return Tuple6[A, B, C, D, E, F]{
		first:  first,
		second: second,
		third:  third,
		fourth: fourth,
		fifth:  fifth,
		sixth:  sixth,
	}
}

// Tuple6 is a type that represents a sextuple of values.
type Tuple6[A any, B any, C any, D any, E any, F any] struct {
	first  A
	second B
	third  C
	fourth D
	fifth  E
	sixth  F
}

// First returns the first element of the tuple.
func (t Tuple6[A, B, C, D, E, F]) First() A { //nolint:ireturn
	return t.first
}

// Second returns the second element of the tuple.
func (t Tuple6[A, B, C, D, E, F]) Second() B { //nolint:ireturn
	return t.second
}

// Third returns the third element of the tuple.
func (t Tuple6[A, B, C, D, E, F]) Third() C { //nolint:ireturn
	return t.third
}

// Fourth returns the fourth element of the tuple.
func (t Tuple6[A, B, C, D, E, F]) Fourth() D { //nolint:ireturn
	return t.fourth
}

// Fifth returns the fifth element of the tuple.
func (t Tuple6[A, B, C, D, E, F]) Fifth() E { //nolint:ireturn
	return t.fifth
}

// Sixth returns the sixth element of the tuple.
func (t Tuple6[A, B, C, D, E, F]) Sixth() F { //nolint:ireturn
	return t.sixth
}
