package query

import (
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

type TypedQuery struct {
	Fields     []string
	Expression TypedExpr
}

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

func (s *SemanticAnalyzer) Analyze(
	query Query,
	catalog *domain.FieldsCatalog,
) (TypedQuery, error) {
	if err := s.analyzeFields(
		query.Fields,
		catalog,
	); err != nil {
		return TypedQuery{}, fmt.Errorf(
			"analyze fields: %w",
			err,
		)
	}

	typed, err := s.analyzeExpression(query.Where)
	if err != nil {
		return TypedQuery{}, fmt.Errorf(
			"analyze expression: %w",
			err,
		)
	}

	return TypedQuery{
		Fields:     query.Fields,
		Expression: typed,
	}, nil
}

func (s *SemanticAnalyzer) analyzeFields(
	fields []string,
	catalog *domain.FieldsCatalog,
) error {
	for _, field := range fields {
		if field == "*" {
			return nil
		}

		if slices.ContainsFunc(
			s.scheme.Parameters,
			func(f domain.Field) bool {
				return f.Name == field
			}) {
			continue
		}

		if !catalog.Has(field) {
			return fmt.Errorf(
				"undefined field: %q",
				field,
			)
		}
	}

	return nil
}

func (s *SemanticAnalyzer) analyzeExpression(
	expr Expr,
) (TypedExpr, error) {
	if expr == nil {
		return nil, nil
	}

	switch e := expr.(type) {
	case *BinaryExpr:
		left, err := s.analyzeExpression(e.Left)
		if err != nil {
			return nil, fmt.Errorf(
				"left analyze: %w",
				err,
			)
		}

		right, err := s.analyzeExpression(e.Right)
		if err != nil {
			return nil, fmt.Errorf(
				"right analyze: %w",
				err,
			)
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
		BinaryExpr{},
		Condition{},
		expr,
	)
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
			return nil, fmt.Errorf(
				"semantic parse int: %w",
				err,
			)
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
			return nil, fmt.Errorf(
				"semantic parse float: %w",
				err,
			)
		}

		return domain.FloatValue(val), nil
	case domain.BoolType:
		if lexeme.Type != Bool {
			return nil, fmt.Errorf(
				"want bool type, got: %s",
				lexeme.Type,
			)
		}

		val, err := strconv.ParseBool(lexeme.Literal)
		if err != nil {
			return nil, fmt.Errorf(
				"semantic parse bool: %w",
				err,
			)
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
