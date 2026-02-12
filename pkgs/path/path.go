package path

import (
	"iter"
	"path/filepath"
	"slices"
)

// Path is a string slice representing a file path split into directory elements
// plus a final file or directory element. An empty slice represents a path root.
type Path []string

func New(files ...string) Path {
	cloned := make([]string, len(files))
	copy(cloned, files)
	return Path(cloned)
}

func (p Path) Append(files ...string) Path {
	cloned := make([]string, len(p)+len(files))
	copy(cloned, p)
	copy(cloned[len(p):], files)
	return Path(cloned)
}

func (p *Path) Push(files ...string) {
	*p = append(*p, files...)
}

func (p *Path) Pop(n int) {
	if n > len(*p) {
		*p = (*p)[:0]
	} else {
		*p = (*p)[:len(*p)-n]
	}
}

func (p Path) Path() string {
	return filepath.Join(p...)
}

func (p Path) String() string {
	return p.Path()
}

func (p Path) FullPath(root string) string {
	return filepath.Join(root, p.Path())
}

func (p Path) AbsFile(root string) (string, error) {
	return filepath.Abs(p.FullPath(root))
}

func (p Path) HasPrefix(prefix Path) bool {
	if len(p) < len(prefix) {
		return false
	}
	for i := range prefix {
		if p[i] != prefix[i] {
			return false
		}
	}
	return true
}

// A RootedPath is a [Path] which is found relative to a location in
// the file system represented as a normal path string.
type RootedPath struct {
	root    string
	subPath Path
}

func NewAt(root string, subdirs ...string) RootedPath {
	return RootedPath{root, New(subdirs...)}
}

func (rp RootedPath) Append(files ...string) RootedPath {
	return RootedPath{rp.root, rp.subPath.Append(files...)}
}

func (rp RootedPath) Path() string {
	return rp.subPath.FullPath(rp.root)
}

func (rp RootedPath) String() string {
	return rp.Path()
}

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

func (rp RootedPath) Len() int {
	return len(rp.subPath)
}

func (rp RootedPath) IsRoot() bool {
	return len(rp.subPath) == 0
}

func (rp RootedPath) SubPaths() iter.Seq[string] {
	return slices.Values(rp.subPath)
}

func (rp RootedPath) Top() string {
	return rp.subPath[0]
}
