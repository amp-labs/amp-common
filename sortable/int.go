package sortable

// Int is a sortable wrapper type for the built-in int type.
// It implements the Sortable[Int] interface, allowing integers to be used
// as keys in sorted data structures like red-black tree sets and maps.
//
// Example:
//
//	set := set.NewRedBlackTreeSet[sortable.Int]()
//	set.Add(sortable.Int(5))
//	set.Add(sortable.Int(3))
//	set.Add(sortable.Int(7))
//	// Iterating yields: 3, 5, 7 (sorted order)
//
// To convert back to a regular int, use a type conversion:
//
//	var s sortable.Int = 42
//	regularInt := int(s)
type Int int

// Compile-time check that Int implements Sortable[Int].
var _ Sortable[Int] = (*Int)(nil)

// Equals returns true if this Int has the same value as the other Int.
func (i Int) Equals(other Int) bool {
	return int(i) == int(other)
}

// LessThan returns true if this Int is numerically less than the other Int.
func (i Int) LessThan(other Int) bool {
	return int(i) < int(other)
}
