package iterators

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"fmt"
	"iter"
	"slices"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/aggregation/interfaces"
	"fishyAHP/LogParser.git/internal/features/parse/common"
	"fishyAHP/LogParser.git/internal/features/storage"
)

type Rows = iter.Seq2[[]byte, error]

func Scan(
	posting iter.Seq[domain.RecordID],
	store *storage.Storage,
) Rows {
	return func(yield func([]byte, error) bool) {
		if posting == nil {
			iterator := store.Iterator()
			defer iterator.Close()

			for iterator.Next() {
				if !yield(bytes.Clone(iterator.Data()), nil) {
					return
				}
			}

			if err := iterator.Err(); err != nil {
				yield(nil, fmt.Errorf(
					"storage iterator: %w",
					err,
				))
			}
			return
		}

		for id := range posting {
			data, err := store.ReadByID(id)

			if errors.Is(err, storage.ErrRecordNotFound) {
				continue
			}

			if err != nil {
				yield(nil, fmt.Errorf("read record %v: %w", id, err))
				return
			}

			if !yield(data, nil) {
				return
			}
		}
	}
}

func Offset(
	source Rows,
	offset uint64,
) Rows {
	return func(yield func([]byte, error) bool) {
		remaining := offset

		for data, err := range source {
			if err != nil {
				yield(nil, err)
				return
			}

			if remaining > 0 {
				remaining--
				continue
			}

			if !yield(data, nil) {
				return
			}
		}
	}
}

func Limit(
	source Rows,
	limit uint64,
) Rows {
	return func(yield func([]byte, error) bool) {
		remaining := limit

		if remaining == 0 {
			return
		}

		for data, err := range source {
			if err != nil {
				yield(nil, err)
				return
			}

			remaining--

			if !yield(data, nil) ||
				remaining == 0 {
				return
			}
		}
	}
}

func Project(
	source Rows,
	fields []string,
	projection common.Projection,
) Rows {
	if len(fields) == 0 ||
		(len(fields) == 1 && fields[0] == "*") {
		return source
	}

	return func(yield func([]byte, error) bool) {
		for data, err := range source {
			if err != nil {
				yield(nil, err)
				return
			}

			projections, err := projection.Project(data, fields)
			if err != nil {
				yield(nil, fmt.Errorf(
					"parser project: %w",
					err,
				))
				return
			}

			res, err := json.Marshal(projections)
			if err != nil {
				yield(nil, fmt.Errorf(
					"json marshal: %w",
					err,
				))
				return
			}

			if !yield(res, nil) {
				return
			}
		}
	}
}

type AggregateExpr struct {
	Field      string
	Name       string
	Aggregator interfaces.Aggregator
}

func Aggregate(
	source Rows,
	aggregators []AggregateExpr,
	parser common.Parser,
	scheme *domain.Scheme,
) Rows {
	return func(yield func([]byte, error) bool) {
		for data, err := range source {
			if err != nil {
				yield(nil, err)
				return
			}

			parsed, err := parser.Parse(data)
			if err != nil {
				yield(nil, err)
				return
			}

			for _, aggregator := range aggregators {
				if aggregator.Field == "*" {
					if err = aggregator.Aggregator.Add(nil); err != nil {
						yield(nil, err)
						return
					}
					continue
				}

				idx := slices.IndexFunc(scheme.Parameters, func(f domain.Field) bool {
					return f.Name == aggregator.Field
				})

				if idx == -1 {
					continue
				}

				value := parsed.Entry.Values[idx]
				if err = aggregator.Aggregator.Add(value); err != nil {
					yield(nil, err)
					return
				}
			}
		}

		result := make(map[string]domain.Value, len(aggregators))

		for _, aggregator := range aggregators {
			value, err := aggregator.Aggregator.Result()
			if err != nil {
				yield(nil, err)
				return
			}

			result[aggregator.Name] = value
		}

		data, err := json.Marshal(result)
		if err != nil {
			yield(nil, err)
			return
		}

		yield(data, nil)
	}
}
