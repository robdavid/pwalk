package path_test

import (
	"testing"

	"github.com/robdavid/pwalk/pkgs/path"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	assert := assert.New(t)
	p := path.New("a", "b")
	assert.Equal([]string{"a", "b"}, []string(p))
}

func TestNewEmpty(t *testing.T) {
	assert := assert.New(t)
	p := path.New()
	assert.Equal(0, len(p))
}

func TestAppend(t *testing.T) {
	assert := assert.New(t)
	p1 := path.New("a", "b")
	p2 := p1.Append("c", "d")
	assert.Equal([]string{"a", "b", "c", "d"}, []string(p2))
	assert.Equal([]string{"a", "b"}, []string(p1))
}

func TestAppendEmpty(t *testing.T) {
	assert := assert.New(t)
	p1 := path.New("a", "b")
	p2 := p1.Append()
	assert.Equal([]string{"a", "b"}, []string(p2))
	assert.Equal([]string{"a", "b"}, []string(p1))
}

func TestAppendTwice(t *testing.T) {
	assert := assert.New(t)
	p1 := path.New("a", "b")
	p2 := p1.Append("c", "d")
	p3 := p1.Append("x", "y")
	assert.Equal([]string{"a", "b", "c", "d"}, []string(p2))
	assert.Equal([]string{"a", "b", "x", "y"}, []string(p3))
}

func TestPushPop(t *testing.T) {
	assert := assert.New(t)
	p := path.New("a", "b")
	p.Push("c", "d")
	assert.Equal([]string{"a", "b", "c", "d"}, []string(p))
	p.Pop(2)
	assert.Equal([]string{"a", "b"}, []string(p))
	p.Pop(1)
	assert.Equal([]string{"a"}, []string(p))
}
