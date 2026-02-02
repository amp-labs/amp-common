package empty

// T is an empty struct type that occupies zero bytes of memory.
// It's commonly used for signaling channels or as map keys where
// only the presence of a key matters, not its value.
//
// Example:
//
//	type Set map[string]empty.T
//	set := make(Set)
//	set["key"] = empty.V
type T struct{}

// V is a pre-allocated instance of the empty struct T.
// Use this to avoid repeated allocations of empty structs.
//
// Example:
//
//	signalChan := make(chan empty.T, 1)
//	signalChan <- empty.V // Send empty struct as signal
var V = T{}

// P is a pointer to the pre-allocated empty struct V.
// Use this when you need a pointer to an empty struct.
//
// Example:
//
//	var ptr *empty.T = empty.P
var P = &V

// Slice returns an empty slice of the specified type T.
// The returned slice has zero length and zero capacity.
//
// This is useful when you need to return a non-nil empty slice
// instead of nil, which can be important for JSON serialization
// or API responses.
//
// Example:
//
//	names := empty.Slice[string]()  // []string{} not nil
//	numbers := empty.Slice[int]()   // []int{} not nil
func Slice[T any]() []T {
	return []T{}
}

// SlictPtr returns a pointer to an empty slice of the specified type T.
// The returned pointer points to a slice with zero length and zero capacity.
//
// This is useful when working with APIs that expect pointers to slices,
// particularly when you want to distinguish between a nil pointer and
// a pointer to an empty slice.
//
// Example:
//
//	slicePtr := empty.SlictPtr[string]()  // *[]string pointing to []string{}
//	if slicePtr != nil {
//	    fmt.Println("Not nil, but empty:", len(*slicePtr) == 0)
//	}
func SlictPtr[T any]() *[]T {
	s := Slice[T]()

	return &s
}

// Map returns an empty initialized map with the specified key type K and value type V.
// The returned map is not nil and has zero length.
//
// This is useful when you need to return a non-nil empty map instead of nil,
// which can be important for JSON serialization or when you want to allow
// immediate insertion without nil checks.
//
// Example:
//
//	userScores := empty.Map[string, int]()  // map[string]int{} not nil
//	userScores["alice"] = 100  // Safe to use immediately
func Map[K comparable, V any]() map[K]V {
	return make(map[K]V)
}

// MapPtr returns a pointer to an empty initialized map with the specified
// key type K and value type V.
//
// This is useful when working with APIs that expect pointers to maps,
// particularly when you want to distinguish between a nil pointer and
// a pointer to an empty map.
//
// Example:
//
//	mapPtr := empty.MapPtr[string, int]()  // *map[string]int pointing to map[string]int{}
//	if mapPtr != nil {
//	    (*mapPtr)["key"] = 42  // Safe to use
//	}
func MapPtr[K comparable, V any]() *map[K]V {
	m := Map[K, V]()

	return &m
}

// Chan returns a closed, empty receive-only channel of the specified type T.
//
// This is useful for creating sentinel channels or default values in select
// statements. Since the channel is already closed, any receive operation will
// immediately return the zero value of T with ok=false.
//
// Example:
//
//	doneChan := empty.Chan[struct{}]()
//	select {
//	case <-doneChan:
//	    fmt.Println("Immediately returns because channel is closed")
//	case <-time.After(time.Second):
//	    fmt.Println("Will not reach here")
//	}
func Chan[T any]() <-chan T {
	c := make(chan T)

	close(c)

	return c
}

// ChanPtr returns a pointer to a closed, empty receive-only channel of the specified type T.
//
// This is useful when working with APIs that expect pointers to channels,
// particularly when you want to distinguish between a nil pointer and
// a pointer to a closed empty channel.
//
// Example:
//
//	chanPtr := empty.ChanPtr[int]()
//	if chanPtr != nil {
//	    val, ok := <-*chanPtr  // ok will be false, val will be 0
//	}
func ChanPtr[T any]() *<-chan T {
	c := Chan[T]()

	return &c
}

// Value returns the zero value of the specified type T.
//
// This is useful when you need to explicitly return or pass a zero value,
// particularly in generic code where you can't use a type-specific zero literal.
//
// Example:
//
//	var defaultInt int = empty.Value[int]()        // 0
//	var defaultStr string = empty.Value[string]()  // ""
//	var defaultBool bool = empty.Value[bool]()     // false
func Value[T any]() T {
	var zeroVal T

	return zeroVal
}

// Func is a no-op function that does nothing and returns nothing.
//
// This is useful as a placeholder callback or when you need to provide
// a function argument but have no operation to perform.
//
// Example:
//
//	type Handler func()
//	var handler Handler = empty.Func  // Provide a safe no-op handler
//	handler()  // Safe to call, does nothing
func Func() {}
