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

	DoAnnotation = "@doOnce"

	// TestAnnotation used to label a function as a test
	TestAnnotation = "@test"

	// BeforeTestAnnotation used to label a function as a before test hook
	BeforeTestAnnotation = "@beforeTest"

	// AfterTestAnnotation used to label a function as a after test hook
	AfterTestAnnotation = "@afterTest"

	// TestLabelAnnotation used to introduce a new test Label.
	TestLabelAnnotation = "@testLabel"

	// DefaultTestLabelAnnotation used to set the default test label.
	DefaultTestLabelAnnotation = "@defaultTestLabel"

	defaultTestLabel = "test"
)

var (
	fixtureRegexp          = annotationRegexp(FixtureAnnotation)
	onceFixtureRegexp      = annotationRegexp(OnceFixtureAnnotation)
	testRegexp             = annotationWithOptionalParamsRegexp(TestAnnotation)
	beforeTestRegexp       = annotationWithOptionalParamsRegexp(BeforeTestAnnotation)
	afterTestRegexp        = annotationWithOptionalParamsRegexp(AfterTestAnnotation)
	testLabelRegexp        = annotationWithParamsRegexp(TestLabelAnnotation)
	defaultTestLabelRegexp = annotationWithParamsRegexp(DefaultTestLabelAnnotation)
)

func annotationRegexp(annotation string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprint(`(^|\n)\s*`, annotation, `\s*($|\n)`))
}

func annotationWithParamsRegexp(annotation string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprint(`(?:^|\n)\s*`, annotation, `\((\w+(?:,\s*\w+)*)\)\s*(?:$|\n)`))
}

func annotationWithOptionalParamsRegexp(annotation string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprint(`(?:^|\n)\s*`, annotation, `(?:\((\w+(?:,\s*\w+)*)\))?\s*(?:$|\n)`))
}

func getParams(annotation *regexp.Regexp, cmt string) ([]string, bool) {
	res := annotation.FindStringSubmatch(cmt)
	if len(res) != 2 {
		return nil, false
	}
	if len(res[1]) > 0 {
		res = strings.Split(res[1], ",")
		for i := range res {
			res[i] = strings.TrimSpace(strings.TrimLeft(res[i], ","))
		}
	} else {
		res = nil
	}
	return res, true
}

// ParseResult holds the package and functions parsed.
type ParseResult struct {
	Package *ast.Package

	DefaultTestLabel string
	// Label name => prefix.
	TestLabels   map[string][]string
	Fixtures     []*Function
	OnceFixtures []*Function
	Tests        []*LabelFunction
	BeforeTests  []*LabelFunction
	AfterTests   []*LabelFunction

	Warnings []string
}

type LabelFunction struct {
	*Function
	Labels []string
}

// Parse returns the parsed result of the package.
func Parse(pkgDir string, filePrefix string, autoLabel bool) (*ParseResult, error) {
	parseResult, err := parsePackage(pkgDir, filePrefix)
	if err != nil {
		return nil, err
	}

	res := &ParseResult{
		DefaultTestLabel: defaultTestLabel,
		TestLabels:       map[string][]string{},
	}

	for _, cmt := range parseResult.comments {
		if testLabelRegexp.MatchString(cmt) {
			params, ok := getParams(testLabelRegexp, cmt)
			if !ok {
				res.Warnings = append(res.Warnings, fmt.Sprintf("@testLabel parameters could not be parsed '%s'", cmt))
				continue
			}

			if len(params) < 1 {
				res.Warnings = append(res.Warnings, fmt.Sprintf("@testLabel must have one argument '%s'", cmt))
				continue
			}

			Label := params[0]
			res.TestLabels[Label] = append(res.TestLabels[Label], params[1:]...)
		}

		if defaultTestLabelRegexp.MatchString(cmt) {
			params, ok := getParams(defaultTestLabelRegexp, cmt)
			if !ok {
				res.Warnings = append(res.Warnings, fmt.Sprintf("@defaultTestLabel parameters could not be parsed '%s'", cmt))
				continue
			}

			if len(params) > 1 {
				res.Warnings = append(res.Warnings, fmt.Sprintf("@defaultTestLabel can only have one argument '%s'", cmt))
				continue
			}

			res.DefaultTestLabel = params[0]
		}
	}

	// Ensure that the default test label exists.
	if _, ok := res.TestLabels[res.DefaultTestLabel]; !ok {
		res.TestLabels[res.DefaultTestLabel] = nil
	}

	parseLabels := func(regex *regexp.Regexp, fn *Function, defaultLabels []string) ([]string, bool) {
		labels, ok := getParams(regex, fn.Comment())
		if !ok {
			return nil, ok
		}
		if len(labels) == 0 {
			labels = defaultLabels
		} else {
			for _, label := range labels {
				if _, ok := res.TestLabels[label]; !ok {
					res.TestLabels[label] = nil
				}
			}
		}

		return labels, ok
	}

	for _, fn := range parseResult.functions {
		if res.Package == nil {
			res.Package = fn.Package
		}

		// Check function annotations
		switch {
		case fn.HasTestAnnotation():
			labels, ok := parseLabels(testRegexp, fn, []string{res.DefaultTestLabel})
			if ok {
				res.Tests = append(res.Tests, &LabelFunction{Function: fn, Labels: labels})
			} else {
				res.Warnings = append(res.Warnings, fmt.Sprintf("@test parameters could not be parsed '%s'", fn.Comment()))
			}
			continue
		case fn.HasFixtureAnnotation():
			res.Fixtures = append(res.Fixtures, fn)
			continue
		case fn.HasOnceFixtureAnnotation():
			res.OnceFixtures = append(res.OnceFixtures, fn)
			continue
		case fn.HasBeforeTestAnnotation():
			labels, ok := parseLabels(beforeTestRegexp, fn, []string{res.DefaultTestLabel})
			if ok {
				res.BeforeTests = append(res.BeforeTests, &LabelFunction{Function: fn, Labels: labels})
			} else {
				res.Warnings = append(res.Warnings, fmt.Sprintf("@beforeTest parameters could not be parsed '%s'", fn.Comment()))
			}
			continue
		case fn.HasAfterTestAnnotation():
			labels, ok := parseLabels(afterTestRegexp, fn, []string{res.DefaultTestLabel})
			if ok {
				res.AfterTests = append(res.AfterTests, &LabelFunction{Function: fn, Labels: labels})
			} else {
				res.Warnings = append(res.Warnings, fmt.Sprintf("@afterTest parameters could not be parsed '%s'", fn.Comment()))
			}
			continue
		}

		// Check auto grouping
		var labels []string
		for label, prefixes := range res.TestLabels {
			for _, prefix := range prefixes {
				if strings.HasPrefix(fn.Name(), prefix) {
					labels = append(labels, label)
				}
			}
		}
		if len(labels) > 0 {
			res.Tests = append(res.Tests, &LabelFunction{Function: fn, Labels: labels})
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
	return regex.MatchString(f.Comment())
}

func (f *Function) Name() string {
	return f.Decl.Name.String()
}

func (f *Function) Comment() string {
	return f.Decl.Doc.Text()
}

type parseResult struct {
	functions []*Function
	comments  []string
}

func parsePackage(pkg string, filePrefix string) (*parseResult, error) {
	fset := token.NewFileSet()

	pkgs, err := parser.ParseDir(fset, pkg, func(fi os.FileInfo) bool {
		return strings.HasSuffix(fi.Name(), filePrefix)
	}, parser.ParseComments)

	if err != nil {
		return nil, err
	}

	res := &parseResult{}

	for pkgName := range pkgs {
		pkg := pkgs[pkgName]
		for fileName, file := range pkg.Files {

			for _, decl := range file.Decls {
				switch decl := decl.(type) {
				case *ast.FuncDecl:
					res.functions = append(res.functions, &Function{Package: pkg, File: fileName, Decl: decl})
				}
			}
			for _, cmt := range file.Comments {
				res.comments = append(res.comments, cmt.Text())
			}
		}
	}

	return res, nil
}
