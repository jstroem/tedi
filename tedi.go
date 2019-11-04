package tedi

import (
	"flag"
	"strings"
	"testing"
)

var (
	_tediTestLabels string
)

func init() {
	flag.StringVar(&_tediTestLabels, "labels", "test", "Tedi test labels to run. Can be multiple with ',' as a seperator")
}

// Tedi encapsulates tests for an entire package.
type Tedi struct {
	m *testing.M

	runLabels   stringSet
	labels      stringSet
	fixtures    []interface{}
	beforeTests map[string][]interface{}
	afterTests  map[string][]interface{}
}

// New creates a new tedi test.
func New(m *testing.M) *Tedi {
	if !flag.Parsed() {
		flag.Parse()
	}

	return &Tedi{
		m:           m,
		runLabels:   newStringSet(strings.Split(_tediTestLabels, ",")...),
		beforeTests: map[string][]interface{}{},
		afterTests:  map[string][]interface{}{},
	}
}

// Run executes the Tedi test.
func (t *Tedi) Run() int {
	return t.m.Run()
}

func (t *Tedi) TestLabel(name string) {
	t.labels.Add(name)
}
