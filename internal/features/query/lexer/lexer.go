package lexer

import (
	"errors"
	"strconv"
	"strings"
	"unicode"
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

	Or
	And

	LeftBracket
	RightBracket

	EOF
)

var EOFLexeme = Lexeme{
	Type: EOF,
}

func (l *Lexer) Parse(input string) ([]Lexeme, error) {
	runes := []rune(strings.TrimSpace(input))
	res := make([]Lexeme, 0, len(runes))

	var builder strings.Builder
	flush := func() {
		if builder.Len() == 0 {
			return
		}

		s := builder.String()
		lex := toLexeme(s)

		res = append(res, lex)
		builder.Reset()
	}
	for i := 0; i < len(runes); i++ {
		if unicode.IsSpace(runes[i]) {
			flush()
			continue
		}

		lexeme := Lexeme{}
		switch runes[i] {
		case '<':
			flush()

			if i+1 < len(runes) {
				if runes[i+1] == '=' {
					lexeme.Type = LessOrEqual
					lexeme.Literal = "<="
					i++
				} else {
					lexeme.Type = Less
					lexeme.Literal = "<"
				}
			}
		case '>':
			flush()

			if i+1 < len(runes) {
				if runes[i+1] == '=' {
					lexeme.Type = BiggerOrEqual
					lexeme.Literal = ">="
					i++
				} else {
					lexeme.Type = Bigger
					lexeme.Literal = ">"
				}
			}
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
			lexeme.Literal = ")"
		case '"', '\'':
			quote := runes[i]
			j := i + 1
			if j >= len(runes) {
				return nil, errors.New("opened quote at end of string")
			}

			for runes[j] != quote {
				builder.WriteRune(runes[j])
				if j == len(runes)-1 {
					return nil, errors.New("need missing literal quote")
				}
				j++
			}
			i = j

			lexeme.Type = String
			lexeme.Literal = builder.String()
		default:
			builder.WriteRune(runes[i])
			continue
		}

		builder.Reset()
		res = append(res, lexeme)
	}

	flush()
	res = append(res, EOFLexeme)

	return res, nil
}

func isIdentifier(s string) bool {
	m := map[string]struct{}{
		"level": {}, "component": {}, "pid": {},
		"ip": {}, "time": {}, "text": {},
	}

	_, ok := m[s]
	return ok
}

func toLexeme(s string) Lexeme {
	var lexeme Lexeme
	switch s {
	case "and":
		lexeme.Type = And
	case "or":
		lexeme.Type = Or
	default:
		if _, err := strconv.Atoi(s); err != nil {
			if isIdentifier(strings.ToLower(s)) {
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
