package ranged

import (
	"bytes"
	"fmt"
	"math"

	indexbin "fishyAHP/LogParser.git/internal/features/index/binary"
	"fishyAHP/LogParser.git/internal/features/index/structs"
)

func (i *Index[K]) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer

	header := indexbin.Header{
		RecordsCount: i.tree.elemsCount,
		EntriesCount: i.tree.nodesCount,
	}

	headerData, err := header.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf(
			"index header marshal binary: %w",
			err,
		)
	}

	if _, err = buf.Write(
		headerData,
	); err != nil {
		return nil, fmt.Errorf(
			"write index header data to buffer: %w",
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
	var header indexbin.Header
	if err := header.UnmarshalBinary(
		data,
	); err != nil {
		return fmt.Errorf(
			"unmarshal binary index header: %w",
			err,
		)
	}

	idx := indexbin.HeaderSize

	for range header.EntriesCount {
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

		if err = tmp.tree.insertPosting(
			key,
			postings,
		); err != nil {
			return fmt.Errorf(
				"insert posting with key='%v': %w",
				key,
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

	if tmp.tree.elemsCount != header.RecordsCount {
		return fmt.Errorf(
			"elements count mismatch: expected %d, got %d",
			header.RecordsCount,
			tmp.tree.elemsCount,
		)
	}

	if tmp.tree.nodesCount != header.EntriesCount {
		return fmt.Errorf(
			"nodes count mismatch: expected %d, got %d",
			header.EntriesCount,
			tmp.tree.nodesCount,
		)
	}

	*i = Index[K]{
		tree: tmp.tree,
	}

	return nil
}
