package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/jstroem/tedi/annotations"
)

const (
	tediPackage = "github.com/jstroem/tedi"

	funcBody = `func %s(m *testing.M) {
		t := tedi.New(m)

		%s

		os.Exit(t.Run())
	}`
	fixtureCall     = `t.Fixture(%s)` + "\n"
	onceFixtureCall = `t.OnceFixture(%s)` + "\n"
	testCall        = `t.Test("%s", %s%s)` + "\n"
	beforeTestCall  = `t.BeforeTest(%s%s)` + "\n"
	afterTestCall   = `t.AfterTest(%s%s)` + "\n"
	testLabelCall   = `t.TestLabel("%s")` + "\n"
)

func init() {
}

var (
	generateCmd      = flag.NewFlagSet("generate", flag.ExitOnError)
	generateFuncname = generateCmd.String("func", "TestMain", "name of the function to generate; default TestMain")
	generatePrefix   = generateCmd.String("prefix", "", "prefix name of tests; default <none>")
	generateOutput   = generateCmd.String("output", "tedi_test.go", "output file name; default srcdir/tedi_test.go")
	generateBuildTag = generateCmd.String("buildTag", "", "build tag to set in the generated file")

	testCmd = flag.NewFlagSet("test", flag.ExitOnError)

	testCover     = testCmd.Bool("cover", false, "")
	testCoverMode = testCmd.Bool("covermode", false, "")
	testCoverPkg  = testCmd.Bool("coverpkg", false, "")
	testExec      = testCmd.Bool("exec", false, "")
	testJSON      = testCmd.Bool("json", false, "")
	testVet       = testCmd.Bool("vet", false, "")

	// Compilation flags, ignored for now.
	// testC   = testCmd.Bool("c", false, "")
	// testI   = testCmd.Bool("i", false, "")
	// testO   = testCmd.Bool("o", false, "")
	testMatchBenchmarks      = testCmd.String("bench", "", "run only benchmarks matching `regexp`")
	testBenchmarkMemory      = testCmd.Bool("benchmem", false, "print memory allocations for benchmarks")
	testBenchTime            = testCmd.Duration("benchtime", time.Second, "run each benchmark for duration `d`")
	testBlockProfile         = testCmd.String("blockprofile", "", "write a goroutine blocking profile to `file`")
	testBlockProfileRate     = testCmd.Int("blockprofilerate", 1, "set blocking profile `rate` (see runtime.SetBlockProfileRate)")
	testCount                = testCmd.Uint("count", 1, "run tests and benchmarks `n` times")
	testCoverProfile         = testCmd.String("coverprofile", "", "write a coverage profile to `file`")
	testCPU                  = testCmd.String("cpu", "", "comma-separated `list` of cpu counts to run each test with")
	testCPUProfile           = testCmd.String("cpuprofile", "", "write a cpu profile to `file`")
	testFailFast             = testCmd.Bool("failfast", false, "do not start new tests after the first test failure")
	testMatchList            = testCmd.String("list", "", "list tests, examples, and benchmarks matching `regexp` then exit")
	testMemProfile           = testCmd.String("memprofile", "", "write an allocation profile to `file`")
	testMemProfileRate       = testCmd.Int("memprofilerate", 0, "set memory allocation profiling `rate` (see runtime.MemProfileRate)")
	testMutexProfile         = testCmd.String("mutexprofile", "", "write a mutex contention profile to the named file after execution")
	testMutexProfileFraction = testCmd.Int("mutexprofilefraction", 1, "if >= 0, calls runtime.SetMutexProfileFraction()")
	testOutputDir            = testCmd.String("outputdir", "", "write profiles to `dir`")
	testParallel             = testCmd.Int("parallel", runtime.GOMAXPROCS(0), "run at most `n` tests in parallel")
	testRun                  = testCmd.String("run", "", "run only tests and examples matching `regexp`")
	testShort                = testCmd.Bool("short", false, "run smaller test suite to save time")
	testTimeout              = testCmd.Duration("timeout", 0, "panic test binary after duration `d` (default 0, timeout disabled)")
	testTraceFile            = testCmd.String("trace", "", "write an execution trace to `file`")
	testV                    = testCmd.Bool("v", false, "verbose: print additional output")

	testTags = testCmd.String("tags", "", "tags")
)

// Usage prints how the tedi command should be executed.
func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of tedi:\n")
	fmt.Fprintf(os.Stderr, "\ttedi <command> [arguments]\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Commands are:\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "\tgenerate\tfor generation of tedi files\n")
	fmt.Fprintf(os.Stderr, "\ttest\t\tto run both generation and test in one command\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "For more information, see:\n")
	fmt.Fprintf(os.Stderr, "\thttp://github.com/jstroem/tedi\n")
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("tedi: ")
	flag.Usage = Usage

	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "generate":
		if err := generateCmd.Parse(os.Args[2:]); err != nil {
			die(err)
			os.Exit(2)
		}
		generateCommand()

	case "test":
		if err := testCmd.Parse(os.Args[2:]); err != nil {
			die(err)
			os.Exit(2)
		}
		testCommand()
	default:
		fmt.Printf("%q is not valid command.\n", os.Args[1])
		os.Exit(2)
	}
}

func generateCommand() {
	dir, err := os.Getwd()
	if err != nil {
		die(err)
	}

	res, err := annotations.Parse(dir, "_test.go", true)
	if err != nil {
		die(err)
	}

	if res == nil {
		return
	}

	for _, warning := range res.Warnings {
		log.Println(warning)
	}

	bytes := generateFile(res, *generateFuncname, *generatePrefix)

	if bytes, err = format.Source(bytes); err != nil {
		die(err)
	}

	outputFile := filepath.Join(dir, *generateOutput)
	err = ioutil.WriteFile(outputFile, bytes, 0644)
	if err != nil {
		log.Fatalf("writing output: %s", err)
	}
}

func testCommand() {
	cmd := exec.Command("go", append([]string{"generate"}, testCmd.Args()...)...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	cmd.Run()
	if exitCode := cmd.ProcessState.ExitCode(); exitCode != 0 {
		os.Exit(exitCode)
	}

	cmd = exec.Command("go", os.Args[1:]...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	cmd.Run()
	if exitCode := cmd.ProcessState.ExitCode(); exitCode != 0 {
		os.Exit(exitCode)
	}
}

func generateFile(parsed *annotations.ParseResult, funcName, prefixTestName string) []byte {
	g := &generator{}

	if tags := *generateBuildTag; len(tags) > 0 {
		g.Printf("// +build %s\n", tags)
		g.Printf("\n")
	}

	// Print the header and package clause.
	g.Printf("// Code generated by \"tedi %s\"; DO NOT EDIT.\n", strings.Join(os.Args[1:], " "))
	g.Printf("\n")
	g.Printf("package %s", parsed.Package.Name)
	g.Printf("\n")
	g.Printf("import (\n")
	g.Printf("\"%s\"\n", tediPackage)
	g.Printf("\"testing\"\n")
	g.Printf("\"os\"\n")
	g.Printf(")\n")

	var buf bytes.Buffer
	if len(parsed.TestLabels) > 0 {
		fmt.Fprintln(&buf, "// TestLabels: ")
		for label := range parsed.TestLabels {
			fmt.Fprintf(&buf, testLabelCall, label)
		}
	}

	if len(parsed.Fixtures) > 0 {
		fmt.Fprintln(&buf, "")
		fmt.Fprintln(&buf, "// Fixtures: ")
		for _, fixture := range parsed.Fixtures {
			fmt.Fprintf(&buf, fixtureCall, fixture.Decl.Name.Name)
		}
	}

	if len(parsed.OnceFixtures) > 0 {
		fmt.Fprintln(&buf, "")
		fmt.Fprintln(&buf, "// OnceFixtures: ")
		for _, fixture := range parsed.OnceFixtures {
			fmt.Fprintf(&buf, onceFixtureCall, fixture.Decl.Name.Name)
		}
	}

	if len(parsed.BeforeTests) > 0 {
		fmt.Fprintln(&buf, "")
		fmt.Fprintln(&buf, "// Before tests: ")
		for _, test := range parsed.BeforeTests {
			labelArgs := ""
			if len(test.Labels) > 0 {
				labelArgs = fmt.Sprint(`, "`, strings.Join(test.Labels, `", "`), `"`)
			}
			fmt.Fprintf(&buf, beforeTestCall, test.Decl.Name.Name, labelArgs)
		}
	}

	if len(parsed.Tests) > 0 {
		fmt.Fprintln(&buf, "")
		fmt.Fprintln(&buf, "// Tests: ")
		for _, test := range parsed.Tests {
			labelArgs := ""
			if len(test.Labels) > 0 {
				labelArgs = fmt.Sprint(`, "`, strings.Join(test.Labels, `", "`), `"`)
			}
			fmt.Fprintf(&buf, testCall, prefixTestName+test.Decl.Name.Name, test.Decl.Name.Name, labelArgs)
		}
	}

	if len(parsed.AfterTests) > 0 {
		fmt.Fprintln(&buf, "")
		fmt.Fprintln(&buf, "// After tests: ")
		for _, test := range parsed.AfterTests {
			labelArgs := ""
			if len(test.Labels) > 0 {
				labelArgs = fmt.Sprint(`, "`, strings.Join(test.Labels, `", "`), `"`)
			}
			fmt.Fprintf(&buf, afterTestCall, test.Decl.Name.Name, labelArgs)
		}
	}

	g.Printf(funcBody, funcName, buf.String())

	return g.buf.Bytes()
}

type generator struct {
	buf bytes.Buffer
}

func (g *generator) Printf(format string, args ...interface{}) {
	fmt.Fprintf(&g.buf, format, args...)
}

func die(err error) {
	log.Fatal(err)
}
