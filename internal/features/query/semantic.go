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
	scheme      *domain.Scheme
	possibleOps map[domain.DataType][]CompareOperator
}

func NewSemanticAnalyzer(scheme *domain.Scheme) *SemanticAnalyzer {
	rangable := []CompareOperator{Equal, Less, LessOrEqual, Bigger, BiggerOrEqual}
	equalable := []CompareOperator{Equal}

	return &SemanticAnalyzer{
		scheme: scheme,
		possibleOps: map[domain.DataType][]CompareOperator{
			domain.StringType: equalable, domain.BoolType: equalable,
			domain.IntType: rangable, domain.FloatType: rangable,
			domain.TimeType: rangable,
		},
	}
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
		if ops, ok := s.possibleOps[field.FieldType]; !ok {
			return fmt.Errorf("unknown type of field: %d", field.FieldType)
		} else if !slices.ContainsFunc(ops, func(a CompareOperator) bool {
			if a == e.Operator {
				return true
			}
			return false
		}) {
			return fmt.Errorf("operator '%s' is not supported for field %q", e.Operator, e.Field)
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
