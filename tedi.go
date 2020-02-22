package tedi

import (
	"flag"
	"fmt"
	"strings"
	"testing"

	"github.com/jstroem/tedi/annotations"
)

var (
	_tediTestLabels string
)

func init() {
	flag.StringVar(&_tediTestLabels, "labels", annotations.DefaultTestLabel, "Tedi test labels to run. Can be multiple with ',' as a seperator")
}

// Tedi encapsulates tests for an entire package.
type Tedi struct {
	m *testing.M

	runLabels   stringSet
	labels      stringSet
	fixtures    []interface{}
	beforeTests []interface{}
	afterTests  []interface{}
}

// New creates a new tedi test.
func New(m *testing.M) *Tedi {
	if !flag.Parsed() {
		flag.Parse()
	}

	return &Tedi{
		m:           m,
		runLabels:   newStringSet(strings.Split(_tediTestLabels, ",")...),
		beforeTests: []interface{}{},
		afterTests:  []interface{}{},
	}
}

// Run executes the Tedi test.
func (t *Tedi) Run() int {
	if len(t.runLabels.Intersect(t.labels)) == 0 {
		fmt.Println("tedi: warning: labels did not match any tests. Available labels:", strings.Join(t.labels.List(), ", "))
	}
	return t.m.Run()
}

func (t *Tedi) TestLabel(name string) {
	t.labels.Add(name)
}
