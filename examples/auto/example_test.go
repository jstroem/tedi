package auto

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/jstroem/tedi"
)

type a struct {
	b string
}

func fixture_nameLength(t *testing.T, r int64) int {
	fmt.Println("Fixture rand", r)
	return len(t.Name())
}

func fixture_rand() int64 {
	fmt.Println("rand fixture called")
	rand.Seed(time.Now().UnixNano())
	return rand.Int63()
}

// @test
func test_timer(t *testing.T, foo int, _ printTimerFunc) {
	fmt.Println(t.Name(), foo)
}

func pre_print(t *tedi.T) {
	fmt.Println("CALLED BEFORE", t.Name())
}

func test_withSub(t *tedi.T, foo int) {
	fmt.Println(t.Name(), foo)
	t.Run("first", func(t *tedi.T, foo int) {
		fmt.Println(t.Name(), foo)
		t.Run("second", func(t *tedi.T, foo int) {
			fmt.Println(t.Name(), foo)
		})
	})
}

func integration_withFoo(t *tedi.T, foo int) {
	fmt.Println("MyIntegrationTest test executed")
}

func integration_simple(t *tedi.T) {
	fmt.Println("integration_Test test executed")
}

type printTimerFunc func()

func fixture_timer(t *tedi.T) printTimerFunc {
	start := time.Now()
	res := func() {
		fmt.Printf("Execution of: %s took: %v\n", t.Name(), time.Now().Sub(start))
	}
	t.AfterTest(res)
	return res
}

func test_withSleep(t *tedi.T, _ printTimerFunc) {
	time.Sleep(time.Second)
}
