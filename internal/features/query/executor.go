package query

import (
	"fmt"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index"
	"fishyAHP/LogParser.git/internal/features/index/common"
	"fishyAHP/LogParser.git/internal/features/index/set"
)

type Executor struct {
	indexes  *index.Service
	syntax   *Syntaxes
	semantic *SemanticAnalyzer
}

func NewExecutor(indexes *index.Service) *Executor {
	return &Executor{
		indexes: indexes,
		syntax: NewSyntaxes(
			NewLexer(),
		),
		semantic: NewSemanticAnalyzer(
			indexes.Scheme,
		),
	}
}

func (e *Executor) Execute(
	s string,
) (*set.Set[domain.RecordData], error) {
	expr, err := e.syntax.Query(s)
	if err != nil {
		return nil, fmt.Errorf(
			"syntax query: %w",
			err,
		)
	}

	typedExpr, err := e.semantic.Analyze(expr)
	res, err := e.executing(typedExpr)

	if err != nil {
		return nil, fmt.Errorf("executing expression: %w", err)
	}
	return res, nil
}

func (e *Executor) executing(
	typedExpr TypedExpr,
) (*set.Set[domain.RecordData], error) {
	switch ex := typedExpr.(type) {
	case *TypedBinaryExpr:
		left, err := e.executing(ex.Left)
		if err != nil {
			return nil, fmt.Errorf("left executing: %w", err)
		}
		right, err := e.executing(ex.Right)
		if err != nil {
			return nil, fmt.Errorf("right executing: %w", err)
		}

		switch ex.Operator {
		case Or:
			return set.Union(left, right), nil
		case And:
			return set.Intersection(left, right), nil
		default:
			return nil, fmt.Errorf("unexpected logical operator")
		}
	case *TypedCondition:
		switch ex.Operator {
		case Equal:
			return e.indexes.Exact(ex.Field, ex.Value)
		case Less, LessOrEqual:
			right := &common.Bound{
				Value:     ex.Value,
				Inclusive: LessOrEqual&ex.Operator != 0,
			}

			return e.indexes.Range(ex.Field, nil, right)
		case Bigger, BiggerOrEqual:
			left := &common.Bound{
				Value:     ex.Value,
				Inclusive: BiggerOrEqual&ex.Operator != 0,
			}

			return e.indexes.Range(ex.Field, left, nil)
		default:
			return nil, fmt.Errorf("unexpected operator")
		}
	default:
		return nil, fmt.Errorf("unexpected expression type: %T", ex)
	}
}
