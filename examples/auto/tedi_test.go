// Code generated by tedi; DO NOT EDIT.

package auto

import (
	"github.com/jstroem/tedi"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	t := tedi.New(m)

	// TestLabels:
	t.TestLabel("integration")
	t.TestLabel("regression")
	t.TestLabel("unit")

	// Fixtures:
	t.Fixture(fixtureNameLength)
	t.Fixture(fixtureRand)
	t.Fixture(fixtureTimer)

	// Before tests:
	t.BeforeTest(prePrint)

	// Tests:
	t.Test("testTimer", testTimer, "unit")
	t.Test("testWithSub", testWithSub, "unit")
	t.Test("integrationWithInt", integrationWithInt, "integration")
	t.Test("integrationPrint", integrationPrint, "integration")
	t.Test("testWithSleep", testWithSleep, "unit")

	os.Exit(t.Run())
}
