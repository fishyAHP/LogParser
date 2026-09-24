package lexer

import (
	"errors"
	"strconv"
	"strings"
	"unicode"
)

type Lexer struct {
	keywords  map[string]LexemeType
	operators map[string]LexemeType
}

func New() *Lexer {
	return &Lexer{
		keywords: map[string]LexemeType{
			"and": And,
			"or":  Or,
		},
		operators: map[string]LexemeType{
			"=":  Equal,
			">":  Bigger,
			">=": BiggerOrEqual,
			"<":  Less,
			"<=": LessOrEqual,
		},
	}
}

type Lexeme struct {
	Type    LexemeType
	Literal string
}

type LexemeType uint8

const (
	// Identifier String and Number are fields
	Identifier LexemeType = iota
	String
	Number

	// Equal to LessOrEqual are operators
	Equal
	Bigger
	BiggerOrEqual
	Less
	LessOrEqual

	// Or And keywords
	Or
	And

	// LeftBracket and RightBracket lexic string
	LeftBracket
	RightBracket

	// EOF - end of file or string
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
		lex := l.toLexeme(s)

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

			lexeme.Type = Less
			lexeme.Literal = "<"

			if i+1 < len(runes) &&
				runes[i+1] == '=' {
				lexeme.Type = LessOrEqual
				lexeme.Literal = "<="
				i++
			}
		case '>':
			flush()

			lexeme.Type = Bigger
			lexeme.Literal = ">"

			if i+1 < len(runes) &&
				runes[i+1] == '=' {
				lexeme.Type = BiggerOrEqual
				lexeme.Literal = ">="
				i++
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
			flush()

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

func (l *Lexer) toLexeme(s string) Lexeme {
	var lexeme Lexeme

	typ, ok := l.keywords[strings.ToLower(s)]
	if ok {
		lexeme.Type = typ
	} else if _, err := strconv.Atoi(s); err != nil {
		lexeme.Type = Identifier
	} else {
		lexeme.Type = Number
	}

	lexeme.Literal = s
	return lexeme
}
