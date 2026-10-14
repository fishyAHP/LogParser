package iterators

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"fmt"
	"iter"

	"fishyAHP/LogParser.git/internal/features/index/structs"
	"fishyAHP/LogParser.git/internal/features/parse/common"
	"fishyAHP/LogParser.git/internal/features/storage"
)

type Rows = iter.Seq2[[]byte, error]

func Scan(
	posting *structs.PostingList,
	store *storage.Storage,
) Rows {
	return func(yield func([]byte, error) bool) {
		if posting == nil {
			iterator := store.Iterator()

			for iterator.Next() {
				if !yield(iterator.Data(), nil) {
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

		// TODO need lazy iterator because Slice() returns a copy
		for _, id := range posting.Slice() {
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

func Count(
	rows Rows,
) Rows {
	return func(yield func([]byte, error) bool) {
		var counter uint64

		for _, err := range rows {
			if err != nil {
				yield(nil, err)
				return
			}

			counter++
		}

		yield(
			[]byte(fmt.Sprintf(`{"COUNT(*)":%d}`, counter)),
			nil,
		)
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
		return func(yield func([]byte, error) bool) {
			for data, err := range source {
				if err != nil {
					yield(nil, err)
					return
				}

				if !yield(bytes.Clone(data), nil) {
					return
				}
			}
		}
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
