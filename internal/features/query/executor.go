package query

import (
	"errors"
	"fmt"

	"fishyAHP/LogParser.git/internal/features/index"
	"fishyAHP/LogParser.git/internal/features/index/common"
	"fishyAHP/LogParser.git/internal/features/index/structs"
)

type Executor struct {
	indexes  *index.Manager
	syntax   *Syntaxes
	semantic *SemanticAnalyzer
}

func NewExecutor(indexes *index.Manager) *Executor {
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
) (*structs.PostingList, error) {
	expr, err := e.syntax.Query(s)
	if err != nil {
		return nil, fmt.Errorf(
			"syntax query: %w",
			err,
		)
	}

	typedExpr, err := e.semantic.Analyze(expr)
	if err != nil {
		return nil, fmt.Errorf(
			"semantic analyze: %w",
			err,
		)
	}

	res, err := e.executeExpr(typedExpr)
	if err != nil {
		return nil, fmt.Errorf(
			"execute expression: %w",
			err,
		)
	}
	return res, nil
}

func (e *Executor) executeExpr(
	typedExpr TypedExpr,
) (*structs.PostingList, error) {
	switch ex := typedExpr.(type) {
	case *TypedBinaryExpr:
		left, err := e.executeExpr(ex.Left)
		if err != nil {
			return nil, fmt.Errorf(
				"left executeExpr: %w",
				err,
			)
		}
		right, err := e.executeExpr(ex.Right)
		if err != nil {
			return nil, fmt.Errorf(
				"right executeExpr: %w",
				err,
			)
		}

		switch ex.Operator {
		case Or:
			return structs.UnionLists(left, right), nil
		case And:
			return structs.IntersectionLists(left, right), nil
		default:
			return nil, fmt.Errorf("unexpected logical operator")
		}
	case *TypedCondition:
		switch ex.Operator {
		case Equal:
			res, err := e.indexes.Exact(ex.Field, ex.Value)
			if err != nil {
				if errors.Is(err, common.ErrRecordNotFound) {
					return &structs.PostingList{}, nil
				}
				return nil, fmt.Errorf(
					"indexes exact: %w",
					err,
				)
			}
			return res, nil
		case Less, LessOrEqual:
			right := &common.Bound{
				Value:     ex.Value,
				Inclusive: LessOrEqual == ex.Operator,
			}

			posting, err := e.indexes.Range(ex.Field, nil, right)
			if err != nil {
				if errors.Is(err, common.ErrRecordNotFound) {
					return &structs.PostingList{}, nil
				}
				return nil, fmt.Errorf(
					"index range: %w",
					err,
				)
			}
			return posting, nil
		case Bigger, BiggerOrEqual:
			left := &common.Bound{
				Value:     ex.Value,
				Inclusive: BiggerOrEqual == ex.Operator,
			}

			posting, err := e.indexes.Range(ex.Field, left, nil)
			if err != nil {
				if errors.Is(err, common.ErrRecordNotFound) {
					return &structs.PostingList{}, nil
				}
				return nil, fmt.Errorf(
					"index range: %w",
					err,
				)
			}
			return posting, nil
		default:
			return nil, fmt.Errorf("unexpected operator")
		}
	default:
		return nil, fmt.Errorf(
			"unexpected expression type: %T",
			ex,
		)
	}
}
