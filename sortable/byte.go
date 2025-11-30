package sortable

type Byte byte

var _ Sortable[Byte] = (*Byte)(nil)

func (b Byte) Equals(other Byte) bool {
	return byte(b) == byte(other)
}

func (b Byte) LessThan(other Byte) bool {
	return byte(b) < byte(other)
}
