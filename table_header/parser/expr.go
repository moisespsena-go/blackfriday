package parser

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/russross/blackfriday/v2/table_header/token"
)

// Expr represents an expression node in the AST.
type Expr interface {
	Node
	exprNode()
}

// ArrayLit represents an array literal.
type ArrayLit struct {
	Elements []Expr
	LBrack   Pos
	RBrack   Pos
}

func (e *ArrayLit) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *ArrayLit) Pos() Pos {
	return e.LBrack
}

// End returns the position of first character immediately after the node.
func (e *ArrayLit) End() Pos {
	return e.RBrack + 1
}

func (e *ArrayLit) String() string {
	var elements []string
	for _, m := range e.Elements {
		elements = append(elements, m.String())
	}
	return "[" + strings.Join(elements, ", ") + "]"
}

// BadExpr represents a bad expression.
type BadExpr struct {
	From Pos
	To   Pos
}

func (e *BadExpr) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *BadExpr) Pos() Pos {
	return e.From
}

// End returns the position of first character immediately after the node.
func (e *BadExpr) End() Pos {
	return e.To
}

func (e *BadExpr) String() string {
	return "<bad expression>"
}

// Ident represents an identifier.
type Ident struct {
	Name    string
	NamePos Pos
}

func (e *Ident) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *Ident) Pos() Pos {
	return e.NamePos
}

// End returns the position of first character immediately after the node.
func (e *Ident) End() Pos {
	return Pos(int(e.NamePos) + len(e.Name))
}

func (e *Ident) String() string {
	if e != nil {
		return e.Name
	}
	return nullRep
}

// Number represents an number.
type Number struct {
	Value    string
	ValuePos Pos
}

func (e *Number) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *Number) Pos() Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *Number) End() Pos {
	return Pos(int(e.ValuePos) + len(e.Value))
}

func (e *Number) String() string {
	if e != nil {
		return e.Value
	}
	return nullRep
}

// SliceExpr represents a slice expression.
type SliceExpr struct {
	Expr   Expr
	LBrack Pos
	Low    Expr
	High   Expr
	RBrack Pos
}

func (e *SliceExpr) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *SliceExpr) Pos() Pos {
	return e.Expr.Pos()
}

// End returns the position of first character immediately after the node.
func (e *SliceExpr) End() Pos {
	return e.RBrack + 1
}

func (e *SliceExpr) String() string {
	var low, high string
	if e.Low != nil {
		low = e.Low.String()
	}
	if e.High != nil {
		high = e.High.String()
	}
	return e.Expr.String() + "[" + low + ":" + high + "]"
}

// StringLit represents a string literal.
type StringLit struct {
	Value    string
	ValuePos Pos
	Literal  string
}

func (e *StringLit) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *StringLit) Pos() Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *StringLit) End() Pos {
	return Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *StringLit) String() string {
	return e.Literal
}

// ValuesElementLit represents a map element.
type ValuesElementLit struct {
	Key      string
	KeyPos   Pos
	ColonPos Pos
	Value    Expr
	Tags     *ValuesLit
}

func (e *ValuesElementLit) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *ValuesElementLit) Pos() Pos {
	return e.KeyPos
}

// End returns the position of first character immediately after the node.
func (e *ValuesElementLit) End() Pos {
	return e.Value.End()
}

func (e *ValuesElementLit) String() string {
	var buf bytes.Buffer
	buf.WriteString(strconv.Quote(e.Key))

	if e.Tags != nil {
		buf.WriteString(e.Tags.String())
	}

	if e.Value != nil {
		buf.WriteString(":")
		buf.WriteString(e.Value.String())
	}
	return buf.String()
}

// ValuesLit represents a map literal.
type ValuesLit struct {
	LBrace    Pos
	Elements  []*ValuesElementLit
	RBrace    Pos
	BraceOpen token.Token
}

func (e *ValuesLit) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *ValuesLit) Pos() Pos {
	return e.LBrace
}

// End returns the position of first character immediately after the node.
func (e *ValuesLit) End() Pos {
	return e.RBrace + 1
}

func (e *ValuesLit) String() string {
	var elements []string
	for _, m := range e.Elements {
		elements = append(elements, m.String())
	}
	return e.BraceOpen.String() + strings.Join(elements, ", ") + (e.BraceOpen + 1).String()
}
