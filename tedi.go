package tedi

import (
	"testing"
)

// Tedi encapsulates tests for an entire package.
type Tedi struct {
	m *testing.M

	fixtures    []interface{}
	beforeTests []interface{}
	afterTests  []interface{}
}

// New creates a new tedi test.
func New(m *testing.M) *Tedi {
	return &Tedi{
		m: m,
	}
}

// Run executes the Tedi test.
func (t *Tedi) Run() int {
	return t.m.Run()
}
