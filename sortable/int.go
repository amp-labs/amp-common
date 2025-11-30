package sortable

type Int int

var _ Sortable[Int] = (*Int)(nil)

func (i Int) Equals(other Int) bool {
	return int(i) == int(other)
}

func (i Int) LessThan(other Int) bool {
	return int(i) < int(other)
}
