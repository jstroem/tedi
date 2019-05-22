package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

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
	testCall        = `t.Test("%s", %s)` + "\n"
	beforeTestCall  = `t.BeforeTest(%s)` + "\n"
	afterTestCall   = `t.afterTest(%s)` + "\n"
)

var (
	funcname = flag.String("func", "TestMain", "name of the function to generate; default TestMain")
	prefix   = flag.String("prefix", "", "prefix name of tests; default <none>")
	output   = flag.String("output", "tedi_test.go", "output file name; default srcdir/tedi_test.go")
)

// Usage prints how the tedi command should be executed.
func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of tedi:\n")
	fmt.Fprintf(os.Stderr, "\ttedi [flags]\n")
	fmt.Fprintf(os.Stderr, "For more information, see:\n")
	fmt.Fprintf(os.Stderr, "\thttp://github.com/jstroem/tedi\n")
	fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("tedi: ")
	flag.Usage = Usage
	flag.Parse()

	dir, err := os.Getwd()
	if err != nil {
		die(err)
	}

	res, err := annotations.Parse(dir, "_test.go")
	if err != nil {
		die(err)
	}

	bytes := generateFile(res, *funcname, *prefix)

	if bytes, err = format.Source(bytes); err != nil {
		die(err)
	}

	outputFile := filepath.Join(dir, *output)
	err = ioutil.WriteFile(outputFile, bytes, 0644)
	if err != nil {
		log.Fatalf("writing output: %s", err)
	}
}

func generateFile(parsed *annotations.ParseResult, funcName, prefixTestName string) []byte {
	g := &generator{}

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
	if len(parsed.Fixtures) > 0 {
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
			fmt.Fprintf(&buf, beforeTestCall, test.Decl.Name.Name)
		}
	}

	if len(parsed.Tests) > 0 {
		fmt.Fprintln(&buf, "")
		fmt.Fprintln(&buf, "// Tests: ")
		for _, test := range parsed.Tests {
			fmt.Fprintf(&buf, testCall, prefixTestName+test.Decl.Name.Name, test.Decl.Name.Name)
		}
	}

	if len(parsed.AfterTests) > 0 {
		fmt.Fprintln(&buf, "")
		fmt.Fprintln(&buf, "// After tests: ")
		for _, test := range parsed.AfterTests {
			fmt.Fprintf(&buf, afterTestCall, test.Decl.Name.Name)
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
