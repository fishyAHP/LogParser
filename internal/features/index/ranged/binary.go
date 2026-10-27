package ranged

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	indexbin "fishyAHP/LogParser.git/internal/features/index/binary"
	"fishyAHP/LogParser.git/internal/features/index/common"
	"fishyAHP/LogParser.git/internal/features/index/structs"
)

func (i *Index[K]) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer

	if err := binary.Write(
		&buf,
		binary.BigEndian,
		i.tree.elemsCount,
	); err != nil {
		return nil, fmt.Errorf(
			"binary write range index elems count: %w",
			err,
		)
	}

	if err := binary.Write(
		&buf,
		binary.BigEndian,
		i.tree.nodesCount,
	); err != nil {
		return nil, fmt.Errorf(
			"binary write range index entries count: %w",
			err,
		)
	}

	for n := range i.tree.traverse(ascending) {
		key := n.key
		ids := structs.NewPostingFromSorted(
			n.records.slice(),
		)

		keyData, err := indexbin.EncodeKey(key)
		if err != nil {
			return nil, fmt.Errorf(
				"binary encoding key: %w",
				err,
			)
		}

		if uint64(len(keyData)) > math.MaxUint32 {
			return nil, fmt.Errorf(
				"key too large: %d",
				len(keyData),
			)
		}

		recordsData, err := ids.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf(
				"posting binary marshal: %w",
				err,
			)
		}

		entry := indexbin.BinaryEntry{
			KeyLength:    uint32(len(keyData)),
			Key:          keyData,
			RecordsCount: uint64(ids.Len()),
			Records:      recordsData,
		}

		data, err := entry.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf(
				"entry binary marshal: %w",
				err,
			)
		}

		if _, err = buf.Write(
			data,
		); err != nil {
			return nil, fmt.Errorf(
				"buffer write entry with key='%v': %w",
				key,
				err,
			)
		}
	}

	return buf.Bytes(), nil
}

func (i *Index[K]) UnmarshalBinary(
	data []byte,
) error {
	if len(data) < indexbin.RecordsCountSize {
		return fmt.Errorf(
			"records count overflow data: %d",
			len(data),
		)
	}

	tmp := New(i.tree.compare)
	elemsCount := binary.BigEndian.Uint64(
		data[:indexbin.RecordsCountSize],
	)
	offset := indexbin.RecordsCountSize

	if len(data) < offset+indexbin.EntriesCountSize {
		return fmt.Errorf(
			"entries count overflow data: %d and %d",
			len(data),
			offset+indexbin.EntriesCountSize,
		)
	}
	entriesCount := binary.BigEndian.Uint32(
		data[offset : offset+indexbin.EntriesCountSize],
	)

	offset += indexbin.EntriesCountSize
	idx := offset

	if uint64(entriesCount) >
		uint64((len(data)-offset)/12) {
		return fmt.Errorf(
			"invalid entries count: %d",
			entriesCount,
		)
	}

	for range entriesCount {
		var entry indexbin.BinaryEntry

		if err := entry.UnmarshalBinary(
			data[idx:],
		); err != nil {
			return fmt.Errorf(
				"entry binary unmarshal: %w",
				err,
			)
		}

		key, err := indexbin.DecodeKey[K](
			entry.Key,
		)
		if err != nil {
			return fmt.Errorf(
				"decode key: %w",
				err,
			)
		}

		if entry.RecordsCount == 0 {
			return fmt.Errorf(
				"empty posting list for key=%v",
				key,
			)
		}

		postings := structs.NewPostingLists(
			int(entry.RecordsCount),
		)
		if err = postings.UnmarshalBinary(
			entry.Records,
		); err != nil {
			return fmt.Errorf(
				"posting unmarshal binary: %w",
				err,
			)
		}

		_, err = tmp.Exact(key)
		switch {
		case err == nil:
			return fmt.Errorf(
				"duplicate index key: %v",
				key,
			)

		case errors.Is(err, common.ErrRecordNotFound):
			if err = tmp.tree.insertPosting(
				key,
				postings,
			); err != nil {
				return fmt.Errorf(
					"temporary tree add: %w",
					err,
				)
			}

		default:
			return fmt.Errorf(
				"check index key: %w",
				err,
			)
		}

		size := indexbin.KeyLengthSize +
			len(entry.Key) +
			indexbin.RecordsCountSize +
			len(entry.Records)

		idx += size
	}

	if idx != len(data) {
		return fmt.Errorf(
			"unexpected trailing data: %d bytes",
			len(data)-idx,
		)
	}

	if tmp.tree.elemsCount != elemsCount {
		return fmt.Errorf(
			"elements count mismatch: expected %d, got %d",
			elemsCount,
			tmp.tree.elemsCount,
		)
	}

	if tmp.tree.nodesCount != entriesCount {
		return fmt.Errorf(
			"nodes count mismatch: expected %d, got %d",
			entriesCount,
			tmp.tree.nodesCount,
		)
	}

	*i = Index[K]{
		tree: tmp.tree,
	}

	return nil
}
