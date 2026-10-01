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

func allowedOperators(typ domain.DataType) CompareOperator {
	switch typ {
	case domain.StringType, domain.BoolType:
		return EqualityOps
	case domain.IntType, domain.FloatType, domain.TimeType:
		return OrderedOps
	default:
		return 0
	}
}

func isAllowed(allowed, op CompareOperator) bool {
	return allowed&op != 0
}

func (s *SemanticAnalyzer) Analyze(expr Expr) error {
	if expr == nil {
		return errors.New("expression is nil")
	}

	switch e := expr.(type) {
	case *BinaryExpr:
		if err := s.Analyze(e.Left); err != nil {
			return fmt.Errorf("left analyze: %w", err)
		}
		if err := s.Analyze(e.Right); err != nil {
			return fmt.Errorf("right analyze: %w", err)
		}

		return nil
	case *Condition:
		idx := slices.IndexFunc(s.scheme.Parameters, func(a domain.Field) bool {
			if a.Name == e.Field {
				return true
			}
			return false
		})
		if idx == -1 {
			return fmt.Errorf("unknown field in condition: %s", e.Field)
		}

		field := s.scheme.Parameters[idx]
		if !isAllowed(
			allowedOperators(field.FieldType),
			e.Operator) {
			return fmt.Errorf(
				"operator %v is not supported for %v",
				e.Operator,
				field.FieldType,
			)
		}

		switch field.FieldType {
		case domain.StringType:
			if e.Value.Type != String {
				return fmt.Errorf("want string type, got: %s", e.Value.Type)
			}
		case domain.IntType:
			if e.Value.Type != Number {
				return fmt.Errorf("want int type, got: %s", e.Value)
			}
			if _, err := strconv.Atoi(e.Value.Literal); err != nil {
				return fmt.Errorf("semantic parse float: %w", err)
			}
		case domain.FloatType:
			if e.Value.Type != Number {
				return fmt.Errorf("want float type, got: %s", e.Value)
			}
			if _, err := strconv.ParseFloat(e.Value.Literal, 64); err != nil {
				return fmt.Errorf("semantic parse float: %w", err)
			}
		case domain.BoolType:
			if e.Value.Type != Bool {
				return fmt.Errorf("want bool type, got: %s", e.Value)
			}
		case domain.TimeType:
			if _, err := time.Parse(time.RFC3339, e.Value.Literal); err != nil {
				return fmt.Errorf("want time type, got: %s", e.Value)
			}
		default:
			return nil
		}
	}
	return nil
}
