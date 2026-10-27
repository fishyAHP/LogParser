package ranged

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"

	indexbin "fishyAHP/LogParser.git/internal/features/index/binary"
	"fishyAHP/LogParser.git/internal/features/index/structs"
)

func (i *Index[K]) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer

	if err := binary.Write(
		&buf,
		binary.BigEndian,
		uint64(i.tree.elemsCount),
	); err != nil {
		return nil, fmt.Errorf(
			"binary write range index elems count: %w",
			err,
		)
	}

	if err := binary.Write(
		&buf,
		binary.BigEndian,
		uint32(i.tree.nodesCount),
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
	return nil
}
