package annotations

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strings"
)

const (
	// FixtureAnnotation used to label a function as a fixture
	FixtureAnnotation = "@fixture"

	// OnceFixtureAnnotation used to label a function as a fixture to be called once
	OnceFixtureAnnotation = "@onceFixture"

	// TestAnnotation used to label a function as a test
	TestAnnotation = "@test"

	// BeforeTestAnnotation used to label a function as a before test hook
	BeforeTestAnnotation = "@beforeTest"

	// AfterTestAnnotation used to label a function as a after test hook
	AfterTestAnnotation = "@afterTest"
)

var (
	fixtureRegexp     = annotationRegexp(FixtureAnnotation)
	onceFixtureRegexp = annotationRegexp(OnceFixtureAnnotation)
	testRegexp        = annotationRegexp(TestAnnotation)
	beforeTestRegexp  = annotationRegexp(BeforeTestAnnotation)
	afterTestRegexp   = annotationRegexp(AfterTestAnnotation)
)

func annotationRegexp(annotation string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprint(`(^|\n)\s*`, annotation, `\s+`))
}

// ParseResult holds the package and functions parsed.
type ParseResult struct {
	Package      *ast.Package
	Fixtures     []*Function
	OnceFixtures []*Function
	Tests        []*Function
	BeforeTests  []*Function
	AfterTests   []*Function
}

// Parse returns the parsed result of the package.
func Parse(pkgDir string, filePrefix string) (*ParseResult, error) {
	fns, err := parseFunctions(pkgDir, filePrefix)
	if err != nil {
		return nil, err
	}

	res := &ParseResult{}

	for _, fn := range fns {
		if res.Package == nil {
			res.Package = fn.Package
		}

		switch {
		case fn.HasTestAnnotation():
			res.Tests = append(res.Tests, fn)
		case fn.HasFixtureAnnotation():
			res.Fixtures = append(res.Fixtures, fn)
		case fn.HasOnceFixtureAnnotation():
			res.OnceFixtures = append(res.OnceFixtures, fn)
		case fn.HasBeforeTestAnnotation():
			res.BeforeTests = append(res.BeforeTests, fn)
		case fn.HasAfterTestAnnotation():
			res.AfterTests = append(res.AfterTests, fn)
		}
	}

	return res, nil
}

// Function represents a function declaration.
type Function struct {
	File    string
	Package *ast.Package
	Decl    *ast.FuncDecl
}

func (f *Function) String() string {
	return fmt.Sprintf(`{"file":"%s","line":%d,"name":"%s", "comment":"%s"}`, f.File, f.Decl.Pos(), f.Decl.Name, f.Decl.Doc.Text())
}

// HasTestAnnotation returns true if the function has a test annotation.
func (f *Function) HasTestAnnotation() bool {
	return f.commentMatches(testRegexp)
}

// HasFixtureAnnotation returns true if the function has a fixture annotation.
func (f *Function) HasFixtureAnnotation() bool {
	return f.commentMatches(fixtureRegexp)
}

// HasOnceFixtureAnnotation returns true if the function has a onceFixture annotation.
func (f *Function) HasOnceFixtureAnnotation() bool {
	return f.commentMatches(onceFixtureRegexp)
}

// HasBeforeTestAnnotation returns true if the function has a beforeTest annotation.
func (f *Function) HasBeforeTestAnnotation() bool {
	return f.commentMatches(beforeTestRegexp)
}

// HasAfterTestAnnotation returns true if the function has a afterTest annotation.
func (f *Function) HasAfterTestAnnotation() bool {
	return f.commentMatches(afterTestRegexp)
}

func (f *Function) commentMatches(regex *regexp.Regexp) bool {
	return regex.MatchString(f.Decl.Doc.Text())
}

func parseFunctions(pkg string, filePrefix string) ([]*Function, error) {
	fset := token.NewFileSet()

	pkgs, err := parser.ParseDir(fset, pkg, func(fi os.FileInfo) bool {
		return strings.HasSuffix(fi.Name(), filePrefix)
	}, parser.ParseComments)

	if err != nil {
		return nil, err
	}

	var fns []*Function

	for pkgName := range pkgs {
		pkg := pkgs[pkgName]
		for fileName, file := range pkg.Files {

			for _, decl := range file.Decls {
				switch decl := decl.(type) {
				case *ast.FuncDecl:
					fns = append(fns, &Function{Package: pkg, File: fileName, Decl: decl})
				}
			}
		}
	}

	return fns, nil
}
