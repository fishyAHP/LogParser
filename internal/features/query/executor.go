package query

import (
	"errors"
	"fmt"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index"
	"fishyAHP/LogParser.git/internal/features/index/common"
	"fishyAHP/LogParser.git/internal/features/index/structs"
)

type Executor struct {
	indexes *index.Manager
	catalog *domain.FieldsCatalog

	syntax   *Syntaxes
	semantic *SemanticAnalyzer
}

func NewExecutor(
	indexes *index.Manager,
	catalog *domain.FieldsCatalog,
) *Executor {
	return &Executor{
		indexes: indexes,
		catalog: catalog,
		syntax: NewSyntaxes(
			NewLexer(),
		),
		semantic: NewSemanticAnalyzer(
			indexes.Scheme,
		),
	}
}

type Result struct {
	Fields   []string
	Posting  *structs.PostingList
	FullScan bool
	Limit    *uint64
}

func (e *Executor) Execute(
	s string,
) (Result, error) {
	query, err := e.syntax.Query(s)
	if err != nil {
		return Result{}, fmt.Errorf(
			"syntax query: %w",
			err,
		)
	}

	typed, err := e.semantic.Analyze(query, e.catalog)
	if err != nil {
		return Result{}, fmt.Errorf(
			"semantic analyze: %w",
			err,
		)
	}

	if typed.Expression == nil {
		if len(typed.Fields) == 0 {
			return Result{}, errors.New(
				"empty typed expression",
			)
		}

		return Result{
			Fields:   typed.Fields,
			FullScan: true,
			Limit:    typed.Limit,
		}, nil
	}

	res, err := e.executeExpr(typed.Expression)
	if err != nil {
		return Result{}, fmt.Errorf(
			"execute expression: %w",
			err,
		)
	}
	return Result{
		Posting: res,
		Fields:  typed.Fields,
		Limit:   typed.Limit,
	}, nil
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
