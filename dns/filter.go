package dns

// Filter decides whether a resolved record should be kept. It lets callers
// restrict results, for example to drop private-range addresses returned by a
// public resolver.
type Filter interface {
	// Accept reports whether record (resolved for host) should be retained.
	Accept(host string, record Record) bool
}

// filterImpl adapts a plain predicate function to the Filter interface.
type filterImpl struct {
	filter func(host string, record Record) bool
}

// Accept implements [Filter] by delegating to the wrapped predicate.
func (f *filterImpl) Accept(host string, record Record) bool {
	return f.filter(host, record)
}

// newFilter wraps a predicate as a Filter. A nil predicate yields a Filter that
// accepts everything, so callers need not special-case "no filter".
func newFilter(f func(host string, record Record) bool) Filter {
	if f == nil {
		f = func(host string, record Record) bool { return true }
	}

	return &filterImpl{
		filter: f,
	}
}
