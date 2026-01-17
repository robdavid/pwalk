package pwalk

import (
	"fmt"
	"io/fs"
	"testing"

	"github.com/robdavid/pwalk/pkgs/path"
	"github.com/robdavid/pwalk/pkgs/workpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalk(t *testing.T) {
	wp := workpool.New(12, 0)
	//defer wp.Stop()
	count := 0
	Walk(".", func(pth path.RootedPath, err error, dirent fs.DirEntry) {
		require.NoError(t, err)
		fmt.Println(pth)
		count++
	}, wp)
	wp.Stop()
	assert.Greater(t, wp.MaxActive, 0)
	assert.Greater(t, count, 100)
}
