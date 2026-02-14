package pwalk_test

import (
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/robdavid/pwalk"
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
	pwalk.Walk(".", func(pth path.RootedPath, dirent fs.DirEntry, err error) {
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
	assert.Greater(t, size, int64(3*1024*1024))
}

type MockDirEntry struct {
	name      string
	mode      fs.FileMode
	children  MockFilesystem
	size      int64
	modtime   time.Time
	readDelay time.Duration
	err       error
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
	if p.Len() == 0 {
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
		sub := mf.Get(p.Top())
		if sub == nil {
			return nil, fs.ErrNotExist
		}
		entries, err := sub.children.ReadDir(p.Sub())
		if p.Len() == 1 {
			return entries, sub.err
		} else {
			return entries, err
		}
	}
}

func (mf MockFilesystem) Lstat(p path.RootedPath) (fs.FileInfo, error) {
	switch p.Len() {
	case 0:
		return MockEntryInfo{&MockDirEntry{mode: os.ModeDir}}, nil
	case 1:
		ent := mf.Get(p.Top())
		if ent == nil {
			return nil, fs.ErrNotExist
		} else {
			return MockEntryInfo{ent}, ent.err
		}
	default:
		ent := mf.Get(p.Top())
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
	err    error
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

func (b *buildState) nextFilemode() (fs.FileMode, error) {

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
		return bi.mode & ^fs.ModeDir, nil
	} else {
		return bi.mode, bi.err
	}
}

func (b *buildState) makeTree() (MockFilesystem, int) {
	children := make([]*MockDirEntry, b.config.breadth)
	total := len(children)
	for i := range children {
		name := fmt.Sprintf("child-%d", i)
		mode, err := b.nextFilemode()
		children[i] = &MockDirEntry{
			name:      name,
			mode:      mode,
			modtime:   time.Now(),
			size:      int64(b.config.breadth)*int64(b.config.depth) + int64(i),
			readDelay: b.config.readDelay,
			err:       err,
		}
		if mode.IsDir() {
			var subtotal int
			subB := buildState{
				config:  b.config,
				infopos: 0,
				depth:   b.depth + 1,
			}
			children[i].children, subtotal = subB.makeTree()
			total += subtotal
		}
	}
	return MockFilesystem{root: children}, total
}

func BuildTree(config buildConfig) (MockFilesystem, int) {
	b := buildState{
		config:  &config,
		depth:   0,
		infopos: 0,
	}
	fs, entries := b.makeTree()
	return fs, entries + 1
}

func TestParTree(t *testing.T) {
	readDelay := time.Millisecond * 10
	tree, entries := BuildTree(buildConfig{
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
	pwalk.Walk("", func(pth path.RootedPath, dirent fs.DirEntry, err error) {
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
	fmt.Println("Count:", count)
	assert.Equal(t, entries, count)
	end := time.Now()
	assert.Greater(t, wp.MaxActive, 1)
	sequentialTime := readDelay * time.Duration(count)
	assert.Less(t, end.Sub(start), sequentialTime)
}

func TestParMapTree(t *testing.T) {
	readDelay := time.Millisecond * 10
	tree, entries := BuildTree(buildConfig{
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
	pwalk.WalkAndMap[fs.FileInfo]("",
		func(pth path.RootedPath, dirent fs.DirEntry, err error) (fs.FileInfo, error) {
			return dirent.Info()
		},
		func(pth path.RootedPath, dirent fs.DirEntry, err error, info fs.FileInfo) {
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
			assert.Equal(t, dirent.Type(), info.Mode())
		},
		walk.ConfigWorkpool(wp), walk.ConfigFilesystem(tree),
	)
	wp.Stop()
	fmt.Println("Count:", count)
	assert.Equal(t, entries, count)
	end := time.Now()
	assert.Greater(t, wp.MaxActive, 1)
	sequentialTime := readDelay * time.Duration(count)
	assert.Less(t, end.Sub(start), sequentialTime)
}

func filterCount(t *testing.T, counter *atomic.Int32) walk.Filter {
	return func(rp path.RootedPath, de os.DirEntry, err error) (walk.FilterAction, error) {
		if err != nil {
			return walk.FilterSkipDir, err
		}
		counter.Add(1)
		return walk.FilterAccept, nil
	}
}

func TestParTreeNoSymlinks(t *testing.T) {
	readDelay := time.Millisecond * 10
	tree, max := BuildTree(buildConfig{
		breadth:   15,
		depth:     5,
		readDelay: readDelay,
		build: []buildInfo{
			{mode: 0777, repeat: 3},
			{mode: os.ModeDir | 0777, repeat: 1},
			{mode: 0777, repeat: 3},
			{mode: os.ModeDir | os.ModeSymlink, repeat: 1},
		},
	})
	wp := workpool.New(12, 0)
	var size int64
	var count int
	var entries atomic.Int32
	prevPath := ""
	start := time.Now()
	pwalk.Walk("", func(pth path.RootedPath, dirent fs.DirEntry, err error) {
		require.NoError(t, err)
		pthString := pth.String()
		if prevPath != "" {
			require.Greater(t, pthString, prevPath)
		}
		prevPath = pthString
		if info, err := dirent.Info(); err == nil {
			size += info.Size()
			count++
		}
	},
		walk.ConfigWorkpool(wp),
		walk.ConfigFilesystem(tree),
		walk.ConfigFilter(walk.FilterDirSymLinks),
		walk.ConfigFilter(filterCount(t, &entries)),
	)
	wp.Stop()
	end := time.Now()
	assert.Less(t, count, max)
	assert.Equal(t, int(entries.Load())+1, count)
	assert.Greater(t, wp.MaxActive, 1)
	sequentialTime := readDelay * time.Duration(count)
	assert.Less(t, end.Sub(start), sequentialTime)
}

func filterErrs(t *testing.T, counter, skipped *atomic.Int32) walk.Filter {
	return func(rp path.RootedPath, de os.DirEntry, err error) (walk.FilterAction, error) {
		if err != nil {
			skipped.Add(1)
			return walk.FilterSkip, err
		}
		counter.Add(1)
		return walk.FilterAccept, nil
	}
}

func TestParTreeWithErrs(t *testing.T) {
	readDelay := time.Millisecond * 10
	tree, max := BuildTree(buildConfig{
		breadth:   15,
		depth:     5,
		readDelay: readDelay,
		build: []buildInfo{
			{mode: 0777, repeat: 3},
			{mode: os.ModeDir | 0777, repeat: 1},
			{mode: 0777, repeat: 3},
			{mode: os.ModeDir, repeat: 1, err: os.ErrPermission},
		},
	})
	wp := workpool.New(12, 0)
	var count int
	var entries, skipped atomic.Int32
	prevPath := ""
	start := time.Now()
	pwalk.Walk("", func(pth path.RootedPath, dirent fs.DirEntry, err error) {
		assert.NoError(t, err, "Error for path %s", pth)
		pthString := pth.String()
		if prevPath != "" {
			require.Greater(t, pthString, prevPath)
		}
		count++
		prevPath = pthString
	},
		walk.ConfigWorkpool(wp),
		walk.ConfigFilesystem(tree),
		walk.ConfigFilter(filterErrs(t, &entries, &skipped)),
	)
	wp.Stop()
	end := time.Now()
	assert.Less(t, count, max)
	// Skipped entries are also seen by the filter in non-err state in dir entry
	// prior to error state when an attempt is made to read that directory.
	// Therefore removing their number from the total count gives the entry
	// count the walk function observes.
	assert.Equal(t, int(entries.Load()-skipped.Load())+1, count)
	assert.Greater(t, wp.MaxActive, 1)
	sequentialTime := readDelay * time.Duration(count)
	assert.Less(t, end.Sub(start), sequentialTime)
}

func filterErrsSkipdir(t *testing.T, counter, skipped *atomic.Int32) walk.Filter {
	return func(rp path.RootedPath, de os.DirEntry, err error) (walk.FilterAction, error) {
		if err != nil {
			skipped.Add(1)
			return walk.FilterSkipDir, err
		}
		counter.Add(1)
		return walk.FilterAccept, nil
	}
}

func TestParTreeWithErrsSkipdir(t *testing.T) {
	readDelay := time.Millisecond * 10
	tree, max := BuildTree(buildConfig{
		breadth:   15,
		depth:     5,
		readDelay: readDelay,
		build: []buildInfo{
			{mode: 0777, repeat: 3},
			{mode: os.ModeDir | 0777, repeat: 1},
			{mode: 0777, repeat: 3},
			{mode: os.ModeDir, repeat: 1, err: os.ErrPermission},
		},
	})
	wp := workpool.New(12, 0)
	var count, errCount int
	var entries, skipped atomic.Int32
	prevPath := ""
	start := time.Now()
	pwalk.Walk("", func(pth path.RootedPath, dirent fs.DirEntry, err error) {
		if err != nil {
			assert.ErrorIs(t, err, os.ErrPermission)
			errCount++
		}
		pthString := pth.String()
		if prevPath != "" {
			require.Greater(t, pthString, prevPath)
		}
		count++
		prevPath = pthString
	},
		walk.ConfigWorkpool(wp),
		walk.ConfigFilesystem(tree),
		walk.ConfigFilter(filterErrsSkipdir(t, &entries, &skipped)),
	)
	wp.Stop()
	end := time.Now()
	assert.Less(t, count, max)
	assert.Equal(t, int(entries.Load())+1, count)
	assert.Equal(t, int(skipped.Load()), errCount)
	assert.Greater(t, wp.MaxActive, 1)
	sequentialTime := readDelay * time.Duration(count)
	assert.Less(t, end.Sub(start), sequentialTime)
}
