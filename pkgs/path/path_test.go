package path

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	assert := assert.New(t)
	p := new("a", "b")
	assert.Equal([]string{"a", "b"}, []string(p))
}

func TestNewEmpty(t *testing.T) {
	assert := assert.New(t)
	p := new()
	assert.Equal(0, len(p))
}

func TestAppend(t *testing.T) {
	assert := assert.New(t)
	p1 := new("a", "b")
	p2 := p1.append("c", "d")
	assert.Equal([]string{"a", "b", "c", "d"}, []string(p2))
	assert.Equal([]string{"a", "b"}, []string(p1))
}

func TestAppendEmpty(t *testing.T) {
	assert := assert.New(t)
	p1 := new("a", "b")
	p2 := p1.append()
	assert.Equal([]string{"a", "b"}, []string(p2))
	assert.Equal([]string{"a", "b"}, []string(p1))
}

func TestAppendTwice(t *testing.T) {
	assert := assert.New(t)
	p1 := new("a", "b")
	p2 := p1.append("c", "d")
	p3 := p1.append("x", "y")
	assert.Equal([]string{"a", "b", "c", "d"}, []string(p2))
	assert.Equal([]string{"a", "b", "x", "y"}, []string(p3))
}

func TestPushPop(t *testing.T) {
	assert := assert.New(t)
	p := new("a", "b")
	p.push("c", "d")
	assert.Equal([]string{"a", "b", "c", "d"}, []string(p))
	p.Pop(2)
	assert.Equal([]string{"a", "b"}, []string(p))
	p.Pop(1)
	assert.Equal([]string{"a"}, []string(p))
}

func TestPathString(t *testing.T) {
	rp := New("/a/b/c")
	assert.Equal(t, "/a/b/c", rp.String())
	rp = New("/a", "b", "c")
	assert.Equal(t, "/a/b/c", rp.String())
	assert.Equal(t, "/a/b/c", rp.String())
}
