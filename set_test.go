package tedi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_stringSet(t *testing.T) {
	s1 := newStringSet()
	s1.Add("a", "b", "c")
	assert.True(t, s1.Has("a"))
	assert.False(t, s1.Has("d"))

	var s2 stringSet
	s2.Add("a", "b", "c")
	assert.True(t, s2.Has("a"))
	assert.False(t, s2.Has("d"))
}
