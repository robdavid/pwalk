package path

import (
	"iter"
	"path/filepath"
	"slices"
)

// Path is a string slice representing a file path split into directory elements
// plus a final file or directory element. An empty slice represents a path root.
type Path []string

// New creates a new [Path] from a list of path elements.
func New(files ...string) Path {
	cloned := make([]string, len(files))
	copy(cloned, files)
	return Path(cloned)
}

// Append adds a list of path elements to a [Path], returning a new path.
// The original path is unchanged.
func (p Path) Append(files ...string) Path {
	cloned := make([]string, len(p)+len(files))
	copy(cloned, p)
	copy(cloned[len(p):], files)
	return Path(cloned)
}

// Push mutates a [Path] by adding elements to the end of it.
func (p *Path) Push(files ...string) {
	*p = append(*p, files...)
}

// Pop mutates a [Path] by removing n elements from the end of it.
// if n is greater or equal to the number of elements, the
// path will be empty.
func (p *Path) Pop(n int) {
	if n > len(*p) {
		*p = (*p)[:0]
	} else {
		*p = (*p)[:len(*p)-n]
	}
}

// Path returns all path elements joined as a path string
func (p Path) Path() string {
	return filepath.Join(p...)
}

// String returns all path elements joined as a path string. Same as [Path.Path].
func (p Path) String() string {
	return p.Path()
}

// FullPath returns the path string of the provided root followed by all
// the elements in the path.
func (p Path) FullPath(root string) string {
	return filepath.Join(root, p.Path())
}

// AbsPath returns the absolute path string of the provided root followed by all
// the elements in the path.
func (p Path) AbsFile(root string) (string, error) {
	return filepath.Abs(p.FullPath(root))
}

// RootedPath is a [Path] which is found relative to a location in
// the file system represented as a normal path string. A RootedPath
// is immutable.
type RootedPath struct {
	root    string
	subPath Path
}

// NewAt returns a [RootedPath] which consists of a number of
// path elements which comprise a path relative to the provided root
// path string.
func NewAt(root string, subdirs ...string) RootedPath {
	return RootedPath{root, New(subdirs...)}
}

// Append appends a number of path elements to a [RootedPath],
// returning a new value. The original RootPath is unchanged.
func (rp RootedPath) Append(files ...string) RootedPath {
	return RootedPath{rp.root, rp.subPath.Append(files...)}
}

// Path returns the entire [RootedPath] as a string.
func (rp RootedPath) Path() string {
	return rp.subPath.FullPath(rp.root)
}

// String returns the entire [RootedPath] as a string. Same as [RootedPath.Path].
func (rp RootedPath) String() string {
	return rp.Path()
}

// AbsFile returns the entire file path as an absolute path string.
func (rp RootedPath) AbsFile() (string, error) {
	return rp.subPath.AbsFile(rp.root)
}

// Sub returns a RootedPath rooted at the first path subdirectory.
func (rp RootedPath) Sub() RootedPath {
	if len(rp.subPath) > 0 {
		return RootedPath{root: filepath.Join(rp.root, rp.subPath[0]), subPath: rp.subPath[1:]}
	} else {
		return rp
	}
}

// Len returns the number of path elements after the root.
func (rp RootedPath) Len() int {
	return len(rp.subPath)
}

// IsRoot returns true if this path is the root (i.e. there are no path elements
// after the root).
func (rp RootedPath) IsRoot() bool {
	return len(rp.subPath) == 0
}

// SubPaths returns an iterator over all path elements after the root
func (rp RootedPath) SubPaths() iter.Seq[string] {
	return slices.Values(rp.subPath)
}

// Top returns the first element of the path after the root
func (rp RootedPath) Top() string {
	return rp.subPath[0]
}
