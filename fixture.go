package tedi

import (
	"errors"
	"reflect"
	"sync"
	"testing"

	"go.uber.org/dig"
)

var (
	// ErrFixtureMustBeFunction thrown if a fixture is not a function
	ErrFixtureMustBeFunction = errors.New("fixture can only be functions")
	// ErrFixtureCannotProduceTestingTB thrown if a fixture produces a testing.TB
	ErrFixtureCannotProduceTestingTB = errors.New("fixture cannot produce testing.TB")

	testingTB = reflect.TypeOf((*testing.TB)(nil)).Elem()
)

// Fixture registers a function as a fixture to tedi.
func (t *Tedi) Fixture(fn interface{}) error {
	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func {
		return ErrFixtureMustBeFunction
	}

	for i := 0; i < fnType.NumOut(); i++ {
		if fnType.Out(i).Implements(testingTB) {
			return ErrFixtureCannotProduceTestingTB
		}
	}

	t.fixtures = append(t.fixtures, fn)
	return nil
}

// OnceFixture registers a function as a fixture that should only be called once.
func (t *Tedi) OnceFixture(fn interface{}) error {
	fnValue := reflect.ValueOf(fn)
	if fnValue.Kind() != reflect.Func {
		return ErrFixtureMustBeFunction
	}

	var o sync.Once
	var res []reflect.Value
	onceFnValue := reflect.MakeFunc(fnValue.Type(), func(args []reflect.Value) []reflect.Value {
		o.Do(func() {
			res = fnValue.Call(args)
		})
		return res
	})

	return t.Fixture(onceFnValue.Interface())
}

type onEndFunc func() error

func (t *Tedi) createContainer(test *testing.T, testName string) (*dig.Container, *T, error) {
	res := dig.New()
	for _, fn := range t.fixtures {
		if err := res.Provide(fn); err != nil {
			return nil, nil, err
		}
	}

	if err := res.Provide(func() *testing.T { return test }); err != nil {
		return nil, nil, err
	}

	tediTest := t.createT(test, res, testName)
	if err := res.Provide(func() *T { return tediTest }); err != nil {
		return nil, nil, err
	}
	return res, tediTest, nil
}
