package query

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

func NewLexer() *Lexer {
	return &Lexer{
		keywords: map[string]LexemeType{
			"and":    AndType,
			"or":     OrType,
			"select": SelectType,
			"where":  WhereType,
		},
		operators: map[string]LexemeType{
			"=":  EqualType,
			">":  BiggerType,
			">=": BiggerOrEqualType,
			"<":  LessType,
			"<=": LessOrEqualType,
		},
	}
}

type Lexeme struct {
	Type    LexemeType
	Literal string
}

type LexemeType uint8

const (
	// Identifier String and Number are lexic string
	Identifier LexemeType = iota
	String
	Number
	Bool

	// EqualType to LessOrEqualType are operators
	EqualType
	BiggerType
	BiggerOrEqualType
	LessType
	LessOrEqualType

	// OrType AndType keywords
	OrType
	AndType

	// LeftParen and RightParen lexic string
	LeftParen
	RightParen

	// SelectType and other for projection and filtering
	SelectType
	WhereType
	CommaType
	AsteriskType

	// EOF - end of file or string
	EOF
)

var EOFLexeme = Lexeme{
	Type: EOF,
}

func (t LexemeType) String() string {
	switch t {
	case Identifier:
		return "identifier"
	case String:
		return "string"
	case Number:
		return "number"
	case Bool:
		return "bool"
	case EqualType:
		return "="
	case LessType:
		return "<"
	case LessOrEqualType:
		return "<="
	case BiggerType:
		return ">"
	case BiggerOrEqualType:
		return ">="
	case OrType:
		return "OR"
	case AndType:
		return "AND"
	case LeftParen:
		return "("
	case RightParen:
		return ")"
	case SelectType:
		return "SELECT"
	case WhereType:
		return "WHERE"
	case AsteriskType:
		return "*"
	case CommaType:
		return ","
	case EOF:
		return "EOF"
	default:
		return "invalid"
	}
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
		case '<', '>':
			op := runes[i]
			flush()

			lexeme.Type = l.operators[string(op)]
			lexeme.Literal = string(op)
			if i+1 < len(runes) &&
				runes[i+1] == '=' {
				newOp := string(op) + string('=')
				lexeme.Type = l.operators[newOp]
				lexeme.Literal = newOp
				i++
			}
		case '=':
			flush()

			lexeme.Type = l.operators["="]
			lexeme.Literal = "="
		case '(':
			flush()

			lexeme.Type = LeftParen
			lexeme.Literal = "("
		case ')':
			flush()

			lexeme.Type = RightParen
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
		case '*':
			flush()

			lexeme.Type = AsteriskType
			lexeme.Literal = "*"
		case ',':
			flush()

			lexeme.Type = CommaType
			lexeme.Literal = ","
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
	} else if _, err := strconv.ParseFloat(s, 64); err == nil {
		lexeme.Type = Number
	} else if _, err = strconv.ParseBool(s); err == nil {
		lexeme.Type = Bool
	} else {
		lexeme.Type = Identifier
	}

	lexeme.Literal = s
	return lexeme
}
