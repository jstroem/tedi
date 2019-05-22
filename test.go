package tedi

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
	"go.uber.org/dig"
)

// Test registers a function as a test.
func (t *Tedi) Test(name string, fn interface{}) {
	testFn := t.wrapTest(name, fn)
	t.addTest(name, testFn)
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

func (t *Tedi) wrapTest(name string, fn interface{}) testFunc {
	return func(test *testing.T) {
		c, t, err := t.createContainer(test)
		require.NoError(test, err, "Failed to build container for test: %s", name)
		require.NoError(test, t.onStart(), "Failed to run onStart for test: %s", name)
		require.NoError(test, c.Invoke(fn), "Failed to Invoke test: %s", name)
		require.NoError(test, t.onEnd(), "Failed to run onEnd for test: %s", name)
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

func (t *Tedi) createT(test *testing.T, container *dig.Container) *T {
	res := &T{
		T:           test,
		tedi:        t,
		container:   container,
		beforeTests: make([]interface{}, len(t.beforeTests)),
		afterTests:  make([]interface{}, len(t.afterTests)),
	}
	copy(res.beforeTests, t.beforeTests)
	copy(res.afterTests, t.afterTests)
	return res
}

// T extends testing.T struct with hooks to be called before and
// after the test has been executed and a Run method that also
// works with dependency injection.
type T struct {
	*testing.T
	tedi      *Tedi
	container *dig.Container

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
	t.beforeTests = append(t.beforeTests, fn)
}

// AfterTest register a function to be called once the test was executed.
func (t *T) AfterTest(fn interface{}) {
	t.afterTests = append(t.afterTests, fn)
}

// Run fn as a subtest of t similar to how testing.T.Run would work.
func (t *T) Run(name string, fn interface{}) bool {
	return t.T.Run(name, t.tedi.wrapTest(name, fn))
}
