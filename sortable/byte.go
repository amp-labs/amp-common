package sortable

// Byte is a sortable wrapper type for the built-in byte type.
// It implements the Sortable[Byte] interface, allowing bytes to be used
// as keys in sorted data structures like red-black tree sets and maps.
//
// Example:
//
//	set := set.NewRedBlackTreeSet[sortable.Byte]()
//	set.Add(sortable.Byte('c'))
//	set.Add(sortable.Byte('a'))
//	set.Add(sortable.Byte('b'))
//	// Iterating yields: 'a', 'b', 'c' (sorted order)
//
// To convert back to a regular byte, use a type conversion:
//
//	var s sortable.Byte = 'x'
//	regularByte := byte(s)
type Byte byte

// Compile-time check that Byte implements Sortable[Byte].
var _ Sortable[Byte] = (*Byte)(nil)

// Equals returns true if this Byte has the same value as the other Byte.
func (b Byte) Equals(other Byte) bool {
	return byte(b) == byte(other)
}

// LessThan returns true if this Byte is numerically less than the other Byte.
func (b Byte) LessThan(other Byte) bool {
	return byte(b) < byte(other)
}
