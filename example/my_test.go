package test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/jstroem/tedi"
)

//go:generate tedi generate

// bar
type a struct {
	b string
}

// @fixture sdsd
func myFixture(t *testing.T, r int64) int {
	fmt.Println("Fixture rand", r)
	return len(t.Name())
}

// @onceFixture
func randFixture() int64 {
	fmt.Println("rand fixture called")
	rand.Seed(time.Now().UnixNano())
	return rand.Int63()
}

func MyTest(t *testing.T, foo int) {
	fmt.Println(t.Name(), foo)
}

// @test
func MyTest2(t *testing.T, foo int, _ printTimerFunc) {
	fmt.Println(t.Name(), foo)
}

// @beforeTest
func myBefore(t *tedi.T) {
	fmt.Println("CALLED BEFORE", t.Name())
}

// @test
func MyTest3(t *tedi.T, foo int) {
	fmt.Println(t.Name(), foo)
	t.Run("first", func(t *tedi.T, foo int) {
		fmt.Println(t.Name(), foo)
		t.Run("second", func(t *tedi.T, foo int) {
			fmt.Println(t.Name(), foo)
		})
	})
}

type printTimerFunc func()

// @fixture
func myTimer(t *tedi.T) printTimerFunc {
	start := time.Now()
	res := func() {
		fmt.Printf("Execution of: %s took: %v\n", t.Name(), time.Now().Sub(start))
	}
	t.AfterTest(res)
	return res
}

// @test
func MyTestTiming(t *tedi.T, _ printTimerFunc) {
	time.Sleep(time.Second)
}
