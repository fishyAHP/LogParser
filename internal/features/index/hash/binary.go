package hash

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"fishyAHP/LogParser.git/internal/features/index/binary"
	"fishyAHP/LogParser.git/internal/features/index/structs"
)

func (i *Index[K]) MarshalBinary() ([]byte, error) {
	if uint64(len(i.idx)) > math.MaxUint32 {
		return nil, errors.New("too many index entries")
	}

	var buf bytes.Buffer

	if err := binary.Write(
		&buf,
		binary.BigEndian,
		i.count,
	); err != nil {
		return nil, fmt.Errorf(
			"binary write index records count: %w",
			err,
		)
	}
	if err := binary.Write(
		&buf,
		binary.BigEndian,
		uint32(len(i.idx)),
	); err != nil {
		return nil, fmt.Errorf(
			"binary write index entries count: %w",
			err,
		)
	}

	for key, posting := range i.idx {
		keyData, err := indexbin.EncodeKey(key)
		if err != nil {
			return nil, err
		}
		if uint64(len(keyData)) > math.MaxUint32 {
			return nil, errors.New("too many index entries")
		}

		postData, err := posting.MarshalBinary()
		if err != nil {
			return nil, err
		}

		entry := indexbin.BinaryEntry{
			KeyLength:    uint32(len(keyData)),
			Key:          keyData,
			RecordsCount: uint64(posting.Len()),
			Records:      postData,
		}

		entryData, err := entry.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf(
				"marshal binary entry: %w",
				err,
			)
		}

		if _, err = buf.Write(entryData); err != nil {
			return nil, fmt.Errorf(
				"buffer write entry data: %w",
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

	tmp := New[K]()
	count := binary.BigEndian.Uint64(
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

		if _, ok := tmp.idx[key]; ok {
			return fmt.Errorf(
				"index with key='%v' already exist",
				key,
			)
		}

		tmp.idx[key] = postings
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

	*i = Index[K]{
		idx:   tmp.idx,
		count: count,
	}

	return nil
}
