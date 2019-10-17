package annotations

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_annotationRegexp(t *testing.T) {
	regexp := annotationRegexp("@foo")
	assert.True(t, regexp.MatchString("@foo"))
	assert.True(t, regexp.MatchString(" @foo"))
	assert.True(t, regexp.MatchString("@foo "))
	assert.True(t, regexp.MatchString(" @foo "))
	assert.False(t, regexp.MatchString("@foo(foo)"))
	assert.True(t, regexp.MatchString(`@foo
and something else`))
	assert.True(t, regexp.MatchString(`some other comment
@foo`))
}

func Test_annotationWithParamsRegexp(t *testing.T) {
	regexp := annotationWithParamsRegexp("@foo")
	assert.True(t, regexp.MatchString("@foo(foo)"))
	assert.True(t, regexp.MatchString("@foo(foo,bar)"))
	assert.True(t, regexp.MatchString("@foo(foo, bar)"))
	assert.True(t, regexp.MatchString("@foo(foo, bar, baz)"))
	assert.True(t, regexp.MatchString(`@foo(foo, bar, baz)
some other comment`))
	assert.True(t, regexp.MatchString(`some other comment
@foo(foo, bar, baz)`))
	assert.False(t, regexp.MatchString("@foo"))
}
func Test_annotationWithOptionalParamsRegexp(t *testing.T) {
	regexp := annotationWithOptionalParamsRegexp("@foo")
	assert.True(t, regexp.MatchString("@foo(foo)"))
	assert.True(t, regexp.MatchString("@foo(foo,bar)"))
	assert.True(t, regexp.MatchString("@foo(foo, bar)"))
	assert.True(t, regexp.MatchString("@foo(foo, bar, baz)"))
	assert.True(t, regexp.MatchString(`@foo(foo, bar, baz)
some other comment`))
	assert.True(t, regexp.MatchString(`some other comment
@foo(foo, bar, baz)`))
	assert.True(t, regexp.MatchString("@foo"))
}

func Test_getParams(t *testing.T) {
	tests := []struct {
		in     string
		regexp *regexp.Regexp
		ok     bool
		res    []string
	}{
		{
			in:     "@foo(foo)",
			regexp: annotationWithParamsRegexp("@foo"),
			ok:     true,
			res:    []string{"foo"},
		},
		{
			in:     "@foo()",
			regexp: annotationWithParamsRegexp("@foo"),
			ok:     false,
			res:    []string(nil),
		},
		{
			in:     "@foo(foo, bar, baz)",
			regexp: annotationWithParamsRegexp("@foo"),
			ok:     true,
			res:    []string{"foo", "bar", "baz"},
		},
		{
			in:     "@foo(foo,bar,baz)",
			regexp: annotationWithParamsRegexp("@foo"),
			ok:     true,
			res:    []string{"foo", "bar", "baz"},
		},
		{
			in:     "@foo",
			regexp: annotationWithOptionalParamsRegexp("@foo"),
			ok:     true,
			res:    nil,
		},
		{
			in:     "@foo(foo,bar,baz)",
			regexp: annotationWithOptionalParamsRegexp("@foo"),
			ok:     true,
			res:    []string{"foo", "bar", "baz"},
		},
	}

	for _, test := range tests {
		t.Run(test.in, func(t *testing.T) {
			res, ok := getParams(test.regexp, test.in)
			if assert.Equal(t, test.ok, ok) {
				assert.Equal(t, test.res, res)
			}
		})
	}
}
