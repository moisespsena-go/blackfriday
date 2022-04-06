package token

import "strconv"

// Token represents a token.
type Token int

// Names of tokens
const (
	Illegal Token = iota
	EOF
	Comment
	_literalBeg
	Ident
	Number
	String
	_literalEnd

	LParen    // (
	RParen    // )
	LBrack    // [
	RBrack    // ]
	LBrace    // {
	RBrace    // }
	Comma     // ,
	Semicolon // ;
	Colon     // :
)

var tokens = [...]string{
	Illegal:   "ILLEGAL",
	EOF:       "EOF",
	Comment:   "COMMENT",
	String:    "STRING",
	LParen:    "(",
	LBrack:    "[",
	LBrace:    "{",
	Semicolon: ";",
	Comma:     ",",
	Colon:     ":",
	RBrace:    "}",
	RParen:    ")",
	RBrack:    "]",
}

func (tok Token) String() string {
	s := ""

	if 0 <= tok && tok < Token(len(tokens)) {
		s = tokens[tok]
	}

	if s == "" {
		s = "token(" + strconv.Itoa(int(tok)) + ")"
	}

	return s
}

// LowestPrec represents lowest operator precedence.
const LowestPrec = 0

// Precedence returns the precedence for the operator token.
func (tok Token) Precedence() int {
	return LowestPrec
}

// IsLiteral returns true if the token is a literal.
func (tok Token) IsLiteral() bool {
	return _literalBeg < tok && tok < _literalEnd
}
