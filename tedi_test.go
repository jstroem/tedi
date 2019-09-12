package tedi

import (
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var (
	globalCount = &PrePostCounter{count: 0}
)

func addBeforeTest(t *testing.T) {
	globalCount.count++
}

func subAfterTest(t *testing.T) {
	globalCount.count--
}

type A struct {
	value int64
}

func fixtureProvideA() *A {
	return &A{value: rand.Int63()}
}

type B struct {
	value int64
}

func testFixture(t *testing.T, a *A) {
	assert.NotEqual(t, int64(0), a.value)
}

func onceFixtureProvideB() *B {
	return &B{value: rand.Int63()}
}

func testFixtureRenewed(t *T) {
	var a, b *A
	t.Run("first", func(in *A) {
		assert.NotEqual(t, int64(0), in.value)
		a = in
	})

	t.Run("second", func(in *A) {
		assert.NotEqual(t, int64(0), in.value)
		b = in
	})

	require.NotNil(t, a)
	require.NotNil(t, b)

	assert.NotEqual(t, a, b)
	assert.NotEqual(t, a.value, b.value)
}

func testOnceFixture(t *T) {
	var a, b *B
	t.Run("first", func(in *B) {
		assert.NotEqual(t, int64(0), in.value)
		a = in
	})

	t.Run("second", func(in *B) {
		assert.NotEqual(t, int64(0), in.value)
		b = in
	})

	require.NotNil(t, a)
	require.NotNil(t, b)

	assert.Equal(t, a, b)
	assert.Equal(t, a.value, b.value)
}

type PrePostCounter struct {
	count int
}

func fixturePrePostCounter(t *T) *PrePostCounter {
	cnt := &PrePostCounter{count: 0}

	t.BeforeTest(func(t *T) {
		assert.Equal(t, cnt.count, 0)
		cnt.count++
	})

	t.AfterTest(func(t *T) {
		assert.Equal(t, cnt.count, 2)
		cnt.count++
	})
	return cnt
}

func testInternalBeforeAndAfterTest(t *T, cnt *PrePostCounter) {
	assert.Equal(t, cnt.count, 1)
	cnt.count++
}

func testInternalAfterTestCalledOnFailedTest(t *T) {

	var cnt *PrePostCounter
	t.Run("FAIL", func(t *T, in *PrePostCounter) {
		cnt = in
		assert.Equal(t, cnt.count, 1)
		cnt.count++
		// Remove skip to test.
		t.Skip()
		assert.Fail(t, "this should fail to test AfterTest is called")
	})
	require.NotNil(t, cnt)
	assert.Equal(t, cnt.count, 3)
}

func testExternalBeforeAndAfterTest(t *T) {
	assert.Equal(t, globalCount.count, 1)
}

func TestMain(m *testing.M) {
	t := New(m)

	// Fixtures:
	t.Fixture(fixtureProvideA)
	t.Fixture(fixturePrePostCounter)

	// Once fixtures:
	t.OnceFixture(onceFixtureProvideB)

	// Before tests:
	t.BeforeTest(addBeforeTest)

	// After tests:
	t.AfterTest(subAfterTest)

	// Tests:
	t.Test("testFixture", testFixture)
	t.Test("testFixtureRenewed", testFixtureRenewed)
	t.Test("testOnceFixture", testOnceFixture)
	t.Test("testInternalBeforeAndAfterTest", testInternalBeforeAndAfterTest)
	t.Test("testExternalBeforeAndAfterTest", testExternalBeforeAndAfterTest)
	t.Test("testInternalAfterTestCalledOnFailedTest", testInternalAfterTestCalledOnFailedTest)

	os.Exit(t.Run())
}
