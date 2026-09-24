package lexer

import (
	"strconv"
	"strings"
)

type Lexer struct {
}

type Lexeme struct {
	Type    LexemeType
	Literal string
}

type LexemeType uint8

const (
	Identifier LexemeType = iota
	String
	Number

	Equal
	Bigger
	BiggerOrEqual
	Less
	LessOrEqual

	And
	Or

	LeftBracket
	RightBracket

	EOF
)

var EOFLexeme = Lexeme{
	Type:    EOF,
	Literal: "\n",
}

func (l *Lexer) Parse(input string) []Lexeme {
	runes := []rune(strings.TrimSpace(input))
	res := make([]Lexeme, 0, len(runes))

	var builder strings.Builder
	for _, r := range runes {
		flush := func() {
			s := builder.String()
			lex := toLexeme(s)

			res = append(res, lex)

			builder.Reset()
		}

		lexeme := Lexeme{}
		switch r {
		case '=':
			flush()

			lexeme.Type = Equal
			lexeme.Literal = "="
		case '(':
			flush()

			lexeme.Type = LeftBracket
			lexeme.Literal = "("
		case ')':
			flush()

			lexeme.Type = RightBracket
			lexeme.Literal = "("
		case ' ':
			flush()
		default:
			builder.WriteRune(r)
			continue
		}

		builder.Reset()
		res = append(res, lexeme)
	}

	res = append(res)
	return res
}

func isIdentifier(s string) bool {
	m := map[string]struct{}{
		"level": {}, "component": {},
		"ip": {}, "time": {}, "text": {},
	}

	_, ok := m[s]
	return ok
}

func toLexeme(s string) Lexeme {
	var lexeme Lexeme
	switch s {
	case ">":
		lexeme.Type = Bigger
	case ">=":
		lexeme.Type = BiggerOrEqual
	case "<":
		lexeme.Type = Less
	case "<=":
		lexeme.Type = LessOrEqual
	case "and":
		lexeme.Type = And
	case "or":
		lexeme.Type = Or
	default:
		if _, err := strconv.Atoi(s); err != nil {
			if isIdentifier(s) {
				lexeme.Type = Identifier
				break
			}
			lexeme.Type = String
			break
		}
		lexeme.Type = Number
	}

	lexeme.Literal = s
	return lexeme
}
