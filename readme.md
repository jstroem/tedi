# Tedi - Test Environment with Dependency Injection

Tedi tries to make testing in go less tedious by extending the built-in test framework in Golang with dependency injection.

## An example

Tedi makes it possible to specify fixtures that later can be used in test:

```
type A struct {
    rand int64
}

// @fixture
func NewA() *A {
	rand.Seed(time.Now().UnixNano())
	&A{rand: rand.Int63()}
}

// @test
func myTest(t *tedi.T, a *A) {
    // write your test
}
```

## How to use `tedi`?

You can simply swap `tedi test` with `go test`.

`tedi test` will first generate the `tedi_test.go` file and then call the go test command.

### With `go test`

If you still want to use `go test` you can add:

```
    //go:generate tedi generate
```

to a file in your go package where you want to use `tedi`. Before running your run `go test` run `go generate`.

## Hooks

### Fixtures

A fixture is a function that provides one or more objects to the test environment. To mark a function as a fixture it must have the prefix `fix` or `fixture` or have the annotation `@fixture`.

Example with annotation:

```
// @fixture
func NewA() *A {
	rand.Seed(time.Now().UnixNano())
	&A{rand: rand.Int63()}
}
```
Example with prefix:
```
// @fixture
func fixA() *A {
	rand.Seed(time.Now().UnixNano())
	&A{rand: rand.Int63()}
}
```

**Note:** every time a fixture is needed by a test it will be executed. If you only want fixtures to be executed once you should use the label `@onceFixture`.

### BeforeTest

A BeforeTest function is executed before a test will be executed. To mark a function as a BeforeTest use the prefix `pre` or `beforeTest` or the label `@beforeTest`.

Example with annotation:

```
// @beforeTest
func PrintTest(t *tedi.T) {
	fmt.Println(t.Name())
}
```
Example with prefix:
```
func preTest(t *tedi.T) {
	fmt.Println(t.Name())
}
```

### AfterTest

A AfterTest function is executed after a test will be executed. To mark a function as a AfterTest use the prefix `post` or `afterTest` or the label `@afterTest`.

Example with annotation:

```
// @afterTest
func PrintTest(t *tedi.T) {
	fmt.Println(t.Name())
}
```
Example with prefix:
```
func postTest(t *tedi.T) {
	fmt.Println(t.Name())
}
```

### Test

A Test function using Tedi is similar to a normal go test. The Tedi framework only extends the functionality of normal tests. A Tedi test function can take multiple arguments which already has been provided as fixtures. To mark a function as test use the prefix `test` or the label `@test`.

Example with annotation:

```
// @test
func randIsPositive(t *tedi.T, a *A) {
	assert.True(t, a.rand > 0)
}
```
Example with prefix:
```
func testRandIsPositive(t *tedi.T, a *A) {
	assert.True(t, a.rand > 0)
}
```

In tedi tests you can use `tedi.T` instead of `testing.T` that makes it possible to make sub-tests that also can leverage the fixtures provided.

## Labeling

Tedi makes it possible to group test using labels. In some scenarios you might want to have multiple types of tests such as integration, regression and unit tests.

By default tedi has 3 labels `regression`, `integration` and `unit`. To label a test add it as parameter in your test annotation `@test(<label>)` as `@test(regression)`, or by using the defined prefixes:


* `reg` or `regression` for regression tests.
* `int` or `integration` for integration tests.
* `unit` or `test` for unit tests.

By default the `tedi test` command will execute unit tests but by using the flag `labels` you can execute different labels like `tedi test -labels regression,integration` will execute integration a regression tests but not unit test.

**Note:** the label flag is also available if you use tedi with the `go test` command.

### Custom labels

You can add your own labels and prefixes to auto match functions into labels with by using the annotation: `@testLabel`. This can be useful if you want another type of tests outside of the default tedi comes with.

Example:

```
// Create a new label called 'blackbox' which matches the prefix: 'black_'
// @testLabel(blackbox,black_)

// @test(blackbox)
func someTest(t *tedi.T) {
	assert.True(t, true)
}

func black_someOtherTest(t *tedi.T) {
    assert.True(t, true)
}
```

These tests would be executed by using the command `tedi test -label blackbox`.


## Disable auto matching using prefixes

TODO


