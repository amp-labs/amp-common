package set

import (
	"fmt"
	"iter"

	"github.com/amp-labs/amp-common/hashing"
	"github.com/amp-labs/amp-common/sortable"
)

// visitor defines the interface for tree traversal using the visitor pattern.
// Implementations should return true to continue traversal, false to stop.
type visitor[K sortable.Sortable[K]] interface {
	Visit(node *rbtNode[K]) bool
}

// countingVisitor is a visitor implementation that counts the total number of nodes in the tree.
// It performs an in-order traversal and increments Count for each non-nil node visited.
type countingVisitor[K sortable.Sortable[K]] struct {
	Count int
}

// Visit performs an in-order traversal of the tree, incrementing the count for each node.
// It recursively visits the left subtree, increments the count, then visits the right subtree.
func (v *countingVisitor[K]) Visit(node *rbtNode[K]) bool {
	if node == nil {
		return true
	}

	if !v.Visit(node.left) {
		return false
	}

	v.Count++

	return v.Visit(node.right)
}

// color represents the color of a node in the red-black tree.
// Red-black trees maintain balance by coloring nodes either red or black
// and enforcing specific color properties during insertions and deletions.
type color bool

// direction represents the relationship between a parent and child node (left, right, or none).
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
	// black and red represent the two possible node colors in a red-black tree.
	// Black is represented as true for efficient nil checks (nil nodes are considered black).
	black, red color = true, false

	// left, right, and nodir represent the directional relationship between nodes.
	// nodir is used when there is no directional relationship (e.g., for the root node).
	left direction = iota
	right
	nodir
)

// rbtNode represents a single node in the red-black tree.
// Each node contains a key, color, and pointers to its parent and children.
type rbtNode[K sortable.Sortable[K]] struct {
	key    K
	color  color
	left   *rbtNode[K]
	right  *rbtNode[K]
	parent *rbtNode[K]
}

// String returns a string representation of the node showing its key and color.
func (n *rbtNode[K]) String() string {
	return fmt.Sprintf("(%#v : %s)", n.key, n.Color())
}

// Parent returns the parent node of this node.
func (n *rbtNode[K]) Parent() *rbtNode[K] {
	return n.parent
}

// SetColor updates the color of this node.
func (n *rbtNode[K]) SetColor(color color) {
	n.color = color
}

// Color returns the color of this node.
func (n *rbtNode[K]) Color() color {
	return n.color
}

// redBlackTreeSet is a Set implementation backed by a red-black tree.
//
// Red-black trees are self-balancing binary search trees that maintain the following properties:
//  1. Every node is either red or black.
//  2. The root is black.
//  3. All leaves (nil) are black.
//  4. If a node is red, then both its children are black (no two red nodes in a row).
//  5. Every path from a node to its descendant nil nodes contains the same number of black nodes.
//
// These properties ensure the tree remains approximately balanced, guaranteeing O(log n)
// time complexity for insertions, deletions, and lookups.
//
// The implementation follows the algorithms from "Introduction to Algorithms" (CLRS).
type redBlackTreeSet[K sortable.Sortable[K]] struct {
	root *rbtNode[K]
}

// AddAll adds multiple elements to the set.
// Returns an error if any element fails to be added (though current implementation never returns errors).
func (r *redBlackTreeSet[K]) AddAll(elements ...K) error {
	for _, element := range elements {
		if err := r.Add(element); err != nil {
			return err
		}
	}

	return nil
}

// Add inserts a new element into the set.
// If the element already exists, the set remains unchanged.
// After insertion, the tree is rebalanced using fixupPut to maintain red-black properties.
// Time complexity: O(log n).
func (r *redBlackTreeSet[K]) Add(element K) error {
	if r.root == nil {
		r.root = &rbtNode[K]{key: element, color: black}

		return nil
	}

	found, parent, dir := r.internalLookup(nil, r.root, element, nodir)
	if found {
		return nil
	}

	if parent != nil {
		newNode := &rbtNode[K]{key: element, parent: parent}

		switch dir {
		case left:
			parent.left = newNode
		case right:
			parent.right = newNode
		case nodir:
		}

		r.fixupPut(newNode)
	}

	return nil
}

// Remove deletes an element from the set.
// If the element does not exist, the set remains unchanged.
// After deletion, the tree is rebalanced using fixupDelete to maintain red-black properties.
// Time complexity: O(log n)
//
// The algorithm follows CLRS chapter 13:
//  1. Find the node to delete (z)
//  2. Identify the node that will be moved or removed (y)
//  3. Track y's original color and the node that takes y's place (x)
//  4. Perform the deletion using transplant operations
//  5. If a black node was removed, rebalance the tree with fixupDelete
func (r *redBlackTreeSet[K]) Remove(element K) error {
	contains, err := r.Contains(element)
	if err != nil {
		return err
	}

	if !contains {
		return nil
	}

	z, _ := r.getNode(element) //nolint:varnamelen // Standard red-black tree variable names from CLRS
	y := z                     //nolint:varnamelen // Standard red-black tree variable names from CLRS
	yOriginalColor := y.color

	var x *rbtNode[K] //nolint:varnamelen // Standard red-black tree variable names from CLRS

	switch {
	case z.left == nil:
		x = z.right
		r.transplant(z, z.right)
	case z.right == nil:
		x = z.left
		r.transplant(z, z.left)
	default:
		y = r.getMinimum(z.right)
		yOriginalColor = y.color
		x = y.right

		if y.parent == z {
			if x != nil {
				x.parent = y
			}
		} else {
			r.transplant(y, y.right)
			y.right = z.right
			y.right.parent = y
		}

		r.transplant(z, y)

		y.left = z.left
		y.left.parent = y
		y.color = z.color
	}

	if yOriginalColor == black {
		r.fixupDelete(x)
	}

	return nil
}

// Clear removes all elements from the set by setting the root to nil.
// Time complexity: O(1).
func (r *redBlackTreeSet[K]) Clear() {
	r.root = nil
}

// Contains checks if an element exists in the set.
// Time complexity: O(log n).
func (r *redBlackTreeSet[K]) Contains(element K) (bool, error) {
	found, _, _ := r.internalLookup(nil, r.root, element, nodir)

	return found, nil
}

// Size returns the number of elements in the set.
// It performs a full tree traversal using a counting visitor.
// Time complexity: O(n).
func (r *redBlackTreeSet[K]) Size() int {
	vis := &countingVisitor[K]{}
	r.walk(vis)

	return vis.Count
}

// Entries returns all elements in the set as a slice, in sorted order.
// Time complexity: O(n).
func (r *redBlackTreeSet[K]) Entries() []K {
	num := r.Size()

	if num == 0 {
		return nil
	}

	entries := make([]K, 0, num)

	for k := range r.Seq() {
		entries = append(entries, k)
	}

	return entries
}

// seqVisitor is a visitor implementation that yields elements to an iterator function.
// It enables Go 1.23+ range-over-func iteration support.
type seqVisitor[K sortable.Sortable[K]] struct {
	yield func(K) bool
}

// Visit performs an in-order traversal, yielding each element to the iterator function.
// Traversal stops early if the yield function returns false.
func (s *seqVisitor[K]) Visit(node *rbtNode[K]) bool {
	if node == nil {
		return true
	}

	if !s.Visit(node.left) {
		return false
	}

	if !s.yield(node.key) {
		return false
	}

	return s.Visit(node.right)
}

// Seq returns an iterator that yields elements in sorted order (in-order traversal).
// This enables Go 1.23+ range-over-func syntax: for element := range set.Seq() { ... }
// Time complexity: O(n) to iterate all elements.
func (r *redBlackTreeSet[K]) Seq() iter.Seq[K] {
	return func(yield func(K) bool) {
		visit := &seqVisitor[K]{yield: yield}

		r.walk(visit)
	}
}

// Union returns a new set containing all elements from both this set and the other set.
// Time complexity: O(n + m) where n and m are the sizes of the two sets.
func (r *redBlackTreeSet[K]) Union(other Set[K]) (Set[K], error) {
	out := NewRedBlackTreeSet[K]()

	for k := range r.Seq() {
		if err := out.Add(k); err != nil {
			return nil, err
		}
	}

	for k := range other.Seq() {
		if err := out.Add(k); err != nil {
			return nil, err
		}
	}

	return out, nil
}

// Intersection returns a new set containing only elements that exist in both this set and the other set.
// Time complexity: O(n log m) where n is the size of this set and m is the size of the other set.
func (r *redBlackTreeSet[K]) Intersection(other Set[K]) (Set[K], error) {
	out := NewRedBlackTreeSet[K]()

	for k := range r.Seq() {
		contains, err := other.Contains(k)
		if err != nil {
			return nil, err
		}

		if contains {
			if err := out.Add(k); err != nil {
				return nil, err
			}
		}
	}

	return out, nil
}

// HashFunction returns nil because red-black tree sets do not use hashing.
// This method exists to satisfy the Set interface.
func (r *redBlackTreeSet[K]) HashFunction() hashing.HashFunc {
	return nil
}

// Clone creates a shallow copy of the set with all the same elements.
// Time complexity: O(n).
func (r *redBlackTreeSet[K]) Clone() Set[K] {
	out := NewRedBlackTreeSet[K]()

	for k := range r.Seq() {
		_ = out.Add(k)
	}

	return out
}

// NewRedBlackTreeSet creates a new empty red-black tree set.
// The returned set maintains elements in sorted order and provides O(log n) operations.
func NewRedBlackTreeSet[K sortable.Sortable[K]]() Set[K] {
	return &redBlackTreeSet[K]{}
}

// walk performs a traversal of the tree using the provided visitor.
// The visitor controls the traversal order and termination.
func (r *redBlackTreeSet[K]) walk(visitor visitor[K]) {
	visitor.Visit(r.root)
}

// internalLookup recursively searches for a key in the tree.
// Returns:
//   - bool: true if the key was found, false otherwise
//   - *rbtNode[K]: the parent of the found/insertion node
//   - direction: the direction from parent to the found/insertion node
//
// This is the core search algorithm used by Add, Remove, and Contains.
func (r *redBlackTreeSet[K]) internalLookup(
	parent *rbtNode[K], this *rbtNode[K], key K, dir direction,
) (bool, *rbtNode[K], direction) {
	switch {
	case this == nil:
		return false, parent, dir
	case key.Equals(this.key):
		return true, parent, dir
	case key.LessThan(this.key):
		return r.internalLookup(this, this.left, key, left)
	default:
		return r.internalLookup(this, this.right, key, right)
	}
}

// getParent finds the parent node for a given key.
// Returns the same values as internalLookup.
func (r *redBlackTreeSet[K]) getParent(key K) (found bool, parent *rbtNode[K], dir direction) {
	if r.root == nil {
		return false, nil, nodir
	}

	return r.internalLookup(nil, r.root, key, nodir)
}

// getNode retrieves the node with the given key.
// Returns the node and true if found, nil and false otherwise.
func (r *redBlackTreeSet[K]) getNode(key K) (*rbtNode[K], bool) {
	found, parent, dir := r.getParent(key)
	if found {
		if parent == nil {
			return r.root, true
		} else {
			var node *rbtNode[K]

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
//
// Before:        y              After:         x
//
//	   / \                           / \
//	  x   c                         a   y
//	 / \              =>               / \
//	a   b                            b   c
//
// This operation maintains the binary search tree property while restructuring
// the tree for rebalancing. Used during insertion and deletion fixup.
//
//nolint:varnamelen,dupword // Standard red-black tree variable names; ASCII diagram
func (r *redBlackTreeSet[K]) rotateRight(y *rbtNode[K]) {
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
		r.root = x
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
//
// Before:                       After:
//
//	  x                             y
//	 / \                           / \
//	a   y                         x   c
//	   / \            =>         / \
//	  b   c                     a   b
//
// This operation maintains the binary search tree property while restructuring
// the tree for rebalancing. Used during insertion and deletion fixup.
//
// nolint:varnamelen // Standard red-black tree variable names
func (r *redBlackTreeSet[K]) rotateLeft(x *rbtNode[K]) {
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
		r.root = y
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

// transplant replaces subtree rooted at u with subtree rooted at v.
// This is a helper function used during deletion to replace nodes in the tree.
// The parent pointers are updated, but v's children are not modified.
func (r *redBlackTreeSet[K]) transplant(u *rbtNode[K], v *rbtNode[K]) {
	switch {
	case u.parent == nil:
		r.root = v
	case u == u.parent.left:
		u.parent.left = v
	default:
		u.parent.right = v
	}

	if v != nil {
		v.parent = u.parent
	}
}

// fixupPut restores red-black tree properties after insertion.
//
// After a standard BST insertion (where new nodes are colored red), the tree may violate
// red-black property #4 (no two consecutive red nodes). This method fixes violations by
// recoloring nodes and performing rotations.
//
// The algorithm handles three cases based on the color of the uncle node (y):
//
//	Case 1: Uncle is red → Recolor parent, uncle, and grandparent
//	Case 2: Uncle is black and z is a "middle child" → Rotate to convert to Case 3
//	Case 3: Uncle is black and z is an "outer child" → Rotate and recolor
//
// The loop terminates when z's parent is black or z becomes the root.
// Finally, the root is always colored black to maintain property #2.
//
// nolint:varnamelen // Standard red-black tree variable names
func (r *redBlackTreeSet[K]) fixupPut(z *rbtNode[K]) {
loop:
	for {
		switch {
		case z.parent == nil:
			fallthrough
		case z.parent.color == black:
			// When the loop terminates, it does so because p[z] is black.
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
						r.rotateLeft(z)
					}

					z.parent.color = black
					grandparent.color = red
					r.rotateRight(grandparent)
				}
			} else {
				y := grandparent.left
				if isRed(y) {
					// case 1 - y is RED
					z.parent.color = black
					y.color = black
					grandparent.color = red
					z = grandparent
				} else {
					if z == z.parent.left {
						z = z.parent
						r.rotateRight(z)
					}

					z.parent.color = black
					grandparent.color = red
					r.rotateLeft(grandparent)
				}
			}
		}
	}

	r.root.color = black
}

// fixupDelete restores red-black tree properties after deletion.
//
// After a standard BST deletion, if a black node was removed, the tree may violate
// red-black property #5 (equal black height on all paths). This method fixes violations
// by recoloring nodes and performing rotations.
//
// The algorithm maintains an "extra black" on node x and pushes it up the tree until:
//   - x becomes the root (extra black can be removed)
//   - x is red (color it black to absorb the extra black)
//
// For each iteration, there are four symmetric cases based on x's sibling (w):
//
//	Case 1: Sibling w is red → Convert to Case 2, 3, or 4 by rotating and recoloring
//	Case 2: Sibling w is black with two black children → Push black up the tree
//	Case 3: Sibling w is black with red far child and black near child → Convert to Case 4
//	Case 4: Sibling w is black with red near child → Rotate, recolor, and terminate
//
// The implementation handles both left and right symmetric cases.
//
// nolint:varnamelen,dupl // Standard red-black tree variable names; symmetric cases
func (r *redBlackTreeSet[K]) fixupDelete(x *rbtNode[K]) {
	if x == nil {
		return
	}

loop:
	for {
		switch {
		case x == r.root:
			break loop
		case x.color == red:
			break loop
		case x == x.parent.right:
			w := x.parent.left //nolint:varnamelen // Standard red-black tree variable names
			if isRed(w) {
				w.color = black
				x.parent.color = red
				r.rotateRight(x.parent)
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
					r.rotateLeft(w)
					w = x.parent.left
				}
				if isRed(w.left) {
					w.color = x.parent.color
					x.parent.color = black
					w.left.color = black
					r.rotateRight(x.parent)
					x = r.root
				}
			}
		case x == x.parent.left:
			w := x.parent.right //nolint:varnamelen // Standard red-black tree variable names from CLRS
			if isRed(w) {
				w.color = black
				x.parent.color = red
				r.rotateLeft(x.parent)
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
					r.rotateRight(w)
					w = x.parent.right
				}
				if isRed(w.right) {
					w.color = x.parent.color
					x.parent.color = black
					w.right.color = black
					r.rotateLeft(x.parent)
					x = r.root
				}
			}
		}
	}

	x.color = black
}

// isRed checks if a node is red.
// Nil nodes are considered black (following red-black tree convention).
func isRed[K sortable.Sortable[K]](n *rbtNode[K]) bool {
	if n == nil {
		return false
	}

	return n.color == red
}

// getMinimum finds the node with the smallest key in the subtree rooted at x.
// This is always the leftmost node in the subtree.
// Used during deletion to find the in-order successor.
func (r *redBlackTreeSet[K]) getMinimum(x *rbtNode[K]) *rbtNode[K] {
	for {
		if x.left != nil {
			x = x.left
		} else {
			return x
		}
	}
}
