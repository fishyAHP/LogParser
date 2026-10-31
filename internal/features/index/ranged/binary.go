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
	tmp := New(i.tree.compare)

	entries := make([]sortedEntry[K], 0)

	_, err := indexbin.UnmarshalIndex(
		data,
		indexbin.DecodeKey[K],
		func(
			key K,
			posting *structs.PostingList,
		) error {
			entries = append(entries, sortedEntry[K]{
				key:     key,
				posting: posting,
			})

			return nil
		},
	)
	if err != nil {
		return fmt.Errorf(
			"range index unmarshal: %w",
			err,
		)
	}

	if err = tmp.tree.buildFromSorted(
		entries,
	); err != nil {
		return fmt.Errorf(
			"build tree from snapshot: %w",
			err,
		)
	}

	*i = *tmp
	return nil
}
