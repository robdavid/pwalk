package path

import (
	"path/filepath"
)

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

func (p Path) AbsPath(root string) (string, error) {
	return filepath.Abs(p.FullPath(root))
}

type RootedPath struct {
	Root    string
	SubPath Path
}

func NewAt(root string, subdirs ...string) RootedPath {
	return RootedPath{root, New(subdirs...)}
}

func (rp RootedPath) Append(files ...string) RootedPath {
	return RootedPath{rp.Root, rp.SubPath.Append(files...)}
}

func (rp *RootedPath) Push(files ...string) {
	rp.SubPath.Push(files...)
}

func (rp *RootedPath) Pop(n int) {
	rp.SubPath.Pop(n)
}

func (rp RootedPath) Path() string {
	return rp.SubPath.FullPath(rp.Root)
}

func (rp RootedPath) String() string {
	return rp.Path()
}

func (rp RootedPath) AbsPath() (string, error) {
	return rp.SubPath.AbsPath(rp.Root)
}
