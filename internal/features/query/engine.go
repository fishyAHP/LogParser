package query

import (
	"errors"
	"fmt"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index"
	"fishyAHP/LogParser.git/internal/features/parse"
	"fishyAHP/LogParser.git/internal/features/query/execution"
	"fishyAHP/LogParser.git/internal/features/query/iterators"
	"fishyAHP/LogParser.git/internal/features/storage"
)

type Engine struct {
	executor *execution.Executor
	store    *storage.Storage
	parser   *parse.LogParser
}

func NewEngine(
	store *storage.Storage,
	parser *parse.LogParser,
	indexes *index.Manager,
	catalog *domain.FieldsCatalog,
) (*Engine, error) {
	if store == nil {
		return nil, errors.New(
			"storage mustn't be nil",
		)
	}

	if parser == nil {
		return nil, errors.New(
			"parser mustn't be nil",
		)
	}

	exec, err := execution.NewExecutor(indexes, catalog)
	if err != nil {
		return nil, err
	}

	return &Engine{
		executor: exec,
		store:    store,
		parser:   parser,
	}, nil
}

func (e *Engine) Query(
	q string,
) (iterators.Rows, error) {
	result, err := e.executor.Execute(q)
	if err != nil {
		return nil, fmt.Errorf(
			"execute query: %w",
			err,
		)
	}

	return e.buildIterators(result)
}

func (e *Engine) buildIterators(
	result execution.Result,
) (iterators.Rows, error) {
	rows := iterators.Scan(result.Posting, e.store)

	// TODO when add group by and other aggregates func need change it
	if hasAggregate(result.Fields) {
		if len(result.Fields) != 1 {
			return nil, errors.New(
				"multiple expressions with aggregates are not supported yet",
			)
		}

		aggregate, ok := result.Fields[0].(execution.SelectAggregate)
		if !ok || aggregate.Func != execution.Count || aggregate.Field != "*" {
			return nil, errors.New("unsupported aggregate")
		}

		rows = iterators.Count(rows)
	} else {
		fields, err := selectFields(result.Fields)
		if err != nil {
			return nil, err
		}

		rows = iterators.Project(rows, fields, e.parser)
	}

	if result.Offset != nil {
		rows = iterators.Offset(rows, *result.Offset)
	}

	if result.Limit != nil {
		rows = iterators.Limit(rows, *result.Limit)
	}

	return rows, nil
}

func hasAggregate(
	selects []execution.SelectExpr,
) bool {
	for _, expr := range selects {
		if _, ok := expr.(execution.SelectAggregate); ok {
			return true
		}
	}
	return false
}

func selectFields(
	projection []execution.SelectExpr,
) ([]string, error) {
	var fields []string

	for _, expr := range projection {
		switch v := expr.(type) {
		case execution.SelectedField:
			fields = append(fields, string(v))

		case execution.SelectWildCard:
			if len(projection) != 1 {
				return nil, errors.New(
					"wildcard mixed with other expressions",
				)
			}
			return nil, nil

		default:
			return nil, fmt.Errorf(
				"expression %T is not supported by row projection",
				expr,
			)
		}
	}

	return fields, nil
}
