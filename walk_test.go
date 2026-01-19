package pwalk

import (
	"fmt"
	"io/fs"
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
	name    string
	mode    fs.FileMode
	child   *MockDirEntry
	size    int64
	modtime time.Time
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
	root *MockDirEntry
}

type buildInfo struct {
	mode fs.FileMode
}
type buildConfig struct {
	breadth int
	depth   int
}
