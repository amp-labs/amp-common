package sortable

type String string

var _ Sortable[String] = (*String)(nil)

func (s String) Equals(other String) bool {
	return string(s) == string(other)
}

func (s String) LessThan(other String) bool {
	return string(s) < string(other)
}
