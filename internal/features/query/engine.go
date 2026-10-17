package query

import (
	"errors"
	"fmt"
	"slices"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/aggregation"
	"fishyAHP/LogParser.git/internal/features/aggregation/interfaces"
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

	if hasAggregate(result.Fields) {
		aggregators, err := e.buildAggregators(result.Fields)
		if err != nil {
			return nil, err
		}

		rows = iterators.Aggregate(
			rows, aggregators,
			e.parser, e.executor.Indexes.Scheme,
		)
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

func (e *Engine) buildAggregators(
	fields []execution.SelectExpr,
) ([]iterators.AggregateExpr, error) {
	result := make([]iterators.AggregateExpr, 0, len(fields))

	for _, field := range fields {
		aggr, ok := field.(execution.SelectAggregate)
		if !ok {
			return nil, errors.New(
				"unexpected type with aggregators",
			)
		}

		instance, err := e.newAggregator(aggr)
		if err != nil {
			return nil, err
		}

		result = append(result, iterators.AggregateExpr{
			Name:       fmt.Sprintf("%s(%s)", aggr.Func.String(), aggr.Field),
			Field:      aggr.Field,
			Aggregator: instance,
		})
	}

	return result, nil
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

func (e *Engine) newAggregator(
	field execution.SelectAggregate,
) (interfaces.Aggregator, error) {
	if field.Func == execution.Count {
		return aggregation.NewCountAggregator(), nil
	}

	idx := slices.IndexFunc(
		e.executor.Indexes.Scheme.Parameters,
		func(f domain.Field) bool {
			return f.Name == field.Field
		},
	)

	if idx == -1 {
		return nil, fmt.Errorf(
			"field %q not found in scheme",
			field.Field,
		)
	}

	typ := e.executor.
		Indexes.
		Scheme.
		Parameters[idx].
		FieldType

	switch field.Func {
	case execution.Sum:
		return aggregation.NewSumAggregator(typ)

	case execution.Avg:
		return aggregation.NewAvgAggregator(typ)

	case execution.Min:
		return aggregation.NewMinAggregator(typ)

	case execution.Max:
		return aggregation.NewMaxAggregator(typ)

	default:
		return nil, fmt.Errorf(
			"unsupported aggregate function: %v",
			field.Func,
		)
	}
}
