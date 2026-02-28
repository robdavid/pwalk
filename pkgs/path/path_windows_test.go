//go:build windows

package path

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPath(t *testing.T) {
	assert := assert.New(t)
	p := new("a", "b")
	assert.Equal("a\\b", p.path())
}

func TestFullPath(t *testing.T) {
	assert := assert.New(t)
	p := new("a", "b")
	assert.Equal("x\\y\\a\\b", p.fullPath("x\\y\\"))
}

func TestNewPath(t *testing.T) {
	assert := assert.New(t)
	p := New("r", "a", "b")
	assert.Equal("r\\a\\b", p.Path())
}

func TestPathAppend(t *testing.T) {
	assert := assert.New(t)
	p1 := New("r", "a", "b")
	p2 := p1.Append("c", "d")
	assert.Equal("r\\a\\b\\c\\d", p2.Path())
}
