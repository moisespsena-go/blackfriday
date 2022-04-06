package parser_test

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	. "github.com/russross/blackfriday/v2/table_header/parser"
	"github.com/russross/blackfriday/v2/table_header/require"
)

func TestParserError(t *testing.T) {
	err := &Error{Pos: SourceFilePos{
		Offset: 10, Line: 1, Column: 10,
	}, Msg: "test"}
	require.Equal(t, "Parse Error: test\n\tat 1:10", err.Error())
}

func TestParserErrorList(t *testing.T) {
	var list ErrorList
	list.Add(SourceFilePos{Offset: 20, Line: 2, Column: 10}, "error 2")
	list.Add(SourceFilePos{Offset: 30, Line: 3, Column: 10}, "error 3")
	list.Add(SourceFilePos{Offset: 10, Line: 1, Column: 10}, "error 1")
	list.Sort()
	require.Equal(t, "Parse Error: error 1\n\tat 1:10 (and 2 more errors)",
		list.Error())
}

func TestParseString(t *testing.T) {
	expectParse(t, `{foo[abc]}`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(valuesLit(1, 5,
				valuesElementLit("foo", p(1, 2), 0, nil),
			)),
		)
	})
	return
	expectParse(t, `{foo}`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(valuesLit(1, 5,
				valuesElementLit("foo", p(1, 2), 0, nil),
			)),
		)
	})
	expectParse(t, `{"foo bar","x":"z"}`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(valuesLit(1, 19,
				valuesElementLit("foo bar", p(1, 2), 0, nil),
				valuesElementLit("x", p(1, 12), 15, stringLit("z", 16)),
			)),
		)
	})
	expectParse(t, `{"foo bar"}`, func(p pfn) []Stmt {
		return stmts(
			exprStmt(valuesLit(1, 11,
				valuesElementLit("foo bar", p(1, 2), 0, nil),
			)),
		)
	})
}

type pfn func(int, int) Pos          // position conversion function
type expectedFn func(pos pfn) []Stmt // callback function to return expected results

type parseTracer struct {
	out []string
}

func (o *parseTracer) Write(p []byte) (n int, err error) {
	o.out = append(o.out, string(p))
	return len(p), nil
}

// type slowPrinter struct {
// }
//
// func (o *slowPrinter) Write(p []byte) (n int, err error) {
//	fmt.Print(string(p))
//	time.Sleep(25 * time.Millisecond)
//	return len(p), nil
// }

func expectParse(t *testing.T, input string, fn expectedFn) {
	testFileSet := NewFileSet()
	testFile := testFileSet.AddFile("test", -1, len(input))

	var ok bool
	defer func() {
		if !ok {
			// print trace
			tr := &parseTracer{}
			p := NewParser(testFile, []byte(input), tr)
			actual, _ := p.ParseFile()
			if actual != nil {
				t.Logf("Parsed:\n%s", actual.String())
			}
			t.Logf("Trace:\n%s", strings.Join(tr.out, ""))
		}
	}()

	p := NewParser(testFile, []byte(input), nil)
	actual, err := p.ParseFile()
	require.NoError(t, err)

	expected := fn(func(line, column int) Pos {
		return Pos(int(testFile.LineStart(line)) + (column - 1))
	})
	require.Equal(t, len(expected), len(actual.Stmts))

	for i := 0; i < len(expected); i++ {
		equalStmt(t, expected[i], actual.Stmts[i])
	}

	ok = true
}

func expectParseError(t *testing.T, input string) {
	testFileSet := NewFileSet()
	testFile := testFileSet.AddFile("test", -1, len(input))

	var ok bool
	defer func() {
		if !ok {
			// print trace
			tr := &parseTracer{}
			p := NewParser(testFile, []byte(input), tr)
			_, _ = p.ParseFile()
			t.Logf("Trace:\n%s", strings.Join(tr.out, ""))
		}
	}()

	p := NewParser(testFile, []byte(input), nil)
	_, err := p.ParseFile()
	require.Error(t, err)
	ok = true
}

func expectParseString(t *testing.T, input, expected string) {
	var ok bool
	defer func() {
		if !ok {
			// print trace
			tr := &parseTracer{}
			_, _ = parseSource("test", []byte(input), tr)
			t.Logf("Trace:\n%s", strings.Join(tr.out, ""))
		}
	}()

	actual, err := parseSource("test", []byte(input), nil)
	require.NoError(t, err)
	require.Equal(t, expected, actual.String())
	ok = true
}

func stmts(s ...Stmt) []Stmt {
	return s
}

func exprStmt(x Expr) *ExprStmt {
	return &ExprStmt{Expr: x}
}

func emptyStmt(implicit bool, pos Pos) *EmptyStmt {
	return &EmptyStmt{Implicit: implicit, Semicolon: pos}
}

func blockStmt(lbrace, rbrace Pos, list ...Stmt) *BlockStmt {
	return &BlockStmt{Stmts: list, LBrace: lbrace, RBrace: rbrace}
}

func ident(name string, pos Pos) *Ident {
	return &Ident{Name: name, NamePos: pos}
}

func exprs(list ...Expr) []Expr {
	return list
}

func stringLit(value string, pos Pos) *StringLit {
	return &StringLit{Value: value, ValuePos: pos}
}

func arrayLit(lbracket, rbracket Pos, list ...Expr) *ArrayLit {
	return &ArrayLit{LBrack: lbracket, RBrack: rbracket, Elements: list}
}

func valuesElementLit(
	key string,
	keyPos Pos,
	colonPos Pos,
	value Expr,
) *ValuesElementLit {
	return &ValuesElementLit{
		Key: key, KeyPos: keyPos, ColonPos: colonPos, Value: value,
	}
}

func valuesLit(
	lbrace, rbrace Pos,
	list ...*ValuesElementLit,
) *ValuesLit {
	return &ValuesLit{LBrace: lbrace, RBrace: rbrace, Elements: list}
}

func sliceExpr(
	x, low, high Expr,
	lbrack, rbrack Pos,
) *SliceExpr {
	return &SliceExpr{
		Expr: x, Low: low, High: high, LBrack: lbrack, RBrack: rbrack,
	}
}

func equalStmt(t *testing.T, expected, actual Stmt) {
	if expected == nil || reflect.ValueOf(expected).IsNil() {
		require.Nil(t, actual, "expected nil, but got not nil")
		return
	}
	require.NotNil(t, actual, "expected not nil, but got nil")
	require.IsType(t, expected, actual)

	switch expected := expected.(type) {
	case *ExprStmt:
		equalExpr(t, expected.Expr, actual.(*ExprStmt).Expr)
	case *EmptyStmt:
		require.Equal(t, expected.Implicit,
			actual.(*EmptyStmt).Implicit)
		require.Equal(t, expected.Semicolon,
			actual.(*EmptyStmt).Semicolon)
	case *BlockStmt:
		require.Equal(t, expected.LBrace,
			actual.(*BlockStmt).LBrace)
		require.Equal(t, expected.RBrace,
			actual.(*BlockStmt).RBrace)
		equalStmts(t, expected.Stmts,
			actual.(*BlockStmt).Stmts)
	case *BranchStmt:
		equalExpr(t, expected.Label,
			actual.(*BranchStmt).Label)
		require.Equal(t, expected.Token,
			actual.(*BranchStmt).Token)
		require.Equal(t, expected.TokenPos,
			actual.(*BranchStmt).TokenPos)
	default:
		panic(fmt.Errorf("unknown type: %T", expected))
	}
}

func equalExpr(t *testing.T, expected, actual Expr) {
	if expected == nil || reflect.ValueOf(expected).IsNil() {
		require.Nil(t, actual, "expected nil, but got not nil")
		return
	}
	require.NotNil(t, actual, "expected not nil, but got nil")
	require.IsType(t, expected, actual)

	switch expected := expected.(type) {
	case *Ident:
		require.Equal(t, expected.Name,
			actual.(*Ident).Name)
		require.Equal(t, int(expected.NamePos),
			int(actual.(*Ident).NamePos))
	case *StringLit:
		require.Equal(t, expected.Value,
			actual.(*StringLit).Value)
		require.Equal(t, int(expected.ValuePos),
			int(actual.(*StringLit).ValuePos))
	case *ArrayLit:
		require.Equal(t, expected.LBrack,
			actual.(*ArrayLit).LBrack)
		require.Equal(t, expected.RBrack,
			actual.(*ArrayLit).RBrack)
		equalExprs(t, expected.Elements,
			actual.(*ArrayLit).Elements)
	case *ValuesLit:
		require.Equal(t, expected.LBrace,
			actual.(*ValuesLit).LBrace)
		require.Equal(t, expected.RBrace,
			actual.(*ValuesLit).RBrace)
		equalValuesElements(t, expected.Elements,
			actual.(*ValuesLit).Elements)
	case *SliceExpr:
		equalExpr(t, expected.Expr,
			actual.(*SliceExpr).Expr)
		equalExpr(t, expected.Low,
			actual.(*SliceExpr).Low)
		equalExpr(t, expected.High,
			actual.(*SliceExpr).High)
		require.Equal(t, expected.LBrack,
			actual.(*SliceExpr).LBrack)
		require.Equal(t, expected.RBrack,
			actual.(*SliceExpr).RBrack)
	default:
		panic(fmt.Errorf("unknown type: %T", expected))
	}
}

func equalIdents(t *testing.T, expected, actual []*Ident) {
	require.Equal(t, len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		equalExpr(t, expected[i], actual[i])
	}
}

func equalExprs(t *testing.T, expected, actual []Expr) {
	require.Equal(t, len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		equalExpr(t, expected[i], actual[i])
	}
}

func equalStmts(t *testing.T, expected, actual []Stmt) {
	require.Equal(t, len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		equalStmt(t, expected[i], actual[i])
	}
}

func equalValuesElements(
	t *testing.T,
	expected, actual []*ValuesElementLit,
) {
	require.Equal(t, len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		require.Equal(t, expected[i].Key, actual[i].Key)
		require.Equal(t, expected[i].KeyPos, actual[i].KeyPos)
		require.Equal(t, expected[i].ColonPos, actual[i].ColonPos)
		equalExpr(t, expected[i].Value, actual[i].Value)
	}
}

func parseSource(
	filename string,
	src []byte,
	trace io.Writer,
) (res *File, err error) {
	fileSet := NewFileSet()
	file := fileSet.AddFile(filename, -1, len(src))

	p := NewParser(file, src, trace)
	return p.ParseFile()
}
