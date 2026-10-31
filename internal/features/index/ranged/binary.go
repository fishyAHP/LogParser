package ranged

import (
	"fmt"

	indexbin "fishyAHP/LogParser.git/internal/features/index/binary"
	"fishyAHP/LogParser.git/internal/features/index/structs"
)

func (i *Index[K]) MarshalBinary() ([]byte, error) {
	entries := func(yield func(
		K,
		*structs.PostingList,
	) bool) {
		for n := range i.tree.traverse(ascending) {
			posting := structs.NewPostingFromSorted(
				n.records.slice(),
			)

			if !yield(n.key, posting) {
				return
			}
		}
	}

	return indexbin.MarshalIndex(
		i.tree.elemsCount,
		i.tree.nodesCount,
		entries,
		indexbin.EncodeKey[K],
	)
}

func (i *Index[K]) UnmarshalBinary(data []byte) error {
	decoder, err := indexbin.NewDecoder(
		data,
		indexbin.DecodeKey[K],
	)
	if err != nil {
		return fmt.Errorf("create index decoder: %w", err)
	}

	tmp := New(i.tree.compare)

	header := decoder.Header()

	var (
		previous K
		hasPrev  bool
	)

	next := func() (K, *structs.PostingList, error) {
		key, posting, err := decoder.Next()
		if err != nil {
			return key, nil, err
		}

		if hasPrev && tmp.tree.compare(previous, key) >= 0 {
			var zero K
			return zero, nil, fmt.Errorf(
				"duplicate or unordered key: %v",
				key,
			)
		}

		previous = key
		hasPrev = true

		return key, posting, nil
	}

	err = tmp.tree.buildFromSorted(
		int(header.EntriesCount),
		next,
	)
	if err != nil {
		return fmt.Errorf(
			"build tree from snapshot: %w",
			err,
		)
	}

	if err = decoder.Finish(); err != nil {
		return fmt.Errorf(
			"finish index decoding: %w",
			err,
		)
	}

	tmp.tree.nodesCount = header.EntriesCount
	tmp.tree.elemsCount = header.RecordsCount

	*i = *tmp
	return nil
}
