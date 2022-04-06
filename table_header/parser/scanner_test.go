package parser_test

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/russross/blackfriday/v2/table_header/parser"
	"github.com/russross/blackfriday/v2/table_header/token"
	"github.com/stretchr/testify/require"
)

var testFileSet = parser.NewFileSet()

type scanResult struct {
	Token   token.Token
	Literal string
	Line    int
	Column  int
}

func TestScanner_Scan(t *testing.T) {
	var testCases = [...]struct {
		token   token.Token
		literal string
	}{
		{token.Number, "1"},
		{token.Number, "-10"},
		{token.Number, "+100"},
		{token.Number, "1.23"},
		{token.Number, "-10.23"},
		{token.Number, "+100.23"},
		{token.Comment, "/* a comment */"},
		{token.Comment, "// a comment \n"},
		{token.Comment, "/*\r*/"},
		{token.Comment, "/**\r/*/"},
		{token.Comment, "/**\r\r/*/"},
		{token.Comment, "//\r\n"},
		{token.Ident, "foobar"},
		{token.Ident, "a۰۱۸"},
		{token.Ident, "foo६४"},
		{token.Ident, "bar９８７６"},
		{token.Ident, "ŝ"},
		{token.Ident, "ŝfoo"},
		{token.LBrace, "{"},
		{token.RBrace, "}"},
		{token.LBrack, "["},
		{token.RBrack, "]"},
		{token.LParen, "("},
		{token.RParen, ")"},
		{token.Comma, ","},
		{token.Colon, ":"},
	}

	// combine
	var lines []string
	var lineSum int
	lineNos := make([]int, len(testCases))
	columnNos := make([]int, len(testCases))
	for i, tc := range testCases {
		// add 0-2 lines before each test case
		emptyLines := rand.Intn(3)
		for j := 0; j < emptyLines; j++ {
			lines = append(lines, strings.Repeat(" ", rand.Intn(10)))
		}

		// add test case line with some whitespaces around it
		emptyColumns := rand.Intn(10)
		lines = append(lines, fmt.Sprintf("%s%s%s",
			strings.Repeat(" ", emptyColumns),
			tc.literal,
			strings.Repeat(" ", rand.Intn(10))))

		lineNos[i] = lineSum + emptyLines + 1
		lineSum += emptyLines + countLines(tc.literal)
		columnNos[i] = emptyColumns + 1
	}

	// expected results
	var expected []scanResult
	var expectedSkipComments []scanResult
	for i, tc := range testCases {
		// expected literal
		var expectedLiteral string
		switch tc.token {
		case token.Comment:
			// strip CRs in comments
			expectedLiteral = string(parser.StripCR([]byte(tc.literal),
				tc.literal[1] == '*'))

			// -style comment literal doesn't contain newline
			if expectedLiteral[1] == '/' {
				expectedLiteral = expectedLiteral[:len(expectedLiteral)-1]
			}
		case token.Ident:
			expectedLiteral = tc.literal
		default:
			if tc.token.IsLiteral() {
				// strip CRs in raw string
				expectedLiteral = tc.literal
				if expectedLiteral[0] == '`' {
					expectedLiteral = string(parser.StripCR(
						[]byte(expectedLiteral), false))
				}
			}
		}

		res := scanResult{
			Token:   tc.token,
			Literal: expectedLiteral,
			Line:    lineNos[i],
			Column:  columnNos[i],
		}

		expected = append(expected, res)
		if tc.token != token.Comment {
			expectedSkipComments = append(expectedSkipComments, res)
		}
	}

	scanExpect(t, strings.Join(lines, "\n"),
		parser.ScanComments|parser.DontInsertSemis, expected...)
	scanExpect(t, strings.Join(lines, "\n"),
		parser.DontInsertSemis, expectedSkipComments...)
}

func TestStripCR(t *testing.T) {
	for _, tc := range []struct {
		input  string
		expect string
	}{
		{"//\n", "//\n"},
		{"//\r\n", "//\n"},
		{"//\r\r\r\n", "//\n"},
		{"//\r*\r/\r\n", "//*/\n"},
		{"/**/", "/**/"},
		{"/*\r/*/", "/*/*/"},
		{"/*\r*/", "/**/"},
		{"/**\r/*/", "/**\r/*/"},
		{"/*\r/\r*\r/*/", "/*/*\r/*/"},
		{"/*\r\r\r\r*/", "/**/"},
	} {
		actual := string(parser.StripCR([]byte(tc.input),
			len(tc.input) >= 2 && tc.input[1] == '*'))
		require.Equal(t, tc.expect, actual)
	}
}

func scanExpect(
	t *testing.T,
	input string,
	mode parser.ScanMode,
	expected ...scanResult,
) {
	testFile := testFileSet.AddFile("test", -1, len(input))

	s := parser.NewScanner(
		testFile,
		[]byte(input),
		func(_ parser.SourceFilePos, msg string) { require.Fail(t, msg) },
		mode)

	for idx, e := range expected {
		tok, literal, pos := s.Scan()

		filePos := testFile.Position(pos)

		require.Equal(t, e.Token, tok, "[%d] expected: %s, actual: %s",
			idx, e.Token.String(), tok.String())
		require.Equal(t, e.Literal, literal)
		require.Equal(t, e.Line, filePos.Line)
		require.Equal(t, e.Column, filePos.Column)
	}

	tok, _, _ := s.Scan()
	require.Equal(t, token.EOF, tok, "more tokens left")
	require.Equal(t, 0, s.ErrorCount())
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	n := 1
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			n++
		}
	}
	return n
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
