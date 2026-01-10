package pwalk

import (
	"fmt"
	"testing"

	"github.com/robdavid/pwalk/pkgs/dir"
	"github.com/robdavid/pwalk/pkgs/path"
	"github.com/robdavid/pwalk/workpool"
	"github.com/stretchr/testify/require"
)

func TestWalk(t *testing.T) {
	wp := workpool.New(12, 12)
	defer wp.Stop()
	Walk(path.NewAt(`.`), func(d dir.Dir, err error) {
		require.NoError(t, err)
		for _, ent := range d.Entries {
			fmt.Println(d.Path.Append(ent.Name()).Path())
		}
	}, wp)
}
