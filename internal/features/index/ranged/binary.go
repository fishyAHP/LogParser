package ranged

import (
	"errors"
	"fmt"

	"fishyAHP/LogParser.git/internal/core/domain"
	indexbin "fishyAHP/LogParser.git/internal/features/index/binary"
	"fishyAHP/LogParser.git/internal/features/index/common"
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

	_, err := indexbin.UnmarshalIndex(
		data,
		indexbin.DecodeKey[K],
		func(
			key K,
			posting *structs.PostingList,
		) error {
			if _, err := tmp.Exact(
				domain.Value(key),
			); err == nil {
				return fmt.Errorf(
					"duplicate key: %v",
					key,
				)

			} else if !errors.Is(err, common.ErrRecordNotFound) {
				return fmt.Errorf(
					"check duplicate key: %w",
					err,
				)
			}

			if err := tmp.tree.insertPosting(
				key,
				posting,
			); err != nil {
				return fmt.Errorf(
					"insert posting into index: %w",
					err,
				)
			}
			return nil
		},
	)

	if err != nil {
		return fmt.Errorf(
			"range index unmarshal: %w",
			err,
		)
	}

	*i = *tmp
	return nil
}
