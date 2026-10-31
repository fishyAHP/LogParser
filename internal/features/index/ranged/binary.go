package ranged

import (
	"errors"
	"fmt"
	"io"
	"iter"

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

	var header indexbin.Header
	if err := header.UnmarshalBinary(data); err != nil {
		return fmt.Errorf("unmarshal header: %w", err)
	}

	next, stop := iter.Pull2(
		indexbin.IterateIndex(
			data,
			indexbin.DecodeKey[K],
		),
	)
	defer stop()

	var (
		previous K
		hasPrev  bool
		total    uint64
	)

	err := tmp.tree.buildFromSorted(
		int(header.EntriesCount),
		func() (K, *structs.PostingList, error) {
			entry, decodeErr, ok := next()

			if decodeErr != nil {
				var zero K
				return zero, nil, decodeErr
			}

			if !ok {
				var zero K
				return zero, nil, io.ErrUnexpectedEOF
			}

			if hasPrev && tmp.tree.compare(previous, entry.Key) >= 0 {
				var zero K
				return zero, nil, fmt.Errorf(
					"unordered or duplicate key",
				)
			}

			previous = entry.Key
			hasPrev = true
			total += uint64(entry.Posting.Len())

			return entry.Key, entry.Posting, nil
		},
	)
	if err != nil {
		return fmt.Errorf(
			"build tree: %w",
			err,
		)
	}

	_, err, ok := next()
	if err != nil {
		return fmt.Errorf(
			"finish decoding: %w",
			err,
		)
	}
	if ok {
		return errors.New("unexpected extra entry")
	}

	if total != header.RecordsCount {
		return fmt.Errorf(
			"records count mismatch: expected=%d, actual=%d",
			header.RecordsCount,
			total,
		)
	}

	tmp.tree.nodesCount = header.EntriesCount
	tmp.tree.elemsCount = total

	*i = *tmp
	return nil
}
