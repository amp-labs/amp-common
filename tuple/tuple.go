//nolint:ireturn
package tuple

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

func (t Tuple2[A, B]) First() A { //nolint:ireturn
	return t.first
}

func (t Tuple2[A, B]) Second() B { //nolint:ireturn
	return t.second
}

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

func (t Tuple3[A, B, C]) First() A { //nolint:ireturn
	return t.first
}

func (t Tuple3[A, B, C]) Second() B { //nolint:ireturn
	return t.second
}

func (t Tuple3[A, B, C]) Third() C { //nolint:ireturn
	return t.third
}

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

func (t Tuple4[A, B, C, D]) First() A { //nolint:ireturn
	return t.first
}

func (t Tuple4[A, B, C, D]) Second() B { //nolint:ireturn
	return t.second
}

func (t Tuple4[A, B, C, D]) Third() C { //nolint:ireturn
	return t.third
}

func (t Tuple4[A, B, C, D]) Fourth() D { //nolint:ireturn
	return t.fourth
}

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

func (t Tuple5[A, B, C, D, E]) First() A { //nolint:ireturn
	return t.first
}

func (t Tuple5[A, B, C, D, E]) Second() B { //nolint:ireturn
	return t.second
}

func (t Tuple5[A, B, C, D, E]) Third() C { //nolint:ireturn
	return t.third
}

func (t Tuple5[A, B, C, D, E]) Fourth() D { //nolint:ireturn
	return t.fourth
}

func (t Tuple5[A, B, C, D, E]) Fifth() E { //nolint:ireturn
	return t.fifth
}

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

// Tuple6 is a type that represents a quintuple of values.
type Tuple6[A any, B any, C any, D any, E any, F any] struct {
	first  A
	second B
	third  C
	fourth D
	fifth  E
	sixth  F
}

func (t Tuple6[A, B, C, D, E, F]) First() A { //nolint:ireturn
	return t.first
}

func (t Tuple6[A, B, C, D, E, F]) Second() B { //nolint:ireturn
	return t.second
}

func (t Tuple6[A, B, C, D, E, F]) Third() C { //nolint:ireturn
	return t.third
}

func (t Tuple6[A, B, C, D, E, F]) Fourth() D { //nolint:ireturn
	return t.fourth
}

func (t Tuple6[A, B, C, D, E, F]) Fifth() E { //nolint:ireturn
	return t.fifth
}

func (t Tuple6[A, B, C, D, E, F]) Sixth() F { //nolint:ireturn
	return t.sixth
}
