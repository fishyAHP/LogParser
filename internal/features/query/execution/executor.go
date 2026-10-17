package execution

import (
	"errors"
	"fmt"
	"iter"
	"slices"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index"
	"fishyAHP/LogParser.git/internal/features/index/common"
)

type Executor struct {
	Indexes *index.Manager
	catalog *domain.FieldsCatalog

	lexer    *lexer
	semantic *semanticAnalyzer
}

func NewExecutor(
	indexes *index.Manager,
	catalog *domain.FieldsCatalog,
) (*Executor, error) {
	if indexes == nil {
		return nil, errors.New(
			"indexes mustn't be nil",
		)
	}

	return &Executor{
		Indexes: indexes,
		catalog: catalog,
		lexer:   newLexer(),
		semantic: newSemanticAnalyzer(
			indexes.Scheme,
		),
	}, nil
}

type Result struct {
	Fields  []SelectExpr
	Posting iter.Seq[domain.RecordID]
	Limit   *uint64
	Offset  *uint64
}

func (e *Executor) Execute(
	s string,
) (Result, error) {
	lexemes, err := e.lexer.parse(s)
	if err != nil {
		return Result{}, fmt.Errorf(
			"lexer parse: %w",
			err,
		)
	}
	syntax := newSyntaxes(lexemes)

	query, err := syntax.query()
	if err != nil {
		return Result{}, fmt.Errorf(
			"syntax query: %w",
			err,
		)
	}

	typed, err := e.semantic.analyze(query, e.catalog)
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
			Fields: typed.Fields,
			Limit:  typed.Limit,
			Offset: typed.Offset,
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
		Offset:  typed.Offset,
	}, nil
}

func (e *Executor) executeExpr(
	typedExpr TypedExpr,
) (iter.Seq[domain.RecordID], error) {
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
			return union(left, right), nil
		case And:
			return intersect(left, right), nil
		default:
			return nil, fmt.Errorf(
				"unexpected logical operator: %s",
				ex.Operator,
			)
		}
	case *TypedCondition:
		switch ex.Operator {
		case Equal:
			res, err := e.Indexes.Exact(ex.Field, ex.Value)
			if err != nil {
				if errors.Is(err, common.ErrRecordNotFound) {
					return slices.Values([]domain.RecordID{}), nil
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

			posting, err := e.Indexes.Range(ex.Field, nil, right)
			if err != nil {
				if errors.Is(err, common.ErrRecordNotFound) {
					return slices.Values([]domain.RecordID{}), nil
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

			posting, err := e.Indexes.Range(ex.Field, left, nil)
			if err != nil {
				if errors.Is(err, common.ErrRecordNotFound) {
					return slices.Values([]domain.RecordID{}), nil
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
