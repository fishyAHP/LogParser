package query

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
)

type SemanticAnalyzer struct {
	scheme *domain.Scheme
}

func NewSemanticAnalyzer(scheme *domain.Scheme) *SemanticAnalyzer {
	return &SemanticAnalyzer{
		scheme: scheme,
	}
}

type TypedExpr interface {
	isTypedExpr()
}

type TypedBinaryExpr struct {
	Left     TypedExpr
	Operator LogicalOperator
	Right    TypedExpr
}

type TypedCondition struct {
	Field    string
	Operator CompareOperator
	Value    domain.Value
}

func (t *TypedCondition) isTypedExpr()  {}
func (t *TypedBinaryExpr) isTypedExpr() {}

func allowedOperators(typ domain.IndexType) CompareOperator {
	switch typ {
	case domain.HashIndex, domain.TextIndex:
		return EqualityOps
	case domain.RangeIndex:
		return OrderedOps
	default:
		return 0
	}
}

func isAllowed(allowed, op CompareOperator) bool {
	return allowed&op != 0
}

func (s *SemanticAnalyzer) Analyze(expr Expr) (TypedExpr, error) {
	if expr == nil {
		return nil, errors.New("expression is nil")
	}

	switch e := expr.(type) {
	case *BinaryExpr:
		left, err := s.Analyze(e.Left)
		if err != nil {
			return nil, fmt.Errorf("left analyze: %w", err)
		}

		right, err := s.Analyze(e.Right)
		if err != nil {
			return nil, fmt.Errorf("right analyze: %w", err)
		}
		return &TypedBinaryExpr{
			Left:     left,
			Operator: e.Operator,
			Right:    right,
		}, nil
	case *Condition:
		idx := slices.IndexFunc(
			s.scheme.Parameters,
			func(a domain.Field) bool {
				return a.Name == e.Field
			})
		if idx == -1 {
			return nil, fmt.Errorf(
				"unknown field in condition: %s",
				e.Field,
			)
		}

		field := s.scheme.Parameters[idx]
		if !isAllowed(
			allowedOperators(field.IndexType),
			e.Operator) {
			return nil, fmt.Errorf(
				"operator %v is not supported for %v",
				e.Operator,
				field.IndexType,
			)
		}

		value, err := parseValue(field.FieldType, e.Value)
		if err != nil {
			return nil, fmt.Errorf(
				"field %q: %w",
				e.Field,
				err,
			)
		}
		return &TypedCondition{
			Field:    e.Field,
			Operator: e.Operator,
			Value:    value,
		}, nil
	}
	return nil, fmt.Errorf(
		"invalid type of expression: want %T or %T, got %T",
		BinaryExpr{}, Condition{}, expr)
}

func parseValue(field domain.DataType, lexeme Lexeme) (domain.Value, error) {
	switch field {
	case domain.StringType:
		if lexeme.Type != String {
			return nil, fmt.Errorf(
				"want string type, got: %s",
				lexeme.Type,
			)
		}

		return domain.StringValue(lexeme.Literal), nil
	case domain.IntType:
		if lexeme.Type != Number {
			return nil, fmt.Errorf(
				"want int type, got: %s",
				lexeme.Type,
			)
		}

		val, err := strconv.ParseInt(lexeme.Literal, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("semantic parse int: %w", err)
		}

		return domain.IntValue(val), nil
	case domain.FloatType:
		if lexeme.Type != Number {
			return nil, fmt.Errorf(
				"want float type, got: %s",
				lexeme.Type,
			)
		}
		val, err := strconv.ParseFloat(lexeme.Literal, 64)
		if err != nil {
			return nil, fmt.Errorf("semantic parse float: %w", err)
		}

		return domain.FloatValue(val), nil
	case domain.BoolType:
		if lexeme.Type != Bool {
			return nil, fmt.Errorf("want bool type, got: %s",
				lexeme.Type,
			)
		}

		val, err := strconv.ParseBool(lexeme.Literal)
		if err != nil {
			return nil, fmt.Errorf("semantic parse bool: %w", err)
		}

		return domain.BoolValue(val), nil
	case domain.TimeType:
		if lexeme.Type != String {
			return nil, fmt.Errorf(
				"want string type, got: %s",
				lexeme.Type,
			)
		}

		parsed, err := time.Parse(time.RFC3339, lexeme.Literal)
		if err != nil {
			return nil, fmt.Errorf(
				"parse time %q as RFC3339: %w",
				lexeme.Literal,
				err,
			)
		}

		return domain.TimeValue(parsed), nil
	default:
		return nil, fmt.Errorf(
			"unsupported field type: %v",
			field,
		)
	}
}
