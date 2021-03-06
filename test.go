package tedi

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
	"go.uber.org/dig"
)

// Test registers a function as a test.
func (t *Tedi) Test(name string, fn interface{}, labels ...string) {
	testsLabel := newStringSet(labels...)

	matchedLabels := testsLabel.Intersect(t.labels)
	matchedLabels = matchedLabels.Intersect(t.runLabels)
	if len(matchedLabels) > 0 {
		// Ignore test if the groupset does not overlap with the running set.
		testFn := t.wrapTest(name, fn, matchedLabels.List()...)
		t.addTest(name, testFn)
	}
}

// BeforeTest registers a function as a beforeTest hook.
func (t *Tedi) BeforeTest(fn interface{}) {
	t.beforeTests = append(t.beforeTests, fn)
}

// AfterTest registers a function as a afterTest hook.
func (t *Tedi) AfterTest(fn interface{}) {
	t.afterTests = append(t.afterTests, fn)
}

type testFunc func(t *testing.T)

func (t *Tedi) wrapTest(name string, fn interface{}, labels ...string) testFunc {
	return func(test *testing.T) {
		c, t, err := t.createContainer(test, name, labels...)
		require.NoError(test, err, "Failed to build container for test: %s", name)
		require.NoError(test, t.onStart(), "Failed to run onStart for test: %s", name)
		t.running = true
		defer func() {
			require.NoError(test, t.onEnd(), "Failed to run onEnd for test: %s", name)
		}()
		require.NoError(t, c.Invoke(fn), "Failed to Invoke test: %s", name)
	}
}

func (t *Tedi) addTest(name string, fn testFunc) {
	tests := reflect.ValueOf(t.m).Elem().FieldByName("tests")

	// tests is a private field on the tesing.M struct so we need to do this trick in order to add new tests.
	tests = reflect.NewAt(tests.Type(), unsafe.Pointer(tests.UnsafeAddr()))

	internalTestType := tests.Type().Elem().Elem()

	newTest := reflect.New(internalTestType)
	newTest.Elem().FieldByName("Name").Set(reflect.ValueOf(name))
	newTest.Elem().FieldByName("F").Set(reflect.ValueOf(fn))

	res := reflect.Append(tests.Elem(), newTest.Elem())
	tests.Elem().Set(res)
}

func (t *Tedi) createT(test *testing.T, container *dig.Container, testName string, testLabels ...string) *T {
	res := &T{
		T:           test,
		tedi:        t,
		container:   container,
		running:     false,
		testName:    testName,
		testLabels:  testLabels,
		beforeTests: t.beforeTests[:],
		afterTests:  t.afterTests[:],
	}
	return res
}

// T extends testing.T struct with hooks to be called before and
// after the test has been executed and a Run method that also
// works with dependency injection.
type T struct {
	*testing.T
	tedi       *Tedi
	container  *dig.Container
	running    bool
	testName   string
	testLabels []string

	beforeTests []interface{}
	afterTests  []interface{}
}

func (t *T) onStart() error {
	for _, fn := range t.beforeTests {
		if err := t.container.Invoke(fn); err != nil {
			return err
		}
	}
	return nil
}

func (t *T) onEnd() error {
	for i := range t.afterTests {
		if err := t.container.Invoke(t.afterTests[len(t.afterTests)-i-1]); err != nil {
			return err
		}
	}
	return nil
}

// BeforeTest register a function to be called before a test will run.
func (t *T) BeforeTest(fn interface{}) {
	if t.running {
		require.NoError(t, t.container.Invoke(fn), "Failed to run BeforeTest for test: %s", t.testName)
		return
	}
	t.beforeTests = append(t.beforeTests, fn)
}

// AfterTest register a function to be called once the test was executed.
func (t *T) AfterTest(fn interface{}) {
	t.afterTests = append(t.afterTests, fn)
}

// Run fn as a subtest of t similar to how testing.T.Run would work.
func (t *T) Run(name string, fn interface{}) bool {
	return t.T.Run(name, t.tedi.wrapTest(name, fn, t.testLabels...))
}

func (t *T) Labels() []string {
	return t.testLabels
}

func (t *T) HasLabel(label string) bool {
	for _, l := range t.testLabels {
		if l == label {
			return true
		}
	}
	return false
}
