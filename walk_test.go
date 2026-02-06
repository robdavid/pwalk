package pwalk

import (
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/robdavid/pwalk/pkgs/path"
	"github.com/robdavid/pwalk/pkgs/walk"
	"github.com/robdavid/pwalk/pkgs/workpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalk(t *testing.T) {
	count := 0
	wp := workpool.New(12, 0)
	var size int64
	Walk(".", func(pth path.RootedPath, err error, dirent fs.DirEntry) {
		require.NoError(t, err)
		fmt.Println(pth)
		count++
		if info, err := dirent.Info(); err == nil {
			size += info.Size()
		}
	}, walk.ConfigWorkpool(wp))
	wp.Stop()
	assert.Greater(t, wp.MaxActive, 0)
	assert.Greater(t, count, 100)
	assert.Greater(t, size, int64(5*1024*1024))
}

type MockDirEntry struct {
	name      string
	mode      fs.FileMode
	children  MockFilesystem
	size      int64
	modtime   time.Time
	readDelay time.Duration
}

type MockEntryInfo struct {
	*MockDirEntry
}

func (mde *MockDirEntry) Name() string {
	return mde.name
}

func (mde *MockDirEntry) Type() fs.FileMode {
	return mde.mode
}

func (mde *MockDirEntry) IsDir() bool {
	return mde.mode.IsDir()
}

func (mde *MockDirEntry) Info() (fs.FileInfo, error) {
	return MockEntryInfo{mde}, nil
}

func (info MockEntryInfo) ModTime() time.Time {
	return info.modtime
}

func (info MockEntryInfo) Mode() fs.FileMode {
	return info.mode
}

func (info MockEntryInfo) Size() int64 {
	return info.size
}

func (info MockEntryInfo) Sys() any {
	return nil
}

type MockFilesystem struct {
	root []*MockDirEntry
}

func (mf MockFilesystem) Get(name string) *MockDirEntry {
	for _, ent := range mf.root {
		if ent.name == name {
			return ent
		}
	}
	return nil
}

func (mf MockFilesystem) ReadDir(p path.RootedPath) ([]fs.DirEntry, error) {
	if len(p.SubPath) == 0 {
		entries := make([]fs.DirEntry, len(mf.root))
		for i, ent := range mf.root {
			entries[i] = ent
			if ent.readDelay > 0 {
				time.Sleep(ent.readDelay)
			}
		}
		slices.SortFunc(entries, func(e1, e2 fs.DirEntry) int { return strings.Compare(e1.Name(), e2.Name()) })
		return entries, nil
	} else {
		sub := mf.Get(p.SubPath[0])
		if sub == nil {
			return nil, fs.ErrNotExist
		}
		return sub.children.ReadDir(p.Sub())
	}
}

func (mf MockFilesystem) Lstat(p path.RootedPath) (fs.FileInfo, error) {
	switch len(p.SubPath) {
	case 0:
		return MockEntryInfo{&MockDirEntry{mode: os.ModeDir}}, nil
	case 1:
		ent := mf.Get(p.SubPath[0])
		if ent == nil {
			return nil, fs.ErrNotExist
		} else {
			return MockEntryInfo{ent}, nil
		}
	default:
		ent := mf.Get(p.SubPath[0])
		if !ent.IsDir() {
			return nil, fs.ErrNotExist
		} else {
			return ent.children.Lstat(p.Sub())
		}
	}
}

type buildInfo struct {
	mode   fs.FileMode
	repeat int
}

type buildConfig struct {
	breadth   int
	depth     int
	build     []buildInfo
	readDelay time.Duration
}

type buildState struct {
	config  *buildConfig
	depth   int
	infopos int
}

func (b *buildState) nextFilemode() fs.FileMode {

	// p is set to logical position of next buildinfo
	p := b.infopos
	var bi *buildInfo
	// Subtract the number of repeats per config build info
	// which should yield the buildinfo for logical position
	// b.infopos in bi.
	for i := range b.config.build {
		bi = &b.config.build[i]
		p -= bi.repeat
		if p <= 0 {
			break
		}
	}
	if p > 0 {
		// Positive p means b.infopos is greater than largest
		// logical position, so wrap around.
		bi = &b.config.build[0]
		b.infopos = 0
	} else {
		b.infopos++
	}
	if b.depth >= b.config.depth {
		// If we hit max depth, make everything a non-directory.
		return bi.mode & ^fs.ModeDir
	} else {
		return bi.mode
	}
}

func (b *buildState) makeTree() MockFilesystem {
	children := make([]*MockDirEntry, b.config.breadth)
	for i := range children {
		name := fmt.Sprintf("child-%d", i)
		mode := b.nextFilemode()
		children[i] = &MockDirEntry{
			name:      name,
			mode:      mode,
			modtime:   time.Now(),
			size:      int64(b.config.breadth)*int64(b.config.depth) + int64(i),
			readDelay: b.config.readDelay,
		}
		if mode.IsDir() {
			subB := buildState{
				config:  b.config,
				infopos: 0,
				depth:   b.depth + 1,
			}
			children[i].children = subB.makeTree()
		}
	}
	return MockFilesystem{root: children}
}

func BuildTree(config buildConfig) MockFilesystem {
	b := buildState{
		config:  &config,
		depth:   0,
		infopos: 0,
	}
	return b.makeTree()
}

func TestParTree(t *testing.T) {
	readDelay := time.Millisecond * 10
	tree := BuildTree(buildConfig{
		breadth:   15,
		depth:     5,
		readDelay: readDelay,
		build:     []buildInfo{{mode: 0777, repeat: 3}, {mode: os.ModeDir | 0777, repeat: 1}},
	})
	wp := workpool.New(12, 0)
	var size int64
	var count int
	prevPath := ""
	start := time.Now()
	Walk("", func(pth path.RootedPath, err error, dirent fs.DirEntry) {
		require.NoError(t, err)
		pthString := pth.String()
		if prevPath != "" {
			assert.Greater(t, pthString, prevPath)
		}
		prevPath = pthString
		if info, err := dirent.Info(); err == nil {
			size += info.Size()
			count++
		}
	}, walk.ConfigWorkpool(wp), walk.ConfigFilesystem(tree))
	wp.Stop()
	end := time.Now()
	assert.Greater(t, wp.MaxActive, 1)
	sequentialTime := readDelay * time.Duration(count)
	assert.Less(t, end.Sub(start), sequentialTime)
}
