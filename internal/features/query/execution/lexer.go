package execution

import (
	"errors"
	"strconv"
	"strings"
	"unicode"
)

type lexer struct {
	keywords  map[string]lexemeType
	operators map[string]lexemeType
}

func newLexer() *lexer {
	return &lexer{
		keywords: map[string]lexemeType{
			"and":    AndType,
			"or":     OrType,
			"select": SelectType,
			"where":  WhereType,
			"limit":  LimitType,
			"offset": OffsetType,
			"count":  CountType,
		},
		operators: map[string]lexemeType{
			"=":  EqualType,
			">":  BiggerType,
			">=": BiggerOrEqualType,
			"<":  LessType,
			"<=": LessOrEqualType,
		},
	}
}

type lexeme struct {
	Type    lexemeType
	Literal string
}

type lexemeType uint8

const (
	// Identifier String and Number are lexic string
	Identifier lexemeType = iota
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
	LimitType
	OffsetType
	CountType

	// EOF - end of file or string
	EOF
)

var EOFLexeme = lexeme{
	Type: EOF,
}

func (t lexemeType) String() string {
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
	case LimitType:
		return "LIMIT"
	case OffsetType:
		return "OFFSET"
	case CountType:
		return "COUNT"
	case EOF:
		return "EOF"
	default:
		return "invalid"
	}
}

func (l *lexer) parse(input string) ([]lexeme, error) {
	runes := []rune(strings.TrimSpace(input))
	res := make([]lexeme, 0, len(runes))

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

		lex := lexeme{}
		switch runes[i] {
		case '<', '>':
			op := runes[i]
			flush()

			lex.Type = l.operators[string(op)]
			lex.Literal = string(op)
			if i+1 < len(runes) &&
				runes[i+1] == '=' {
				newOp := string(op) + string('=')
				lex.Type = l.operators[newOp]
				lex.Literal = newOp
				i++
			}
		case '=':
			flush()

			lex.Type = l.operators["="]
			lex.Literal = "="
		case '(':
			flush()

			lex.Type = LeftParen
			lex.Literal = "("
		case ')':
			flush()

			lex.Type = RightParen
			lex.Literal = ")"
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

			lex.Type = String
			lex.Literal = builder.String()
		case '*':
			flush()

			lex.Type = AsteriskType
			lex.Literal = "*"
		case ',':
			flush()

			lex.Type = CommaType
			lex.Literal = ","
		default:
			builder.WriteRune(runes[i])
			continue
		}

		builder.Reset()
		res = append(res, lex)
	}

	flush()
	res = append(res, EOFLexeme)

	return res, nil
}

func (l *lexer) toLexeme(s string) lexeme {
	var lex lexeme

	typ, ok := l.keywords[strings.ToLower(s)]
	if ok {
		lex.Type = typ
	} else if _, err := strconv.ParseFloat(s, 64); err == nil {
		lex.Type = Number
	} else if _, err = strconv.ParseBool(s); err == nil {
		lex.Type = Bool
	} else {
		lex.Type = Identifier
	}

	lex.Literal = s
	return lex
}
