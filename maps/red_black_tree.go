// Package maps provides a red-black tree implementation of the Map interface.
// This file contains redBlackTreeMap, a self-balancing binary search tree that
// maintains sorted key-value pairs with guaranteed O(log n) performance for
// insertions, deletions, and lookups.
//
// Red-black trees enforce the following properties to maintain balance:
//  1. Every node is either red or black
//  2. The root is always black
//  3. All leaves (nil nodes) are considered black
//  4. Red nodes cannot have red children (no two consecutive red nodes on any path)
//  5. Every path from root to leaf contains the same number of black nodes
//
// These properties ensure the tree remains approximately balanced, preventing
// the worst-case O(n) behavior of unbalanced binary search trees.
package maps

import (
	"fmt"
	"iter"

	"github.com/amp-labs/amp-common/hashing"
	"github.com/amp-labs/amp-common/optional"
	"github.com/amp-labs/amp-common/set"
	"github.com/amp-labs/amp-common/sortable"
	"github.com/amp-labs/amp-common/zero"
)

// visitor defines an interface for traversing red-black tree nodes.
// Implementations can perform custom operations during tree traversal
// (e.g., counting nodes, collecting keys, or checking predicates).
// Visit returns false to stop traversal early.
type visitor[K sortable.Sortable[K], V any] interface {
	Visit(node *rbtNode[K, V]) bool
}

// color represents the color of a red-black tree node.
// Red-black trees use node colors to maintain balance during insertions and deletions.
type color bool

// direction indicates the relationship of a node to its parent (left child, right child, or root).
type direction byte

// String returns a human-readable representation of the node color.
func (c color) String() string {
	switch c {
	case true:
		return "Black"
	default:
		return "Red"
	}
}

// String returns a human-readable representation of the direction.
func (d direction) String() string {
	switch d {
	case left:
		return "left"
	case right:
		return "right"
	case nodir:
		return "center"
	default:
		return "not recognized"
	}
}

const (
	// black and red are the two node colors in a red-black tree.
	// Black is represented as true for a default zero-value of black when nodes are created.
	black, red color = true, false

	// left, right, and nodir represent the position of a node relative to its parent.
	// nodir is used for the root node which has no parent.
	left direction = iota
	right
	nodir
)

// rbtNode represents a single node in the red-black tree.
// Each node stores a key-value pair, maintains pointers to its children and parent,
// and tracks its color for tree balancing.
type rbtNode[K sortable.Sortable[K], V any] struct {
	key    K
	value  V
	color  color
	left   *rbtNode[K, V]
	right  *rbtNode[K, V]
	parent *rbtNode[K, V]
}

// String returns a string representation of the node showing its key and color.
func (n *rbtNode[K, V]) String() string {
	return fmt.Sprintf("(%#v : %s)", n.key, n.Color())
}

// Parent returns the parent node of this node.
func (n *rbtNode[K, V]) Parent() *rbtNode[K, V] {
	return n.parent
}

// SetColor sets the color of this node.
func (n *rbtNode[K, V]) SetColor(color color) {
	n.color = color
}

// Color returns the color of this node.
func (n *rbtNode[K, V]) Color() color {
	return n.color
}

// redBlackTreeMap is a self-balancing binary search tree implementation of the Map interface.
// It maintains O(log n) performance for insertions, deletions, and lookups by enforcing
// red-black tree properties:
//  1. Every node is either red or black
//  2. The root is black
//  3. All leaves (nil nodes) are black
//  4. Red nodes cannot have red children
//  5. Every path from root to leaf contains the same number of black nodes
type redBlackTreeMap[K sortable.Sortable[K], V any] struct {
	root *rbtNode[K, V]
}

// getParent finds the parent node where a key either exists or should be inserted.
// Returns (true, parent, direction) if key exists, (false, parent, direction) if not.
// The direction indicates whether the key is/should be a left or right child of parent.
func (t *redBlackTreeMap[K, V]) getParent(key K) (found bool, parent *rbtNode[K, V], dir direction) {
	if t.root == nil {
		return false, nil, nodir
	}

	return t.internalLookup(nil, t.root, key, nodir)
}

// internalLookup recursively searches the tree for a key, tracking the parent node
// and direction at each step. This is used for both lookups and insertions.
func (t *redBlackTreeMap[K, V]) internalLookup(
	parent *rbtNode[K, V], this *rbtNode[K, V], key K, dir direction,
) (bool, *rbtNode[K, V], direction) {
	switch {
	case this == nil:
		return false, parent, dir
	case key.Equals(this.key):
		return true, parent, dir
	case key.LessThan(this.key):
		return t.internalLookup(this, this.left, key, left)
	default:
		return t.internalLookup(this, this.right, key, right)
	}
}

// getNode retrieves the node containing the specified key.
// Returns the node and true if found, nil and false otherwise.
func (t *redBlackTreeMap[K, V]) getNode(key K) (*rbtNode[K, V], bool) {
	found, parent, dir := t.getParent(key)
	if found {
		if parent == nil {
			return t.root, true
		} else {
			var node *rbtNode[K, V]

			switch dir {
			case left:
				node = parent.left
			case right:
				node = parent.right
			case nodir:
				node = nil
			}

			if node != nil {
				return node, true
			}
		}
	}

	return nil, false
}

// rotateRight performs a right rotation around node y.
// This is a fundamental operation for rebalancing the tree:
//
//	    y              x
//	   / \            / \
//	  x   C   =>     A   y
//	 / \                / \
//	A   B              B   C
//
// nolint:dupword,varnamelen // ASCII art; standard RB tree variable names
func (t *redBlackTreeMap[K, V]) rotateRight(y *rbtNode[K, V]) {
	if y == nil {
		return
	}

	if y.left == nil {
		return
	}

	x := y.left //nolint:varnamelen // Standard red-black tree variable names from CLRS
	y.left = x.right

	if x.right != nil {
		x.right.parent = y
	}

	x.parent = y.parent

	if y.parent == nil {
		t.root = x
	} else {
		if y == y.parent.left {
			y.parent.left = x
		} else {
			y.parent.right = x
		}
	}

	x.right = y
	y.parent = x
}

// rotateLeft performs a left rotation around node x.
// This is a fundamental operation for rebalancing the tree:
//
//	  x                y
//	 / \              / \
//	A   y      =>    x   C
//	   / \          / \
//	  B   C        A   B
//
// nolint:varnamelen // Standard red-black tree variable names
func (t *redBlackTreeMap[K, V]) rotateLeft(x *rbtNode[K, V]) {
	if x == nil {
		return
	}

	if x.right == nil {
		return
	}

	y := x.right //nolint:varnamelen // Standard red-black tree variable names from CLRS
	x.right = y.left

	if y.left != nil {
		y.left.parent = x
	}

	y.parent = x.parent

	if x.parent == nil {
		t.root = y
	} else {
		if x == x.parent.left {
			x.parent.left = y
		} else {
			x.parent.right = y
		}
	}

	y.left = x
	x.parent = y
}

// Get retrieves the value associated with the given key.
// Returns (value, true, nil) if found, (zero, false, nil) if not found.
func (t *redBlackTreeMap[K, V]) Get(key K) (value V, found bool, err error) {
	node, ok := t.getNode(key)
	if ok {
		return node.value, true, nil
	} else {
		return zero.Value[V](), false, nil
	}
}

// GetOrElse retrieves the value for the given key, or returns defaultValue if not found.
func (t *redBlackTreeMap[K, V]) GetOrElse(key K, defaultValue V) (value V, err error) {
	value, found, err := t.Get(key)
	if err != nil {
		return zero.Value[V](), err
	}

	if found {
		return value, nil
	}

	return defaultValue, nil
}

// Add inserts or updates a key-value pair in the map.
// If the key already exists, its value is updated.
// After insertion, the tree is rebalanced to maintain red-black properties.
func (t *redBlackTreeMap[K, V]) Add(key K, value V) error {
	if t.root == nil {
		t.root = &rbtNode[K, V]{key: key, color: black, value: value}

		return nil
	}

	found, parent, dir := t.internalLookup(nil, t.root, key, nodir)
	if found {
		if parent == nil {
			t.root.value = value
		} else {
			switch dir {
			case left:
				parent.left.value = value
			case right:
				parent.right.value = value
			case nodir:
			}
		}
	} else {
		if parent != nil {
			newNode := &rbtNode[K, V]{key: key, parent: parent, value: value}

			switch dir {
			case left:
				parent.left = newNode
			case right:
				parent.right = newNode
			case nodir:
			}

			t.fixupPut(newNode)
		}
	}

	return nil
}

// Remove deletes the key-value pair with the given key from the map.
// After deletion, the tree is rebalanced to maintain red-black properties.
// If the key doesn't exist, this is a no-op.
func (t *redBlackTreeMap[K, V]) Remove(key K) error {
	contains, err := t.Contains(key)
	if err != nil {
		return err
	}

	if !contains {
		return nil
	}

	z, _ := t.getNode(key) //nolint:varnamelen // Standard red-black tree variable names from CLRS
	y := z                 //nolint:varnamelen // Standard red-black tree variable names from CLRS
	yOriginalColor := y.color

	var x *rbtNode[K, V] //nolint:varnamelen // Standard red-black tree variable names from CLRS

	switch {
	case z.left == nil:
		x = z.right
		t.transplant(z, z.right)
	case z.right == nil:
		x = z.left
		t.transplant(z, z.left)
	default:
		y = t.getMinimum(z.right)
		yOriginalColor = y.color
		x = y.right

		if y.parent == z {
			if x != nil {
				x.parent = y
			}
		} else {
			t.transplant(y, y.right)
			y.right = z.right
			y.right.parent = y
		}

		t.transplant(z, y)

		y.left = z.left
		y.left.parent = y
		y.color = z.color
	}

	if yOriginalColor == black {
		t.fixupDelete(x)
	}

	return nil
}

// Clear removes all entries from the map, resetting it to empty.
func (t *redBlackTreeMap[K, V]) Clear() {
	t.root = nil
}

// Contains checks whether the map contains the given key.
func (t *redBlackTreeMap[K, V]) Contains(key K) (bool, error) {
	found, _, _ := t.internalLookup(nil, t.root, key, nodir)

	return found, nil
}

// countingVisitor is a visitor implementation that counts the number of nodes in the tree.
// It traverses the entire tree in-order and increments the count for each node.
type countingVisitor[K sortable.Sortable[K], V any] struct {
	Count int
}

// Visit recursively traverses the tree in-order (left, current, right) and counts nodes.
func (v *countingVisitor[K, V]) Visit(node *rbtNode[K, V]) bool {
	if node == nil {
		return true
	}

	if !v.Visit(node.left) {
		return false
	}

	v.Count++

	return v.Visit(node.right)
}

// Size returns the number of key-value pairs in the map.
// It traverses the entire tree to count nodes, so this is O(n).
func (t *redBlackTreeMap[K, V]) Size() int {
	vis := &countingVisitor[K, V]{}
	t.walk(vis)

	return vis.Count
}

// seqVisitor is a visitor implementation that yields key-value pairs in sorted order.
// It's used to implement the Seq method for range-based iteration.
type seqVisitor[K sortable.Sortable[K], V any] struct {
	yield func(K, V) bool
}

// Visit recursively traverses the tree in-order, yielding each key-value pair.
// Traversal stops early if yield returns false.
func (s *seqVisitor[K, V]) Visit(node *rbtNode[K, V]) bool {
	if node == nil {
		return true
	}

	if !s.Visit(node.left) {
		return false
	}

	if !s.yield(node.key, node.value) {
		return false
	}

	return s.Visit(node.right)
}

// Seq returns an iterator over the map's key-value pairs in sorted order (by key).
// This enables range-based iteration: for k, v := range m.Seq() { ... }.
func (t *redBlackTreeMap[K, V]) Seq() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		visit := &seqVisitor[K, V]{yield: yield}

		t.walk(visit)
	}
}

// Union returns a new map containing all entries from both this map and the other map.
// If a key exists in both maps, the value from the other map takes precedence.
func (t *redBlackTreeMap[K, V]) Union(other Map[K, V]) (Map[K, V], error) {
	out := NewRedBlackTreeMap[K, V]()

	for k, v := range t.Seq() {
		if err := out.Add(k, v); err != nil {
			return nil, err
		}
	}

	for k, v := range other.Seq() {
		if err := out.Add(k, v); err != nil {
			return nil, err
		}
	}

	return out, nil
}

// Intersection returns a new map containing only entries whose keys exist in both maps.
// Values are taken from this map, not the other map.
func (t *redBlackTreeMap[K, V]) Intersection(other Map[K, V]) (Map[K, V], error) {
	out := NewRedBlackTreeMap[K, V]()

	for k, v := range t.Seq() {
		found, err := other.Contains(k)
		if err != nil {
			return nil, err
		}

		if found {
			if err := out.Add(k, v); err != nil {
				return nil, err
			}
		}
	}

	return out, nil
}

// Clone returns a shallow copy of the map with the same key-value pairs.
func (t *redBlackTreeMap[K, V]) Clone() Map[K, V] {
	cloned := NewRedBlackTreeMap[K, V]()

	for key, value := range t.Seq() {
		_ = cloned.Add(key, value)
	}

	return cloned
}

// HashFunction returns nil as red-black trees don't use hash functions.
// They rely on sortable keys instead.
func (t *redBlackTreeMap[K, V]) HashFunction() hashing.HashFunc {
	return nil
}

// Keys returns a set containing all keys in the map.
func (t *redBlackTreeMap[K, V]) Keys() set.Set[K] {
	s := set.NewRedBlackTreeSet[K]()

	for key := range t.Seq() {
		_ = s.Add(key)
	}

	return s
}

// ForEach applies the given function to each key-value pair in the map.
// Entries are processed in sorted order by key.
func (t *redBlackTreeMap[K, V]) ForEach(f func(key K, value V)) {
	for k, v := range t.Seq() {
		f(k, v)
	}
}

// ForAll returns true if the predicate returns true for all key-value pairs in the map.
// Returns true for an empty map.
func (t *redBlackTreeMap[K, V]) ForAll(predicate func(key K, value V) bool) bool {
	for key, value := range t.Seq() {
		if !predicate(key, value) {
			return false
		}
	}

	return true
}

// Filter returns a new map containing only entries for which the predicate returns true.
func (t *redBlackTreeMap[K, V]) Filter(predicate func(key K, value V) bool) Map[K, V] {
	filtered := NewRedBlackTreeMap[K, V]()

	for key, value := range t.Seq() {
		if predicate(key, value) {
			_ = filtered.Add(key, value)
		}
	}

	return filtered
}

// FilterNot returns a new map containing only entries for which the predicate returns false.
func (t *redBlackTreeMap[K, V]) FilterNot(predicate func(key K, value V) bool) Map[K, V] {
	filtered := NewRedBlackTreeMap[K, V]()

	for key, value := range t.Seq() {
		if !predicate(key, value) {
			_ = filtered.Add(key, value)
		}
	}

	return filtered
}

// Map applies a transformation function to each key-value pair and returns a new map
// with the transformed entries.
func (t *redBlackTreeMap[K, V]) Map(f func(key K, value V) (K, V)) Map[K, V] {
	out := NewRedBlackTreeMap[K, V]()

	for key, value := range t.Seq() {
		keyOut, valOut := f(key, value)

		_ = out.Add(keyOut, valOut)
	}

	return out
}

// FlatMap applies a function that returns a map for each key-value pair,
// then flattens all resulting maps into a single map.
func (t *redBlackTreeMap[K, V]) FlatMap(f func(key K, value V) Map[K, V]) Map[K, V] {
	out := NewRedBlackTreeMap[K, V]()

	for key, value := range t.Seq() {
		m := f(key, value)

		if m != nil {
			for key2, val2 := range m.Seq() {
				_ = out.Add(key2, val2)
			}
		}
	}

	return out
}

// Exists returns true if at least one key-value pair satisfies the predicate.
func (t *redBlackTreeMap[K, V]) Exists(predicate func(key K, value V) bool) bool {
	for k, v := range t.Seq() {
		if predicate(k, v) {
			return true
		}
	}

	return false
}

// FindFirst returns the first key-value pair (in sorted order) that satisfies the predicate.
// Returns None if no pair satisfies the predicate.
func (t *redBlackTreeMap[K, V]) FindFirst(predicate func(key K, value V) bool) optional.Value[KeyValuePair[K, V]] {
	for k, v := range t.Seq() {
		if predicate(k, v) {
			return optional.Some[KeyValuePair[K, V]](KeyValuePair[K, V]{
				Key:   k,
				Value: v,
			})
		}
	}

	return optional.None[KeyValuePair[K, V]]()
}

// transplant replaces the subtree rooted at node u with the subtree rooted at node v.
// This is a helper used during node deletion.
func (t *redBlackTreeMap[K, V]) transplant(u *rbtNode[K, V], v *rbtNode[K, V]) {
	switch {
	case u.parent == nil:
		t.root = v
	case u == u.parent.left:
		u.parent.left = v
	default:
		u.parent.right = v
	}

	if v != nil {
		v.parent = u.parent
	}
}

// walk traverses the tree using the provided visitor.
func (t *redBlackTreeMap[K, V]) walk(visitor visitor[K, V]) {
	visitor.Visit(t.root)
}

// NewRedBlackTreeMap creates a new empty red-black tree map.
// The map maintains O(log n) performance for all operations by keeping the tree balanced.
func NewRedBlackTreeMap[K sortable.Sortable[K], V any]() Map[K, V] {
	return &redBlackTreeMap[K, V]{}
}

// isRed returns true if the node is red, false if the node is black or nil.
// nil nodes are considered black by red-black tree convention.
func isRed[K sortable.Sortable[K], V any](n *rbtNode[K, V]) bool {
	if n == nil {
		return false
	}

	return n.color == red
}

// fixupPut restores red-black tree properties after inserting a new node.
// New nodes are inserted as red, which may violate the property that red nodes
// cannot have red children. This method fixes violations by recoloring and rotating.
//
// The algorithm handles several cases:
//  1. New node is root - color it black
//  2. Parent is black - no violation, done
//  3. Parent is red:
//     a. Uncle is red - recolor parent, uncle, and grandparent
//     b. Uncle is black - perform rotations and recoloring
//
// The method continues fixing violations up the tree until no violations remain.
// nolint:varnamelen // Standard red-black tree variable names
func (t *redBlackTreeMap[K, V]) fixupPut(z *rbtNode[K, V]) {
loop:
	for {
		switch {
		case z.parent == nil:
			fallthrough
		case z.parent.color == black:
			break loop
		case z.parent.color == red:
			grandparent := z.parent.parent
			if z.parent == grandparent.left { //nolint:nestif // Red-black tree algorithm complexity
				y := grandparent.right
				if isRed(y) {
					z.parent.color = black
					y.color = black
					grandparent.color = red
					z = grandparent
				} else {
					if z == z.parent.right {
						z = z.parent
						t.rotateLeft(z)
					}

					z.parent.color = black
					grandparent.color = red
					t.rotateRight(grandparent)
				}
			} else {
				y := grandparent.left
				if isRed(y) {
					z.parent.color = black
					y.color = black
					grandparent.color = red
					z = grandparent
				} else {
					if z == z.parent.left {
						z = z.parent
						t.rotateRight(z)
					}

					z.parent.color = black
					grandparent.color = red
					t.rotateLeft(grandparent)
				}
			}
		}
	}

	t.root.color = black
}

// fixupDelete restores red-black tree properties after deleting a node.
// Deletion can violate the property that all paths from root to leaf have the
// same number of black nodes. This method fixes violations by recoloring and rotating.
//
// The algorithm handles several cases based on the "sibling" (w) of the node being fixed:
//  1. Node is root or red - can be colored black, done
//  2. Sibling is red - rotate and recolor to create a black sibling
//  3. Sibling is black with two black children - recolor sibling, move problem up
//  4. Sibling is black with red child - rotate and recolor to fix the violation
//
// The method is more complex than fixupPut because deletion affects black-height,
// requiring careful handling of all cases to maintain tree balance.
// nolint:varnamelen,dupl // Standard red-black tree variable names; symmetric cases
func (t *redBlackTreeMap[K, V]) fixupDelete(x *rbtNode[K, V]) {
	if x == nil {
		return
	}

loop:
	for {
		switch {
		case x == t.root:
			break loop
		case x.color == red:
			break loop
		case x == x.parent.right:
			w := x.parent.left //nolint:varnamelen // Standard red-black tree variable names from CLRS
			if isRed(w) {
				w.color = black
				x.parent.color = red
				t.rotateRight(x.parent)
				w = x.parent.left
			}
			if w != nil {
				switch {
				case !isRed(w.left) && !isRed(w.right):
					w.color = red
					x = x.parent // recurse up tree
				case isRed(w.right) && !isRed(w.left):
					w.right.color = black
					w.color = red
					t.rotateLeft(w)
					w = x.parent.left
				}
				if isRed(w.left) {
					w.color = x.parent.color
					x.parent.color = black
					w.left.color = black
					t.rotateRight(x.parent)
					x = t.root
				}
			}
		case x == x.parent.left:
			w := x.parent.right //nolint:varnamelen // Standard red-black tree variable names from CLRS
			if isRed(w) {
				w.color = black
				x.parent.color = red
				t.rotateLeft(x.parent)
				w = x.parent.right
			}
			if w != nil {
				switch {
				case !isRed(w.left) && !isRed(w.right):
					w.color = red
					x = x.parent // recurse up tree
				case isRed(w.left) && !isRed(w.right):
					w.left.color = black
					w.color = red
					t.rotateRight(w)
					w = x.parent.right
				}
				if isRed(w.right) {
					w.color = x.parent.color
					x.parent.color = black
					w.right.color = black
					t.rotateLeft(x.parent)
					x = t.root
				}
			}
		}
	}

	x.color = black
}

// getMinimum returns the node with the minimum key in the subtree rooted at x.
// This is always the leftmost node in the subtree.
// Used during deletion to find the in-order successor of a node.
func (t *redBlackTreeMap[K, V]) getMinimum(x *rbtNode[K, V]) *rbtNode[K, V] {
	for {
		if x.left != nil {
			x = x.left
		} else {
			return x
		}
	}
}
