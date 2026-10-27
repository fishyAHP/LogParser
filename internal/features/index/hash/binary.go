package hash

import (
	"bytes"
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
	header := indexbin.Header{
		RecordsCount: i.count,
		EntriesCount: uint32(len(i.idx)),
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
	var actualCount uint64

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

		actualCount += entry.RecordsCount
		idx += size
	}

	if idx != len(data) {
		return fmt.Errorf(
			"unexpected trailing data: %d bytes",
			len(data)-idx,
		)
	}

	if actualCount != header.RecordsCount {
		return fmt.Errorf(
			"records count mismatch: expected %d, got %d",
			header.RecordsCount,
			actualCount,
		)
	}

	*i = Index[K]{
		idx:   tmp.idx,
		count: header.RecordsCount,
	}

	return nil
}
