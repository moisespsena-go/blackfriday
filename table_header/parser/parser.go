package parser

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/russross/blackfriday/v2/table_header/token"
)

type bailout struct{}

// Parser parses the Tengo source files. It's based on Go's parser
// implementation.
type Parser struct {
	file      *SourceFile
	errors    ErrorList
	scanner   *Scanner
	pos       Pos
	token     token.Token
	tokenLit  string
	exprLevel int // < 0: in control clause, >= 0: in expression
	syncPos   Pos // last sync position
	syncCount int // number of advance calls without progress
	trace     bool
	indent    int
	traceOut  io.Writer
}

// NewParser creates a Parser.
func NewParser(file *SourceFile, src []byte, trace io.Writer) *Parser {
	p := &Parser{
		file:     file,
		trace:    trace != nil,
		traceOut: trace,
	}
	p.scanner = NewScanner(p.file, src,
		func(pos SourceFilePos, msg string) {
			p.errors.Add(pos, msg)
		}, 0)
	p.next()
	return p
}

func (p *Parser) Pos() Pos {
	return p.pos
}

// ParseFile parses the source and returns an AST file unit.
func (p *Parser) ParseFile() (file *File, err error) {
	defer func() {
		if e := recover(); e != nil {
			if _, ok := e.(bailout); !ok {
				panic(e)
			}
		}

		p.errors.Sort()
		err = p.errors.Err()
	}()

	if p.trace {
		defer untracep(tracep(p, "File"))
	}

	if p.errors.Len() > 0 {
		return nil, p.errors.Err()
	}

	stmts := p.parseStmtList()
	if p.errors.Len() > 0 {
		return nil, p.errors.Err()
	}

	file = &File{
		InputFile: p.file,
		Stmts:     stmts,
	}
	return
}

// ParseFile parses the source and returns an AST file unit.
func (p *Parser) ParseFileC(stmtCount int) (file *File, err error) {
	defer func() {
		if e := recover(); e != nil {
			if _, ok := e.(bailout); !ok {
				panic(e)
			}
		}

		p.errors.Sort()
		err = p.errors.Err()
	}()

	if p.trace {
		defer untracep(tracep(p, "File"))
	}

	if p.errors.Len() > 0 {
		return nil, p.errors.Err()
	}

	stmts := p.parseStmtListCount(stmtCount)
	if p.errors.Len() > 0 {
		return nil, p.errors.Err()
	}

	file = &File{
		InputFile: p.file,
		Stmts:     stmts,
	}
	return
}

func (p *Parser) parseStmtList() (list []Stmt) {
	if p.trace {
		defer untracep(tracep(p, "StatementList"))
	}

	for p.token != token.RBrace && p.token != token.EOF {
		list = append(list, p.parseStmt())
	}
	return
}

func (p *Parser) parseStmtListCount(c int) (list []Stmt) {
	if p.trace {
		defer untracep(tracep(p, "StatementList"))
	}

	for q := 0; q < c && p.token != token.RBrace && p.token != token.EOF; q++ {
		list = append(list, p.parseStmt())
	}
	return
}

func (p *Parser) parseStmt() (stmt Stmt) {
	if p.trace {
		defer untracep(tracep(p, "Statement"))
	}

	switch p.token {
	case // simple statements
		token.Ident, token.LBrace:
		s := p.parseSimpleStmt()
		p.expectSemi()
		return s
	case token.Semicolon:
		s := &EmptyStmt{Semicolon: p.pos, Implicit: p.tokenLit == "\n"}
		p.next()
		return s
	case token.RBrace:
		// semicolon may be omitted before a closing "}"
		return &EmptyStmt{Semicolon: p.pos, Implicit: true}
	default:
		pos := p.pos
		p.errorExpected(pos, "statement")
		p.advance()
		return &BadStmt{From: pos, To: p.pos}
	}
}

func (p *Parser) parseSimpleStmt() Stmt {
	if p.trace {
		defer untracep(tracep(p, "SimpleStmt"))
	}

	x := p.parseExprList()

	if len(x) > 1 {
		p.errorExpected(x[0].Pos(), "1 expression")
		// continue with first expression
	}
	return &ExprStmt{Expr: x[0]}
}

func (p *Parser) parseExprList() (list []Expr) {
	if p.trace {
		defer untracep(tracep(p, "ExpressionList"))
	}

	list = append(list, p.parsePrimaryExpr())
	for p.token == token.Comma {
		p.next()
		list = append(list, p.parseOperand())
	}
	return
}

func (p *Parser) parseExpr() Expr {
	if p.trace {
		defer untracep(tracep(p, "Expression"))
	}
	return p.parsePrimaryExpr()
}

func (p *Parser) parsePrimaryExpr() Expr {
	if p.trace {
		defer untracep(tracep(p, "PrimaryExpression"))
	}

	x := p.parseOperand()
	return x
}

func (p *Parser) parseOperand() Expr {
	if p.trace {
		defer untracep(tracep(p, "Operand"))
	}

	switch p.token {
	case token.Ident:
		return p.parseIdent()
	case token.Number:
		return p.parseNumber()
	case token.String:
		v, _ := strconv.Unquote(p.tokenLit)
		x := &StringLit{
			Value:    v,
			ValuePos: p.pos,
			Literal:  p.tokenLit,
		}
		p.next()
		return x
	case token.LBrace: // map literal
		return p.parseValuesLit(token.LBrace)
	}

	pos := p.pos
	p.errorExpected(pos, "operand")
	p.advance()
	return &BadExpr{From: pos, To: p.pos}
}

func (p *Parser) parseIdent() *Ident {
	pos := p.pos
	name := "_"

	if p.token == token.Ident {
		name = p.tokenLit
		p.next()
	} else {
		p.expect(token.Ident)
	}
	return &Ident{
		NamePos: pos,
		Name:    name,
	}
}

func (p *Parser) parseNumber() *Number {
	pos := p.pos
	value := "0"

	if p.token == token.Number {
		value = p.tokenLit
		p.next()
	} else {
		p.expect(token.Number)
	}
	return &Number{
		ValuePos: pos,
		Value:    value,
	}
}

func (p *Parser) parseValuesElementLit(braceOpen token.Token) *ValuesElementLit {
	if p.trace {
		defer untracep(tracep(p, "ValuesElementLit"))
	}

	pos := p.pos
	name := "_"
	if p.token == token.Ident {
		name = p.tokenLit
	} else if p.token == token.String {
		v, _ := strconv.Unquote(p.tokenLit)
		name = v
	} else {
		p.errorExpected(pos, "map key")
	}

	var (
		element = &ValuesElementLit{
			Key:    name,
			KeyPos: pos,
		}
		next token.Token
	)

	p.next()
try:
	switch p.token {
	case token.Comma, braceOpen + 1:
	default:
		if p.token != braceOpen {
			switch p.token {
			case token.LParen, token.LBrace, token.LBrack:
				element.Tags = p.parseValuesLit(p.token)
				goto try
			}
		}
		element.ColonPos, next = p.expectAny(token.Colon, token.Comma)
		if next == token.Colon {
			element.Value = p.parseExpr()
		}
	}
	return element
}

func (p *Parser) parseValuesLit(open token.Token) *ValuesLit {
	if p.trace {
		defer untracep(tracep(p, "ValuesLit"))
	}
	close := open + 1
	lbrace := p.expect(open)
	p.exprLevel++

	var elements []*ValuesElementLit
	for p.token != (open+1) && p.token != token.EOF {
		elements = append(elements, p.parseValuesElementLit(open))

		if !p.expectComma(close, "map element") {
			break
		}
	}

	p.exprLevel--
	rbrace := p.expect(close)
	return &ValuesLit{
		BraceOpen: open,
		LBrace:    lbrace,
		RBrace:    rbrace,
		Elements:  elements,
	}
}

func (p *Parser) expect(token token.Token) Pos {
	pos := p.pos

	if p.token != token {
		p.errorExpected(pos, "'"+token.String()+"'")
	}
	p.next()
	return pos
}

func (p *Parser) expectAny(token ...token.Token) (Pos, token.Token) {
	pos := p.pos

	for _, t := range token {
		if p.token == t {
			p.next()
			return pos, t
		}
	}
	var s = make([]string, len(token))
	for i, t := range token {
		s[i] = "'" + t.String() + "'"
	}
	p.errorExpected(pos, strings.Join(s, " | "))
	return 0, 0
}

func (p *Parser) expectComma(closing token.Token, want string) bool {
	if p.token == token.Comma {
		p.next()

		if p.token == closing {
			p.errorExpected(p.pos, want)
			return false
		}
		return true
	}

	if p.token == token.Semicolon && p.tokenLit == "\n" {
		p.next()
	}
	return false
}

func (p *Parser) expectSemi() {
	switch p.token {
	case token.RBrace:
		// semicolon is optional before a closing ')' or '}'
	case token.Comma:
		// permit a ',' instead of a ';' but complain
		p.errorExpected(p.pos, "';'")
		fallthrough
	case token.Semicolon:
		p.next()
	default:
		p.errorExpected(p.pos, "';'")
		p.advance()
	}
}

func (p *Parser) advance() {
	for ; p.token != token.EOF; p.next() {
	}
}

func (p *Parser) error(pos Pos, msg string) {
	filePos := p.file.Position(pos)

	n := len(p.errors)
	if n > 0 && p.errors[n-1].Pos.Line == filePos.Line {
		// discard errors reported on the same line
		return
	}
	if n > 10 {
		// too many errors; terminate early
		panic(bailout{})
	}
	p.errors.Add(filePos, msg)
}

func (p *Parser) errorExpected(pos Pos, msg string) {
	msg = "expected " + msg
	if pos == p.pos {
		// error happened at the current position: provide more specific
		switch {
		case p.token == token.Semicolon && p.tokenLit == "\n":
			msg += ", found newline"
		case p.token.IsLiteral():
			msg += ", found " + p.tokenLit
		default:
			msg += ", found '" + p.token.String() + "'"
		}
	}
	p.error(pos, msg)
}

func (p *Parser) next() {
	if p.trace && p.pos.IsValid() {
		s := p.token.String()
		switch {
		case p.token.IsLiteral():
			p.printTrace(s, p.tokenLit)
		default:
			p.printTrace(s)
		}
	}
	p.token, p.tokenLit, p.pos = p.scanner.Scan()
}

func (p *Parser) printTrace(a ...interface{}) {
	const (
		dots = ". . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . "
		n    = len(dots)
	)

	filePos := p.file.Position(p.pos)
	_, _ = fmt.Fprintf(p.traceOut, "%5d: %5d:%3d: ", p.pos, filePos.Line,
		filePos.Column)
	i := 2 * p.indent
	for i > n {
		_, _ = fmt.Fprint(p.traceOut, dots)
		i -= n
	}
	_, _ = fmt.Fprint(p.traceOut, dots[0:i])
	_, _ = fmt.Fprintln(p.traceOut, a...)
}

func (p *Parser) safePos(pos Pos) Pos {
	fileBase := p.file.Base
	fileSize := p.file.Size

	if int(pos) < fileBase || int(pos) > fileBase+fileSize {
		return Pos(fileBase + fileSize)
	}
	return pos
}

func tracep(p *Parser, msg string) *Parser {
	p.printTrace(msg, "(")
	p.indent++
	return p
}

func untracep(p *Parser) {
	p.indent--
	p.printTrace(")")
}
