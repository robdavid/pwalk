package assemble

import (
	"github.com/robdavid/pwalk/pkgs/walk"
)

// CoOrd is a location in the virtual [Assembly], consisting
// of a directory pointer and an index for the entry within that
// directory.
type CoOrd[T any] struct {
	Dir   *walk.Dir[T]
	Index int
}

// Location is a vector of [CoOrd]s, representing the state of
// a depth first search algorithm. It represents a path from
// root to a location in the tree.
type Location[T any] []CoOrd[T]

// Last returns a pointer to the last element in a [Location], or
// nil if the [Location] is just the root path (zero path length)
func (l Location[T]) Last() *CoOrd[T] {
	if len(l) == 0 {
		return nil
	} else {
		return &l[len(l)-1]
	}
}

// Push adds a deeper directory level [CoOrd] to the given
// [Location]. It returns a new [Location] without modifying
// the receiver.
func (l Location[T]) Push(c CoOrd[T]) Location[T] {
	return append(l, c)
}

// Pop removes the last (deepest) [CoOrd] from the location,
// returning a new [Location] one level shallower than the caller.
func (l Location[T]) Pop() Location[T] {
	return l[:len(l)-1]
}
