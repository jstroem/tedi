package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/build"
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
	"golang.org/x/tools/go/packages"
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
	testRace                 = testCmd.Bool("race", false, "enable the race detector when running tests")
	testV                    = testCmd.Bool("v", false, "verbose: print additional output")

	tediTestLabels = testCmd.String("labels", "test", "Tedi test labels to run. Can be multiple with ',' as a seperator")

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

	if err = writeTediFile(dir, writeTediFileOptions{
		Funcname:   *generateFuncname,
		Prefix:     *generatePrefix,
		BuildTag:   *generateBuildTag,
		OutputFile: *generateOutput,
	}); err != nil {
		die(err)
	}
}

type writeTediFileOptions struct {
	Funcname   string
	Prefix     string
	BuildTag   string
	OutputFile string
	ForceWrite bool
}

func writeTediFile(dir string, o writeTediFileOptions) error {
	res, err := annotations.Parse(dir, "_test.go", true)
	if err != nil {
		return err
	}

	if res == nil || res.Package == nil {
		return nil
	}

	bytes, write := generateFile(res, o.Funcname, o.Prefix, o.BuildTag)
	if !write && !o.ForceWrite {
		return nil
	}

	for _, warning := range res.Warnings {
		log.Println(warning)
	}

	if bytes, err = format.Source(bytes); err != nil {
		return err
	}

	outputFile := filepath.Join(dir, o.OutputFile)
	return ioutil.WriteFile(outputFile, bytes, 0644)
}

func pathToPackageDirs(args []string) ([]string, error) {
	var paths []string
	for _, a := range args {
		if build.IsLocalImport(a) {
			paths = append(paths, a)
		}
	}

	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName & packages.NeedFiles,
	}, paths...)
	if err != nil {
		return nil, err
	}

	var res []string
	for _, pkg := range pkgs {
		if len(pkg.GoFiles) > 0 {
			res = append(res, filepath.Dir(pkg.GoFiles[0]))
		}
	}
	return res, nil
}

func testCommand() {
	paths, err := pathToPackageDirs(testCmd.Args())
	if err != nil {
		die(err)
	}

	for _, path := range paths {
		if err := writeTediFile(path, writeTediFileOptions{
			Funcname:   "TestMain",
			Prefix:     "",
			OutputFile: "tedi_test.go",
			BuildTag:   "",
			ForceWrite: true,
		}); err != nil {
			die(err)
		}
	}

	idx := 0
	for i, arg := range os.Args {
		if arg == "-labels" {
			idx = i
			break
		}
	}

	// '-labels' flag is a custom 'tedi' flag so we need to move them as the last arguments to go test.
	if idx > 0 && idx+1 < len(os.Args) {
		labelFlag, labelValue := os.Args[idx], os.Args[idx+1]
		os.Args = append(append(os.Args[0:idx], os.Args[idx+2:]...), labelFlag, labelValue)
	}

	cmd := exec.Command("go", os.Args[1:]...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	cmd.Run()
	if exitCode := cmd.ProcessState.ExitCode(); exitCode != 0 {
		os.Exit(exitCode)
	}
}

func generateFile(parsed *annotations.ParseResult, funcName, prefixTestName, buildTags string) ([]byte, bool) {
	g := &generator{}

	if tags := buildTags; len(tags) > 0 {
		g.Printf("// +build %s\n", tags)
		g.Printf("\n")
	}

	// Print the header and package clause.
	g.Printf("// Code generated by tedi; DO NOT EDIT.\n")
	g.Printf("\n")
	g.Printf("package %s", parsed.Package.Name)
	g.Printf("\n")
	g.Printf("import (\n")
	g.Printf("\"%s\"\n", tediPackage)
	g.Printf("\"testing\"\n")
	g.Printf("\"os\"\n")
	g.Printf(")\n")

	write := false

	var buf bytes.Buffer
	if len(parsed.TestLabels) > 0 {
		fmt.Fprintln(&buf, "// TestLabels: ")
		for label := range parsed.TestLabels {
			fmt.Fprintf(&buf, testLabelCall, label)
		}
	}

	if len(parsed.Fixtures) > 0 {
		write = true
		fmt.Fprintln(&buf, "")
		fmt.Fprintln(&buf, "// Fixtures: ")
		for _, fixture := range parsed.Fixtures {
			fmt.Fprintf(&buf, fixtureCall, fixture.Decl.Name.Name)
		}
	}

	if len(parsed.OnceFixtures) > 0 {
		write = true
		fmt.Fprintln(&buf, "")
		fmt.Fprintln(&buf, "// OnceFixtures: ")
		for _, fixture := range parsed.OnceFixtures {
			fmt.Fprintf(&buf, onceFixtureCall, fixture.Decl.Name.Name)
		}
	}

	if len(parsed.BeforeTests) > 0 {
		write = true
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
		write = true
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
		write = true
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

	return g.buf.Bytes(), write
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
