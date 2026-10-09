package query

import (
	"errors"
	"fmt"
)

// Grammar:
// query  	  = SELECT fields [WHERE expression] | expression
// fields 	  = "*" | identifier {"," identifier}
// expression = orExpr

type Syntaxes struct {
	Lex     *Lexer
	lexemes []Lexeme
	pos     int
}

func NewSyntaxes(lexer *Lexer) *Syntaxes {
	return &Syntaxes{
		Lex: lexer,
	}
}

var (
	ErrUnexpectedLexeme = errors.New("unexpected lexeme type")
)

func (s *Syntaxes) current() Lexeme {
	return s.lexemes[s.pos]
}

func (s *Syntaxes) advance() {
	if s.pos < len(s.lexemes)-1 {
		s.pos++
	}
}

func (s *Syntaxes) check(t LexemeType) bool {
	return s.current().Type == t
}

func (s *Syntaxes) match(t LexemeType) bool {
	if s.check(t) {
		return false
	}

	s.advance()
	return true
}

func (s *Syntaxes) expect(t LexemeType) (Lexeme, error) {
	lexeme := s.current()

	if !s.match(t) {
		return Lexeme{}, fmt.Errorf(
			"%w: want %s, got %s",
			ErrUnexpectedLexeme,
			t,
			lexeme.Type,
		)
	}

	return lexeme, nil
}

// Query represents a parsed query:
//
// Valid states:
//
//  1. Fields == nil, Where != nil:
//     Old filtering without SELECT
//
//  2. Fields == ["*"], Where != nil:
//     All fields with filtering
//
//  3. Fields contain field names, Where != nil:
//     Field projection with filtering
//
//  4. Fields contain field names, Where == nil:
//     Field projection without filtering
//
//  5. Fields == ["*"], Where == nil:
//     SELECT all records
type Query struct {
	Fields []string
	Where  Expr
}

func (s *Syntaxes) Query(input string) (Query, error) {
	lexemes, err := s.Lex.Parse(input)
	if err != nil {
		return Query{}, fmt.Errorf(
			"query parse input: %w",
			err,
		)
	}

	if len(lexemes) == 0 {
		return Query{}, errors.New(
			"empty query",
		)
	}
	s.lexemes = lexemes
	s.pos = 0

	first := s.current()
	query := Query{}

	switch first.Type {
	case SelectType:
		s.advance()

		fields, err := s.parseFields()
		if err != nil {
			return Query{}, err
		}
		query.Fields = fields

		switch s.current().Type {
		case WhereType:
			s.advance()

			expr, err := s.parseOr()
			if err != nil {
				return Query{}, err
			}

			query.Where = expr
		case EOF:
			return query, nil
		default:
			return Query{}, fmt.Errorf(
				"%w: %s",
				ErrUnexpectedLexeme,
				lexemes[s.pos].Type,
			)
		}
	default:
		expr, err := s.parseOr()
		if err != nil {
			return Query{}, err
		}

		query.Where = expr
	}

	if !s.match(EOF) {
		return Query{}, fmt.Errorf(
			"%w %s",
			ErrUnexpectedLexeme,
			lexemes[s.pos].Type,
		)
	}

	return query, nil
}

func (s *Syntaxes) parseFields() ([]string, error) {
	if s.match(AsteriskType) {
		return []string{"*"}, nil
	}

	first, err := s.expect(Identifier)
	if err != nil {
		return nil, err
	}

	fields := make([]string, 0, 5)
	fields = append(fields, first.Literal)

	for s.match(CommaType) {
		field, err := s.expect(Identifier)
		if err != nil {
			return nil, err
		}

		fields = append(fields, field.Literal)
	}

	return fields, nil
}

func (s *Syntaxes) parseExpression() (Expr, error) {
	expression, err := s.parseOr()
	if err != nil {
		return nil, fmt.Errorf(
			"parse or: %w",
			err,
		)
	}

	return expression, nil
}

func (s *Syntaxes) parseOr() (Expr, error) {
	left, err := s.parseAnd()
	if err != nil {
		return nil, fmt.Errorf(
			"left parse and: %w",
			err,
		)
	}

	for s.match(OrType) {
		right, err := s.parseAnd()
		if err != nil {
			return nil, fmt.Errorf(
				"right parse and: %w",
				err,
			)
		}

		left = &BinaryExpr{
			Left:     left,
			Operator: Or,
			Right:    right,
		}
	}

	return left, nil
}

func (s *Syntaxes) parseAnd() (Expr, error) {
	left, err := s.parsePrimary()
	if err != nil {
		return nil, fmt.Errorf(
			"query parse primary: %w",
			err,
		)
	}

	for s.match(AndType) {
		right, err := s.parsePrimary()

		if err != nil {
			return nil, fmt.Errorf(
				"right parse primary: %w",
				err,
			)
		}

		left = &BinaryExpr{
			Left:     left,
			Operator: And,
			Right:    right,
		}
	}

	return left, nil
}

func (s *Syntaxes) parsePrimary() (Expr, error) {
	if s.match(LeftParen) {
		expression, err := s.parseExpression()
		if err != nil {
			return nil, fmt.Errorf(
				"parse expression: %w",
				err,
			)
		}

		if _, err = s.expect(RightParen); err != nil {
			return nil, err
		}

		return expression, nil
	}

	return s.parseComparison()
}

func (s *Syntaxes) parseComparison() (Expr, error) {
	field, err := s.expect(Identifier)
	if err != nil {
		return nil, err
	}

	op := s.current()
	conditionOp, ok := Map(op.Type)
	if !ok {
		return nil, errors.New(
			"not found compare operator",
		)
	}
	s.advance()

	value := s.current()
	switch value.Type {
	case Number, String, Bool:
		s.advance()

	default:
		return nil, fmt.Errorf(
			"expected number, string or bool value, got %s",
			value.Type,
		)
	}

	return &Condition{
		Field:    field.Literal,
		Operator: conditionOp,
		Value:    value,
	}, nil
}

func Map(t LexemeType) (CompareOperator, bool) {
	switch t {
	case EqualType:
		return Equal, true
	case LessType:
		return Less, true
	case LessOrEqualType:
		return LessOrEqual, true
	case BiggerType:
		return Bigger, true
	case BiggerOrEqualType:
		return BiggerOrEqual, true
	default:
		return 0, false
	}
}
