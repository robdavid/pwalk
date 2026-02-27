package path

import (
	"iter"
	"path/filepath"
	"slices"
)

// pathSlice is a string slice representing a file path split into directory elements
// plus a final file or directory element. An empty slice represents a path root.
type pathSlice []string

// new creates a new [pathSlice] from a list of path elements.
func new(files ...string) pathSlice {
	cloned := make([]string, len(files))
	copy(cloned, files)
	return pathSlice(cloned)
}

// append adds a list of path elements to a [pathSlice], returning a new path.
// The original path is unchanged.
func (p pathSlice) append(files ...string) pathSlice {
	cloned := make([]string, len(p)+len(files))
	copy(cloned, p)
	copy(cloned[len(p):], files)
	return pathSlice(cloned)
}

// push mutates a [pathSlice] by adding elements to the end of it.
func (p *pathSlice) push(files ...string) {
	*p = append(*p, files...)
}

// Pop mutates a [pathSlice] by removing n elements from the end of it.
// if n is greater or equal to the number of elements, the
// path will be empty.
func (p *pathSlice) Pop(n int) {
	if n > len(*p) {
		*p = (*p)[:0]
	} else {
		*p = (*p)[:len(*p)-n]
	}
}

// path returns all path elements joined as a path string
func (p pathSlice) path() string {
	return filepath.Join(p...)
}

// String returns all path elements joined as a path string. Same as [pathSlice.Path].
func (p pathSlice) String() string {
	return p.path()
}

// fullPath returns the path string of the provided root followed by all
// the elements in the path.
func (p pathSlice) fullPath(root string) string {
	return filepath.Join(root, p.path())
}

// AbsPath returns the absolute path string of the provided root followed by all
// the elements in the path.
func (p pathSlice) absFile(root string) (string, error) {
	return filepath.Abs(p.fullPath(root))
}

// Path is a filesystem path rooted at a given directory. A Path
// is immutable.
type Path struct {
	root    string
	subPath pathSlice
	asStr   string
}

// New returns a [Path] which consists of a number of
// path elements which comprise a path relative to the provided root
// path string.
func New(root string, subdirs ...string) *Path {
	return &Path{root, new(subdirs...), ""}
}

// Append appends a number of path elements to a [Path],
// returning a new value. The original RootPath is unchanged.
func (rp *Path) Append(files ...string) *Path {
	return &Path{rp.root, rp.subPath.append(files...), ""}
}

// Path returns the entire [Path] as a string.
func (rp *Path) Path() string {
	if rp.asStr == "" {
		rp.asStr = rp.subPath.fullPath(rp.root)
	}
	return rp.asStr
}

// String returns the entire [Path] as a string. Same as [Path.Path].
func (rp *Path) String() string {
	return rp.Path()
}

// Sub returns a Path rooted at the first path subdirectory.
func (rp *Path) Sub() *Path {
	if len(rp.subPath) > 0 {
		return &Path{root: filepath.Join(rp.root, rp.subPath[0]), subPath: rp.subPath[1:]}
	} else {
		return rp
	}
}

// Len returns the number of path elements after the root.
func (rp *Path) Len() int {
	return len(rp.subPath)
}

// Root returns the root path string of this Path.
func (rp *Path) Root() string {
	return rp.root
}

// Get returns the path element at the given index after the root.
func (rp *Path) Get(n int) string {
	return rp.subPath[n]
}

// IsRoot returns true if this path is the root (i.e. there are no path elements
// after the root).
func (rp *Path) IsRoot() bool {
	return len(rp.subPath) == 0
}

// SubPaths returns an iterator over all path elements after the root
func (rp *Path) SubPaths() iter.Seq[string] {
	return slices.Values(rp.subPath)
}

// Top returns the first element of the path after the root
func (rp *Path) Top() string {
	return rp.subPath[0]
}
