package hash

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"slices"
	"sync"

	"fishyAHP/LogParser.git/internal/features/index/binary"
	"fishyAHP/LogParser.git/internal/features/index/structs"
)

func (i *Index[K]) MarshalBinary() ([]byte, error) {
	if uint64(len(i.idx)) > math.MaxUint32 {
		return nil, errors.New(
			"entries count overflow",
		)
	}

	type item struct {
		key     K
		encoded []byte
	}

	items := make([]item, 0, len(i.idx))

	for key := range i.idx {
		data, err := indexbin.EncodeKey(key)
		if err != nil {
			return nil, fmt.Errorf(
				"encode key: %w",
				err,
			)
		}

		items = append(items, item{
			key:     key,
			encoded: data,
		})
	}

	slices.SortFunc(items, func(a, b item) int {
		return bytes.Compare(a.encoded, b.encoded)
	})

	entries := func(yield func(K, *structs.PostingList) bool) {
		for _, item := range items {
			if !yield(item.key, i.idx[item.key]) {
				return
			}
		}
	}

	return indexbin.MarshalIndex(
		i.count,
		uint32(len(i.idx)),
		entries,
		indexbin.EncodeKey[K],
	)
}

func (i *Index[K]) UnmarshalBinary(data []byte) error {
	tmp := New[K]()

	header, err := indexbin.UnmarshalIndex(
		data,
		indexbin.DecodeKey[K],
		func(key K, posting *structs.PostingList) error {
			if _, exists := tmp.idx[key]; exists {
				return fmt.Errorf("duplicate key: %v", key)
			}

			tmp.idx[key] = posting
			return nil
		},
	)
	if err != nil {
		return fmt.Errorf("hash index unmarshal: %w", err)
	}

	tmp.count = header.RecordsCount

	*i = Index[K]{
		idx:   tmp.idx,
		count: tmp.count,
		mtx:   sync.RWMutex{},
	}

	return nil
}
