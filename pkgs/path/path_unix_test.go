//go:build unix

package path_test

import (
	"testing"

	"github.com/robdavid/pwalk/pkgs/path"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPath(t *testing.T) {
	assert := assert.New(t)
	p := path.New("a", "b")
	assert.Equal("a/b", p.Path())
}

func TestFullPath(t *testing.T) {
	assert := assert.New(t)
	p := path.New("a", "b")
	assert.Equal("x/y/a/b", p.FullPath("x/y/"))
}

func TestNewAt(t *testing.T) {
	assert := assert.New(t)
	p := path.NewAt("r", "a", "b")
	assert.Equal("r/a/b", p.Path())
}

func TestRootedAppend(t *testing.T) {
	assert := assert.New(t)
	p1 := path.NewAt("r", "a", "b")
	p2 := p1.Append("c", "d")
	assert.Equal("r/a/b/c/d", p2.Path())
}

func TestRootedAbs(t *testing.T) {
	assert := assert.New(t)
	p := path.NewAt("/var/lib", "dbus", "machine-id")
	abs, err := p.AbsFile()
	assert.NoError(err)
	assert.Equal("/var/lib/dbus/machine-id", abs)
}

func TestIsCaseInsensitive(t *testing.T) {
	ins, err := path.IsCaseInsensitive("/tmp")
	require.NoError(t, err)
	assert.False(t, ins)
}
