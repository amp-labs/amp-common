package sortable

// String is a sortable wrapper type for the built-in string type.
// It implements the Sortable[String] interface, allowing strings to be used
// as keys in sorted data structures like red-black tree sets and maps.
// Strings are compared lexicographically using Go's standard string comparison.
//
// Example:
//
//	set := set.NewRedBlackTreeSet[sortable.String]()
//	set.Add(sortable.String("cherry"))
//	set.Add(sortable.String("apple"))
//	set.Add(sortable.String("banana"))
//	// Iterating yields: "apple", "banana", "cherry" (sorted order)
//
// To convert back to a regular string, use a type conversion:
//
//	var s sortable.String = "hello"
//	regularString := string(s)
type String string

// Compile-time check that String implements Sortable[String].
var _ Sortable[String] = (*String)(nil)

// Equals returns true if this String has the same value as the other String.
func (s String) Equals(other String) bool {
	return string(s) == string(other)
}

// LessThan returns true if this String is lexicographically less than the other String.
func (s String) LessThan(other String) bool {
	return string(s) < string(other)
}
