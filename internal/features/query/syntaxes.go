package query

import (
	"errors"
	"fmt"
)

type Syntaxes struct {
	Lex *Lexer
	pos int
}

func (q *Syntaxes) Query(input string) (Expr, error) {
	q.pos = 0
	lexemes, err := q.Lex.Parse(input)
	if err != nil {
		return nil, fmt.Errorf("query parse input: %w", err)
	}

	expression, err := q.parseExpression(lexemes)
	if err != nil {
		return nil, err
	}

	if lexemes[q.pos].Type != EOF {
		return nil, fmt.Errorf("unexpected lexeme %s", lexemes[q.pos].Type)
	}

	return expression, nil
}

func (q *Syntaxes) parseExpression(lexemes []Lexeme) (Expr, error) {
	expression, err := q.parseOr(lexemes)
	if err != nil {
		return nil, fmt.Errorf("parse or: %w", err)
	}

	return expression, nil
}

func (q *Syntaxes) parseOr(lexemes []Lexeme) (Expr, error) {
	left, err := q.parseAnd(lexemes)
	if err != nil {
		return nil, fmt.Errorf("left parse and: %w", err)
	}

	if q.pos >= len(lexemes) {
		return left, nil
	}

	for q.pos < len(lexemes) &&
		lexemes[q.pos].Type == OrType {
		q.pos++

		right, err := q.parseAnd(lexemes)
		if err != nil {
			return nil, fmt.Errorf("right parse and: %w", err)
		}

		left = &BinaryExpr{
			Left:     left,
			Operator: Or,
			Right:    right,
		}
	}

	return left, nil
}

func (q *Syntaxes) parseAnd(lexemes []Lexeme) (Expr, error) {
	left, err := q.parsePrimary(lexemes)
	if err != nil {
		return nil, fmt.Errorf("query parse primary: %w", err)
	}

	if q.pos >= len(lexemes) {
		return left, nil
	}

	for q.pos < len(lexemes) &&
		lexemes[q.pos].Type == AndType {
		q.pos++
		right, err := q.parsePrimary(lexemes)

		if err != nil {
			return nil, fmt.Errorf("right parse primary: %w", err)
		}

		left = &BinaryExpr{
			Left:     left,
			Operator: And,
			Right:    right,
		}
	}

	return left, nil
}

func (q *Syntaxes) parsePrimary(lexemes []Lexeme) (Expr, error) {
	if q.pos < len(lexemes) &&
		lexemes[q.pos].Type == LeftParen {
		q.pos++
		expression, err := q.parseExpression(lexemes)
		if err != nil {
			return nil, fmt.Errorf("primary parse expression: %w", err)
		}

		if q.pos >= len(lexemes) {
			return nil, errors.New("want ')' lexeme, got EOF")
		}
		if lexemes[q.pos].Type != RightParen {
			return nil, fmt.Errorf("incorrect lexeme: want ')', got %s", lexemes[q.pos].Type)
		}

		q.pos++
		return expression, nil
	}

	return q.parseComparison(lexemes)
}

func (q *Syntaxes) parseComparison(lexemes []Lexeme) (Expr, error) {
	var cond Condition

	if q.pos+2 >= len(lexemes) {
		return nil, errors.New("index out of ranged")
	}

	field := lexemes[q.pos]
	if field.Type != Identifier {
		return nil, fmt.Errorf("mismatched identifier, got %s", field.Type)
	}
	cond.Field = field.Literal

	op := lexemes[q.pos+1]
	conditionOp, ok := Map(op.Type)
	if !ok {
		return nil, errors.New("not found compare operator")
	}
	cond.Operator = conditionOp

	value := lexemes[q.pos+2]
	if !(value.Type == Number ||
		value.Type == String) {
		return nil, fmt.Errorf("expected number or string value, got %s", value.Type)
	}
	cond.Value = value

	q.pos += 3
	return &cond, nil
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
