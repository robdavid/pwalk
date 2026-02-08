package path

import (
	"os"
	"path/filepath"
	"strings"
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

// IsCaseInsensitive returns true if the filesystem at dir appears case-insensitive.
func IsCaseInsensitive(dir string) (bool, error) {
	f, err := os.CreateTemp(dir, "caseprobe-*")
	if err != nil {
		return false, err
	}
	orig := f.Name()
	f.Close()
	defer os.Remove(orig)

	base := filepath.Base(orig)
	dirpath := filepath.Dir(orig)
	altBase := strings.ToUpper(base)

	altPath := filepath.Join(dirpath, altBase)

	// Try to create the alternate name exclusively.
	f2, err := os.OpenFile(altPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err == nil {
		// Created distinct file => case-sensitive
		f2.Close()
		os.Remove(altPath)
		return false, nil
	}
	if os.IsExist(err) {
		// Creation failed because file exists (same name under different case) => case-insensitive
		return true, nil
	}

	// If we get here, creation failed for another reason (permissions etc).
	// Fall back to stat-based check: see if alt path resolves to the same file.
	fi1, err1 := os.Stat(orig)
	if err1 != nil {
		return false, err1
	}
	fi2, err2 := os.Stat(altPath)
	if err2 == nil {
		// Both exist; check if they are same file
		if os.SameFile(fi1, fi2) {
			return true, nil
		}
		return false, nil
	}
	if os.IsNotExist(err2) {
		// alt doesn't exist => case-sensitive
		return false, nil
	}
	return false, err2
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

func (rp RootedPath) Sub() RootedPath {
	if len(rp.SubPath) > 0 {
		return RootedPath{Root: filepath.Join(rp.Root, rp.SubPath[0]), SubPath: rp.SubPath[1:]}
	} else {
		return rp
	}
}
