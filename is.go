// Package is provides a lightweight extension to the
// standard library's testing capabilities.
//
// Comments on the assertion lines are used to add
// a description.
//
// The following failing test:
//
//	func Test(t *testing.T) {
//		is := is.New(t)
//		a, b := 1, 2
//		is.Equal(a, b) // expect to be the same
//	}
//
// Will output:
//
//		your_test.go:123: 1 != 2 // expect to be the same
//
// Usage
//
// The following code shows a range of useful ways you can use
// the helper methods:
//
//	func Test(t *testing.T) {
//
//		// always start tests with this
//		is := is.New(t)
//
//		signedin, err := isSignedIn(ctx)
//		is.NoErr(err)            // isSignedIn error
//		is.Equal(signedin, true) // must be signed in
//
//		body := readBody(r)
//		is.True(strings.Contains(body, "Hi there"))
//
//	}
package is

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// T reports when failures occur.
// testing.T implements this interface.
type T interface {
	// Fail indicates that the test has failed but
	// allowed execution to continue.
	// Fail is called in relaxed mode (via NewRelaxed).
	Fail()
	// FailNow indicates that the test has failed and
	// aborts the test.
	// FailNow is called in strict mode (via New).
	FailNow()
}

// I is the test helper harness.
type I struct {
	t        T
	fail     func()
	out      io.Writer
	colorful bool
}

type cacheEntryType string

const commentCacheEntry cacheEntryType = "Comment"
const isCallCacheEntry cacheEntryType = "IsCall"

var errNoCallerInfoFound = errors.New("could not find args")

// global fileset
var fset = token.NewFileSet()

// astCache is a map of file[CacheEntryType:linenumber[Node]]
var astCache = make(map[string]map[string]ast.Node)

var noColorFlag bool

func init() {
	flag.BoolVar(&noColorFlag, "nocolor", false, "turns off colors")
}

// New makes a new testing helper using the specified
// T through which failures will be reported.
// In strict mode, failures call T.FailNow causing the test
// to be aborted. See NewRelaxed for alternative behavior.
func New(t T) *I {
	return &I{t, t.FailNow, os.Stdout, !noColorFlag}
}

// NewRelaxed makes a new testing helper using the specified
// T through which failures will be reported.
// In relaxed mode, failures call T.Fail allowing
// multiple failures per test.
func NewRelaxed(t T) *I {
	return &I{t, t.Fail, os.Stdout, !noColorFlag}
}

func (is *I) log(args ...interface{}) {
	s := is.decorate(fmt.Sprint(args...))
	fmt.Fprintf(is.out, s)
	is.fail()
}

func (is *I) logf(format string, args ...interface{}) {
	is.log(fmt.Sprintf(format, args...))
}

// Fail immediately fails the test.
//
//	func Test(t *testing.T) {
//		is := is.New(t)
//		is.Fail() // TODO: write this test
//	}
//
// In relaxed mode, execution will continue after a call to
// Fail, but that test will still fail.
func (is *I) Fail() {
	is.log("failed")
}

// True asserts that the expression is true. The expression
// code itself will be reported if the assertion fails.
//
//	func Test(t *testing.T) {
//		is := is.New(t)
//		val := method()
//		is.True(val != nil) // val should never be nil
//	}
//
// Will output:
//
//	your_test.go:123: not true: val != nil
func (is *I) True(expression bool) {
	if !expression {
		is.log("not true: $ARGS")
	}
}

// Equal asserts that a and b are equal.
//
//	func Test(t *testing.T) {
//		is := is.New(t)
//		a := greet("Mat")
//		is.Equal(a, "Hi Mat") // greeting
//	}
//
// Will output:
//
//	your_test.go:123: Hey Mat != Hi Mat // greeting
func (is *I) Equal(a, b interface{}) {
	if areEqual(a, b) {
		return
	}

	argNames, _ := getArgNames()
	aName := getElementFrom(argNames, 0, "")
	bName := getElementFrom(argNames, 1, "")
	aType := fmt.Sprintf("%T", a)
	bType := fmt.Sprintf("%T", b)
	aValue := formatValue(a)
	bValue := formatValue(b)

	if !(isNil(a) || isNil(b)) && reflect.ValueOf(a).Type() != reflect.ValueOf(b).Type() {
		aValue = fmt.Sprintf("%s(%s)", aType, aValue)
		bValue = fmt.Sprintf("%s(%s)", bType, bValue)
	}

	if aValue != aName {
		if is.colorful {
			aValue = fmt.Sprintf("%s%s(%s)%s", aName, colorType, aValue, colorNormal)
		} else {
			aValue = fmt.Sprintf("%s(%s)", aName, aValue)
		}
	}

	if bValue != bName {
		if is.colorful {
			bValue = fmt.Sprintf("%s%s(%s)%s", bName, colorType, bValue, colorNormal)
		} else {
			bValue = fmt.Sprintf("%s(%s)", bName, bValue)
		}
	}

	// if bValue != bName {
	// 	bValue = fmt.Sprintf("%s<%s>", bName, bValue)
	// }

	is.logf("%s != %s", aValue, bValue)

}

func formatValue(object interface{}) string {
	if isNil(object) {
		return "nil"
	}

	value := reflect.ValueOf(object)
	kind := value.Kind()
	if kind == reflect.Map {
		buf := new(bytes.Buffer)
		for _, mapKey := range value.MapKeys() {
			mapVal := value.MapIndex(mapKey)
			mapValStr := formatValue(mapVal.Interface())
			mapKeyStr := fmt.Sprintf("%v", mapKey.Interface())
			buf.WriteString(fmt.Sprintf("%s:%s ", mapKeyStr, mapValStr))
		}

		result := strings.TrimSpace(buf.String())
		return result
	}
	if kind == reflect.Slice || kind == reflect.Array {
		buf := new(bytes.Buffer)
		for i := 0; i < value.Len(); i++ {
			sliceVal := value.Index(i)
			sliceValStr := formatValue(sliceVal.Interface())
			buf.WriteString(fmt.Sprintf("%s ", sliceValStr))
		}

		// fullStr = fmt.Sprintf("[%s]", strings.TrimSpace(fullStr))
		result := strings.TrimSpace(buf.String())
		return result
	}

	return fmt.Sprint(object)
}

func getElementFrom(array []string, at int, orDefault string) string {
	if array == nil || len(array) <= at {
		return orDefault
	}

	result := array[at]
	return result
}

// func nameValueLabel(allArgNames []string, argValue interface{}, argIndex int){
// 	argValueStr := ""
// 	if isNil(argValue) {
// 		argValue = "nil"
// 	}
// 	if
// 	if allArgNames == nil || len(allArgNames) <= argIndex{

// 	}
// }

// New is a method wrapper around the New function.
// It allows you to write subtests using a similar
// pattern:
//
//	func Test(t *testing.T) {
//		is := is.New(t)
//		t.Run("sub", func(t *testing.T) {
//			is := is.New(t)
//			// TODO: test
//		})
//	}
func (is *I) New(t *testing.T) *I {
	return New(t)
}

// NewRelaxed is a method wrapper around the NewRelaxed
// method. It allows you to write subtests using a similar
// pattern:
//
//	func Test(t *testing.T) {
//		is := is.NewRelaxed(t)
//		t.Run("sub", func(t *testing.T) {
//			is := is.NewRelaxed(t)
//			// TODO: test
//		})
//	}
func (is *I) NewRelaxed(t *testing.T) *I {
	return NewRelaxed(t)
}

func (is *I) valWithType(v interface{}) string {
	if is.colorful {
		return fmt.Sprintf("%[1]s%[3]T(%[2]s%[3]v%[1]s)%[2]s", colorType, colorNormal, v)
	}
	return fmt.Sprintf("%[1]T(%[1]v)", v)
}

func findAssignStmtFromIdent(ident *ast.Ident) *ast.AssignStmt {
	if ident.Obj == nil {
		return nil
	}
	if ident.Obj.Decl == nil {
		return nil
	}

	result := ident.Obj.Decl.(*ast.AssignStmt)
	return result
}

// NoErr asserts that err is nil.
//
//	func Test(t *testing.T) {
//		is := is.New(t)
//		val, err := getVal()
//		is.NoErr(err)        // getVal error
//		is.True(len(val) > 10) // val cannot be short
//	}
//
// Will output:
//
//	your_test.go:123: err: not found // getVal error
func (is *I) NoErr(err error) {
	if err == nil {
		return
	}

	args, argsErr := getArgExprs()
	if argsErr != nil || args == nil || len(args) <= 0 {
		is.logf("%v", err)
		return
	}

	argNames, _ := getArgNames()
	errSrc := getElementFrom(argNames, 0, "")

	argExpr := args[0]
	errVar, ok := argExpr.(*ast.Ident)
	if ok && errVar.Obj.Decl != nil {
		// try to find the declaration of the err var
		errVarAssignment := findAssignStmtFromIdent(errVar)
		if errVarAssignment != nil {
			errSrc = nodeToStr(fset, errVarAssignment)

		}
	}

	errStr := ""
	if is.colorful {
		errStr = fmt.Sprintf("error: %s%s(%s)%s", formatValue(err), colorType, errSrc, colorNormal)
	} else {
		errStr = fmt.Sprintf("error: %s(%s)", formatValue(err), errSrc)
	}

	is.logf(errStr)

}

// isNil gets whether the object is nil or not.
func isNil(object interface{}) bool {
	if object == nil {
		return true
	}
	value := reflect.ValueOf(object)
	kind := value.Kind()
	if kind >= reflect.Chan && kind <= reflect.Slice && value.IsNil() {
		return true
	}
	return false
}

// areEqual gets whether a equals b or not.
func areEqual(a, b interface{}) bool {
	if isNil(a) || isNil(b) {
		if isNil(a) && !isNil(b) {
			return false
		}
		if !isNil(a) && isNil(b) {
			return false
		}
		return true
	}
	if reflect.DeepEqual(a, b) {
		return true
	}
	aValue := reflect.ValueOf(a)
	bValue := reflect.ValueOf(b)
	return aValue == bValue
}

func getArgExprs() ([]ast.Expr, error) {
	path, lineNumber, ok := callerinfo()
	if !ok {
		return nil, errNoCallerInfoFound
	}

	node, err := getNodeFromCache(isCallCacheEntry, path, lineNumber)
	if err != nil {
		return nil, err
	}

	callExpr, ok := node.(*ast.CallExpr)
	if !ok {
		return nil, fmt.Errorf("not a call expr line %d in file %s", lineNumber, path)
	}

	return callExpr.Args, nil
}

func getArgNames() ([]string, error) {
	args, err := getArgExprs()
	if err != nil {
		return nil, err
	}

	result := []string{}
	for _, arg := range args {
		argStr := nodeToStr(fset, arg)
		result = append(result, argStr)
	}

	return result, nil
}

func callerinfo() (path string, line int, ok bool) {
	for i := 0; ; i++ {
		_, path, line, ok = runtime.Caller(i)
		if !ok {
			return
		}
		if strings.HasSuffix(path, "is.go") {
			continue
		}
		return path, line, true
	}
}

func getFileCache(path string) (map[string]ast.Node, bool) {
	file, ok := astCache[path]
	if ok {
		return file, false
	}

	result := make(map[string]ast.Node)
	astCache[path] = result
	return result, true

}

func getNodeFromCache(kind cacheEntryType, path string, line int) (ast.Node, error) {
	key := fmt.Sprintf("%s:%d", kind, line)
	fileCache, newCacheEntry := getFileCache(path)
	entry, ok := fileCache[key]
	if ok {
		return entry, nil
	}

	if !newCacheEntry {
		return nil, fmt.Errorf("key %s not found in cache", key)
	}

	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	for _, comment := range f.Comments {
		pos := fset.Position(comment.Pos())
		if !pos.IsValid() {
			continue
		}

		key := fmt.Sprintf("%s:%d", commentCacheEntry, pos.Line)
		fileCache[key] = comment
	}

	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			return false
		}

		pos := fset.Position(n.Pos())

		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		ident, ok := selExpr.X.(*ast.Ident)
		if !ok {
			return true
		}

		if ident.Name != "is" {
			return true
		}

		key := fmt.Sprintf("%s:%d", isCallCacheEntry, pos.Line)
		fileCache[key] = callExpr
		return false
	})

	entry, ok = fileCache[key]
	if !ok {
		return nil, fmt.Errorf("key %s not found in cache", key)
	}

	return entry, nil
}

func nodeToStr(fset *token.FileSet, node ast.Node) string {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		panic(err)
	}

	return string(buf.Bytes())
}

func formatCallExprArgs(fset *token.FileSet, callExpr *ast.CallExpr) string {
	selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}

	result := ""
	if selExpr.Sel.Name == "True" {
		// true has only one arg
		arg := callExpr.Args[0]
		result = nodeToStr(fset, arg)
	}

	// only true is currentley supported
	return result
}

// loadComment gets the Go comment from the specified line
// in the specified file.
func loadComment(path string, line int) (string, bool) {
	node, err := getNodeFromCache(commentCacheEntry, path, line)
	if err != nil {
		return "", false
	}

	comment, ok := node.(*ast.CommentGroup)
	if !ok {
		return "", false
	}

	result := strings.TrimSpace(comment.Text())
	return result, true
}

// loadArguments gets the arguments from the function call
// on the specified line of the file.
func loadArguments(path string, line int) (string, bool) {
	node, err := getNodeFromCache(isCallCacheEntry, path, line)
	if err != nil {
		return "", false
	}

	callExpr, ok := node.(*ast.CallExpr)
	if !ok {
		return "", false
	}

	argStr := formatCallExprArgs(fset, callExpr)
	if argStr == "" {
		return "", false
	}

	return argStr, true
}

// decorate prefixes the string with the file and line of the call site
// and inserts the final newline if needed and indentation tabs for formatting.
// this function was copied from the testing framework and modified.
func (is *I) decorate(s string) string {
	path, lineNumber, ok := callerinfo() // decorate + log + public function.
	file := filepath.Base(path)
	if ok {
		// Truncate file name at last file name separator.
		if index := strings.LastIndex(file, "/"); index >= 0 {
			file = file[index+1:]
		} else if index = strings.LastIndex(file, "\\"); index >= 0 {
			file = file[index+1:]
		}
	} else {
		file = "???"
		lineNumber = 1
	}
	buf := new(bytes.Buffer)
	// Every line is indented at least one tab.
	buf.WriteByte('\t')
	if is.colorful {
		buf.WriteString(colorFile)
	}
	fmt.Fprintf(buf, "%s:%d: ", file, lineNumber)
	if is.colorful {
		buf.WriteString(colorNormal)
	}
	lines := strings.Split(s, "\n")
	if l := len(lines); l > 1 && lines[l-1] == "" {
		lines = lines[:l-1]
	}
	for i, line := range lines {
		if i > 0 {
			// Second and subsequent lines are indented an extra tab.
			buf.WriteString("\n\t\t")
		}
		// expand arguments (if $ARGS is present)
		if strings.Contains(line, "$ARGS") {
			args, _ := loadArguments(path, lineNumber)
			line = strings.Replace(line, "$ARGS", args, -1)
		}
		buf.WriteString(line)
	}
	comment, ok := loadComment(path, lineNumber)
	if ok {
		if is.colorful {
			buf.WriteString(colorComment)
		}
		buf.WriteString(" // ")
		buf.WriteString(comment)
		if is.colorful {
			buf.WriteString(colorNormal)
		}
	}
	buf.WriteString("\n")
	return buf.String()
}

const (
	colorNormal  = "\u001b[39m"
	colorComment = "\u001b[32m"
	colorFile    = "\u001b[90m"
	colorType    = "\u001b[90m"
)
