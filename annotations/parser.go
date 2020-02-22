package annotations

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strings"
	"unicode"
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

	// TestLabelAnnotation used to introduce a new test Label.
	TestLabelAnnotation = "@testLabel"

	// DisableAutoLabellingAnnotation can be used to toggle the auto matching of tests.
	DisableAutoLabellingAnnotation = "@disableAutoLabelling"

	unitTestLabel        = "unit"
	regressionTestLabel  = "regression"
	integrationTestLabel = "integration"

	DefaultTestLabel = unitTestLabel
)

var (
	unitTestMatcher        = []string{"test", "unit"}
	regressionTestMatcher  = []string{"reg", "regression"}
	integrationTestMatcher = []string{"int", "integration"}
	fixtureMatcher         = []string{"fix", "fixture"}
	beforeTestMatcher      = []string{"pre", "beforeTest"}
	afterTestMatcher       = []string{"post", "afterTest"}
)

var (
	fixtureRegexp              = annotationRegexp(FixtureAnnotation)
	onceFixtureRegexp          = annotationRegexp(OnceFixtureAnnotation)
	testRegexp                 = annotationWithOptionalParamsRegexp(TestAnnotation)
	beforeTestRegexp           = annotationRegexp(BeforeTestAnnotation)
	afterTestRegexp            = annotationRegexp(AfterTestAnnotation)
	testLabelRegexp            = annotationWithParamsRegexp(TestLabelAnnotation)
	disableAutoLabellingRegexp = annotationRegexp(DisableAutoLabellingAnnotation)
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
	BeforeTests  []*Function
	AfterTests   []*Function

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

	return parse(parseResult, autoLabel)
}

func parse(parseResult *parseResult, autoLabel bool) (*ParseResult, error) {
	res := &ParseResult{
		DefaultTestLabel: DefaultTestLabel,
		TestLabels: map[string][]string{
			unitTestLabel:        unitTestMatcher,
			integrationTestLabel: integrationTestMatcher,
			regressionTestLabel:  regressionTestMatcher,
		},
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

		if disableAutoLabellingRegexp.MatchString(cmt) {
			autoLabel = false
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

funcLoop:
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
			continue funcLoop
		case fn.HasFixtureAnnotation():
			res.Fixtures = append(res.Fixtures, fn)
			continue funcLoop
		case fn.HasOnceFixtureAnnotation():
			res.OnceFixtures = append(res.OnceFixtures, fn)
			continue funcLoop
		case fn.HasBeforeTestAnnotation():
			res.BeforeTests = append(res.BeforeTests, fn)
			continue funcLoop
		case fn.HasAfterTestAnnotation():
			res.AfterTests = append(res.AfterTests, fn)
			continue funcLoop
		}

		if autoLabel {
			// Check auto grouping
			for _, prefix := range fixtureMatcher {
				if prefixMatch(fn.Name(), prefix) {
					res.Fixtures = append(res.Fixtures, fn)
					continue funcLoop
				}
			}

			for _, prefix := range beforeTestMatcher {
				if prefixMatch(fn.Name(), prefix) {
					res.BeforeTests = append(res.BeforeTests, fn)
					continue funcLoop
				}
			}

			for _, prefix := range afterTestMatcher {
				if prefixMatch(fn.Name(), prefix) {
					res.AfterTests = append(res.AfterTests, fn)
					continue funcLoop
				}
			}

			var labels []string
		labelLoop:
			for label, prefixes := range res.TestLabels {
				for _, prefix := range prefixes {
					if strings.HasPrefix(fn.Name(), prefix) {
						labels = append(labels, label)
						continue labelLoop
					}
				}
			}
			if len(labels) > 0 {
				res.Tests = append(res.Tests, &LabelFunction{Function: fn, Labels: labels})
			}
		}
	}

	return res, nil
}

func prefixMatch(str, prefix string) bool {
	if !strings.HasPrefix(str, prefix) {
		return false
	}
	if len(str) == len(prefix) {
		return true
	}
	c := rune(str[len(prefix)])
	return c == '_' || unicode.IsUpper(c)
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
