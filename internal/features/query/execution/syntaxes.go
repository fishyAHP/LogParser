package execution

import (
	"errors"
	"fmt"
	"strconv"
)

// Grammar:
// query	= SELECT fields [WHERE expression] [LIMIT number] | expression [LIMIT number]
// fields	= "*" | identifier {"," identifier}
// expression = orExpr

type syntaxes struct {
	lexemes []lexeme
	pos     int
}

func newSyntaxes(lexemes []lexeme) *syntaxes {
	return &syntaxes{
		lexemes: lexemes,
	}
}

var (
	ErrUnexpectedLexeme = errors.New("unexpected lexeme type")
	ErrEmptyQuery       = errors.New("empty query")
)

func (s *syntaxes) current() lexeme {
	return s.lexemes[s.pos]
}

func (s *syntaxes) advance() {
	if s.pos < len(s.lexemes)-1 {
		s.pos++
	}
}

func (s *syntaxes) check(t lexemeType) bool {
	return s.current().Type == t
}

func (s *syntaxes) match(t lexemeType) bool {
	if !s.check(t) {
		return false
	}

	s.advance()
	return true
}

func (s *syntaxes) expect(t lexemeType) (lexeme, error) {
	lex := s.current()

	if !s.match(t) {
		return lexeme{}, fmt.Errorf(
			"%w: want %s, got %s",
			ErrUnexpectedLexeme,
			t,
			lex.Type,
		)
	}

	return lex, nil
}

// query represents a parsed query:
//
// Valid states:
//
//  1. Projection == nil, Where != nil:
//     Old filtering without SELECT
//
//  2. Projection == ["*"], Where != nil:
//     FullScan fields with filtering
//
//  3. Projection contain field names, Where != nil:
//     Field projection with filtering
//
//  4. Projection contain field names, Where == nil:
//     Field projection without filtering
//
//  5. Projection == ["*"], Where == nil:
//     SELECT all records
//
// Limit has 3 states:
//
//  1. Limit == nil:
//     Expect all records
//
//  2. Limit == 0:
//     Expect 0 records
//
//  3. Limit > 0:
//     Expect Limit records
type query struct {
	Projection []SelectExpr
	Where      Expr
	Limit      *uint64
	Offset     *uint64
}

type SelectExpr interface {
	isSelect()
}

type SelectedField string

type AggregateFunc uint8

const (
	Count AggregateFunc = iota
	Max
	Min
	Avg
	Sum
)

func (a AggregateFunc) String() string {
	switch a {
	case Count:
		return "COUNT"
	case Max:
		return "MAX"
	case Min:
		return "MIN"
	case Avg:
		return "AVG"
	case Sum:
		return "SUM"
	default:
		return "unknown"
	}
}

type SelectAggregate struct {
	Func  AggregateFunc
	Field string
}

type SelectWildCard struct{}

func (SelectWildCard) isSelect()  {}
func (SelectedField) isSelect()   {}
func (SelectAggregate) isSelect() {}

func (s *syntaxes) query() (query, error) {
	if len(s.lexemes) == 0 ||
		(len(s.lexemes) == 1 && s.lexemes[0].Type == EOF) {
		return query{}, ErrEmptyQuery
	}
	s.pos = 0

	var q query
	if s.match(SelectType) {
		projection, err := s.parseProjection()
		if err != nil {
			return query{}, fmt.Errorf(
				"parse SELECT expression: %w",
				err,
			)
		}
		q.Projection = projection

		if s.match(WhereType) {
			expr, err := s.parseOr()
			if err != nil {
				return query{}, fmt.Errorf(
					"parse WHERE expression: %w",
					err,
				)
			}

			q.Where = expr
		}
	} else {
		expr, err := s.parseOr()
		if err != nil {
			return query{}, err
		}

		q.Where = expr
	}

	limit, offset, err := s.parsePagination()
	if err != nil {
		return query{}, err
	}
	q.Limit = limit
	q.Offset = offset

	if _, err = s.expect(EOF); err != nil {
		return query{}, err
	}

	s.lexemes = nil
	return q, nil
}

func (s *syntaxes) parseProjection() ([]SelectExpr, error) {
	selected := make([]SelectExpr, 0, 4)

	for {
		expr, err := s.parseSelectExpr()
		if err != nil {
			return nil, err
		}

		selected = append(selected, expr)

		if !s.match(CommaType) {
			break
		}
	}

	return selected, nil
}

func (s *syntaxes) parseSelectExpr() (SelectExpr, error) {
	lex := s.current()
	switch lex.Type {
	case AsteriskType:
		s.advance()
		return SelectWildCard{}, nil

	case Identifier:
		s.advance()
		return SelectedField(lex.Literal), nil

	case CountType, MaxType, MinType, SumType, AvgType:
		return s.parseAggregate()
	default:
		return nil, fmt.Errorf(
			"%w: got %s",
			ErrUnexpectedLexeme,
			lex.Type,
		)
	}
}

func (s *syntaxes) parseAggregate() (SelectAggregate, error) {
	lex := s.current()

	var aggregate AggregateFunc
	switch lex.Type {
	case CountType:
		aggregate = Count
	case MaxType:
		aggregate = Max
	case MinType:
		aggregate = Min
	case AvgType:
		aggregate = Avg
	case SumType:
		aggregate = Sum
	default:
		return SelectAggregate{}, fmt.Errorf(
			"%w: invalid aggregate: %s",
			ErrUnexpectedLexeme,
			lex.Type,
		)
	}

	s.advance()
	if _, err := s.expect(LeftParen); err != nil {
		return SelectAggregate{}, err
	}

	argument := s.current()
	if aggregate == Count {
		if argument.Type != Identifier &&
			argument.Type != AsteriskType {
			return SelectAggregate{}, fmt.Errorf(
				"%w: invalid COUNT argument",
				ErrUnexpectedLexeme,
			)
		}
	} else if argument.Type != Identifier {
		return SelectAggregate{}, fmt.Errorf(
			"%w: aggregate requires field name",
			ErrUnexpectedLexeme,
		)
	}

	s.advance()

	if _, err := s.expect(RightParen); err != nil {
		return SelectAggregate{}, err
	}

	return SelectAggregate{
		Func:  aggregate,
		Field: argument.Literal,
	}, nil
}

func (s *syntaxes) parseExpression() (Expr, error) {
	expression, err := s.parseOr()
	if err != nil {
		return nil, fmt.Errorf(
			"parse or: %w",
			err,
		)
	}

	return expression, nil
}

func (s *syntaxes) parseOr() (Expr, error) {
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

func (s *syntaxes) parseAnd() (Expr, error) {
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

func (s *syntaxes) parsePrimary() (Expr, error) {
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

func (s *syntaxes) parseComparison() (Expr, error) {
	field, err := s.expect(Identifier)
	if err != nil {
		return nil, err
	}

	op := s.current()
	conditionOp, ok := mapping(op.Type)
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

func (s *syntaxes) parsePagination() (limit, offset *uint64, err error) {
	for {
		switch {
		case s.check(LimitType):
			if limit != nil {
				return nil, nil, errors.New(
					"duplicate LIMIT",
				)
			}

			limit, err = s.parsePipeline(LimitType)
			if err != nil {
				return nil, nil, err
			}
		case s.check(OffsetType):
			if offset != nil {
				return nil, nil, errors.New(
					"duplicate offset",
				)
			}

			offset, err = s.parsePipeline(OffsetType)
			if err != nil {
				return nil, nil, err
			}
		default:
			return
		}
	}
}

func (s *syntaxes) parsePipeline(t lexemeType) (*uint64, error) {
	if s.match(t) {
		lexeme := s.current()
		if !s.match(Number) {
			return nil, fmt.Errorf(
				"%w: want %s, got %s",
				ErrUnexpectedLexeme,
				Number,
				lexeme.Type,
			)
		}

		parsed, err := strconv.ParseUint(
			lexeme.Literal,
			10, 64,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"parse uint: %w",
				err,
			)
		}
		return &parsed, nil
	}
	return nil, nil
}

func (s *syntaxes) parseOffset() (*uint64, error) {
	return s.parsePipeline(OffsetType)
}

func (s *syntaxes) parseLimit() (*uint64, error) {
	return s.parsePipeline(LimitType)
}

func mapping(t lexemeType) (CompareOperator, bool) {
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
