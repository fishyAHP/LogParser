package execution

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
)

type semanticAnalyzer struct {
	scheme *domain.Scheme
}

func newSemanticAnalyzer(scheme *domain.Scheme) *semanticAnalyzer {
	return &semanticAnalyzer{
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
	Fields     []SelectExpr
	Expression TypedExpr
	Limit      *uint64
	Offset     *uint64
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

func (s *semanticAnalyzer) analyze(
	query query,
	catalog *domain.FieldsCatalog,
) (TypedQuery, error) {
	if err := s.analyzeFields(
		query.Projection,
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

	fields := query.Projection
	if len(fields) == 0 {
		fields = []SelectExpr{SelectWildCard{}}
	}

	return TypedQuery{
		Fields:     fields,
		Expression: typed,
		Limit:      query.Limit,
		Offset:     query.Offset,
	}, nil
}

func (s *semanticAnalyzer) analyzeFields(
	fields []SelectExpr,
	catalog *domain.FieldsCatalog,
) error {
	for _, field := range fields {
		switch expr := field.(type) {
		case SelectWildCard:
			if len(fields) != 1 {
				return errors.New(
					"wildcard cannot be mixed with other fields",
				)
			}
			return nil

		case SelectedField:
			name := string(expr)

			if slices.ContainsFunc(
				s.scheme.Parameters,
				func(f domain.Field) bool {
					return f.Name == name
				}) {
				continue
			}

			if !catalog.Has(name) {
				return fmt.Errorf(
					"undefined field: %q",
					field,
				)
			}

		case SelectAggregate:
			if expr.Func == Count &&
				expr.Field == "*" {
				continue
			}

			idx := slices.IndexFunc(s.scheme.Parameters, func(a domain.Field) bool {
				return a.Name == expr.Field
			})

			if idx == -1 {
				return fmt.Errorf(
					"undefined aggregate field: %q",
					expr.Field,
				)
			}

			need := s.scheme.Parameters[idx].FieldType

			switch expr.Func {
			case Max, Min:
				if !In(need, domain.FloatType,
					domain.IntType, domain.TimeType,
					domain.StringType) {
					return fmt.Errorf(
						"unsupported type for MIN/MAX: %v",
						need,
					)
				}
			case Avg:
				if !In(need, domain.FloatType,
					domain.IntType, domain.TimeType) {
					return fmt.Errorf(
						"unsupported type for AVG: %v",
						need,
					)
				}
			case Sum:
				if !In(need, domain.FloatType,
					domain.IntType) {
					return fmt.Errorf(
						"unsupported type for SUM: %v",
						need,
					)
				}

			case Count:
			default:
				return fmt.Errorf(
					"unsupported aggregate func: %s",
					expr.Field,
				)
			}
		}
	}

	return nil
}

func (s *semanticAnalyzer) analyzeExpression(
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

func parseValue(field domain.DataType, lexeme lexeme) (domain.Value, error) {
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

func In(need domain.DataType, other ...domain.DataType) bool {
	return slices.Contains(other, need)
}
