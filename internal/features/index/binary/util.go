package indexbin

import (
	"bytes"
	"errors"
	"fmt"
	"iter"
	"math"

	"fishyAHP/LogParser.git/internal/features/index/structs"
)

func MarshalIndex[K comparable](
	recordsCount uint64,
	entriesCount uint32,
	entries iter.Seq2[K, *structs.PostingList],
	encodeKey func(K) ([]byte, error),
) ([]byte, error) {
	var buf bytes.Buffer

	header := Header{
		RecordsCount: recordsCount,
		EntriesCount: entriesCount,
	}

	headerData, err := header.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("marshal header: %w", err)
	}

	buf.Write(headerData)

	var (
		actualRecords uint64
		actualEntries uint64
	)

	for key, posting := range entries {
		actualEntries++

		if actualEntries > uint64(entriesCount) {
			return nil, errors.New(
				"entries count exceeds header",
			)
		}

		if posting == nil || posting.Len() == 0 {
			return nil, errors.New(
				"empty posting list",
			)
		}

		keyData, err := encodeKey(key)
		if err != nil {
			return nil, fmt.Errorf(
				"encode key: %w",
				err,
			)
		}

		if uint64(len(keyData)) > math.MaxUint32 {
			return nil, fmt.Errorf(
				"key length overflow: %d",
				len(keyData),
			)
		}

		postingCount := uint64(posting.Len())
		if postingCount > recordsCount-actualRecords {
			return nil, errors.New(
				"records count exceeds header",
			)
		}

		postingData, err := posting.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf(
				"marshal posting: %w",
				err,
			)
		}

		entry := BinaryEntry{
			KeyLength:    uint32(len(keyData)),
			Key:          keyData,
			RecordsCount: postingCount,
			Records:      postingData,
		}

		entryData, err := entry.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf(
				"marshal entry: %w",
				err,
			)
		}

		buf.Write(entryData)
		actualRecords += postingCount
	}

	if actualEntries != uint64(entriesCount) {
		return nil, fmt.Errorf(
			"entries count mismatch: expected=%d, actual=%d",
			entriesCount,
			actualEntries,
		)
	}

	if actualRecords != recordsCount {
		return nil, fmt.Errorf(
			"records count mismatch: expected=%d, actual=%d",
			recordsCount,
			actualRecords,
		)
	}

	return buf.Bytes(), nil
}

func UnmarshalIndex[K comparable](
	data []byte,
	decodeKey func([]byte) (K, error),
	insert func(K, *structs.PostingList) error,
) (Header, error) {
	var header Header

	if err := header.UnmarshalBinary(
		data,
	); err != nil {
		return Header{}, fmt.Errorf(
			"unmarshal index header: %w",
			err,
		)
	}

	offset := HeaderSize
	var actualRecords uint64

	for entryIndex := range header.EntriesCount {
		var entry BinaryEntry

		if err := entry.UnmarshalBinary(
			data[offset:],
		); err != nil {
			return Header{}, fmt.Errorf(
				"unmarshal entry %d: %w",
				entryIndex, err,
			)
		}

		key, err := decodeKey(entry.Key)
		if err != nil {
			return Header{}, fmt.Errorf(
				"decode entry %d key: %w",
				entryIndex, err,
			)
		}

		if entry.RecordsCount == 0 {
			return Header{}, fmt.Errorf(
				"empty posting list in entry %d",
				entryIndex,
			)
		}

		if entry.RecordsCount > uint64(len(entry.Records))/8 {
			return Header{}, fmt.Errorf(
				"posting count exceeds available bytes in entry %d",
				entryIndex,
			)
		}

		if actualRecords > header.RecordsCount ||
			entry.RecordsCount > header.RecordsCount-actualRecords {
			return Header{}, fmt.Errorf(
				"records count exceeds header in entry %d",
				entryIndex,
			)
		}

		posting := structs.NewPostingLists(
			int(entry.RecordsCount),
		)

		if err = posting.UnmarshalBinary(entry.Records); err != nil {
			return Header{}, fmt.Errorf(
				"unmarshal posting in entry %d: %w",
				entryIndex,
				err,
			)
		}

		if posting.Len() != int(entry.RecordsCount) {
			return Header{}, fmt.Errorf(
				"posting length mismatch in entry %d",
				entryIndex,
			)
		}

		if err = insert(
			key,
			posting,
		); err != nil {
			return Header{}, fmt.Errorf(
				"insert entry %d: %w",
				entryIndex,
				err,
			)
		}

		offset += KeyLengthSize +
			len(entry.Key) +
			RecordsCountSize +
			len(entry.Records)

		actualRecords += entry.RecordsCount
	}

	if offset != len(data) {
		return Header{}, fmt.Errorf(
			"unexpected trailing data: %d bytes",
			len(data)-offset,
		)
	}

	if actualRecords != header.RecordsCount {
		return Header{}, fmt.Errorf(
			"records count mismatch: expected=%d, actual=%d",
			header.RecordsCount,
			actualRecords,
		)
	}

	return header, nil
}

func IterateIndex[K comparable](
	data []byte,
	decodeKey func([]byte) (K, error),
) iter.Seq2[Entry[K], error] {
	return func(yield func(Entry[K], error) bool) {
		var header Header

		if err := header.UnmarshalBinary(data); err != nil {
			yield(Entry[K]{}, fmt.Errorf(
				"unmarshal header: %w", err,
			))
			return
		}

		offset := HeaderSize
		var actualRecords uint64

		for entryIndex := range header.EntriesCount {
			var entry BinaryEntry

			if err := entry.UnmarshalBinary(data[offset:]); err != nil {
				yield(Entry[K]{}, fmt.Errorf(
					"unmarshal entry %d: %w",
					entryIndex, err,
				))
				return
			}

			key, err := decodeKey(entry.Key)
			if err != nil {
				yield(Entry[K]{}, fmt.Errorf(
					"decode key: %w", err,
				))
				return
			}

			if entry.RecordsCount == 0 ||
				entry.RecordsCount > uint64(len(entry.Records))/8 ||
				actualRecords > header.RecordsCount ||
				entry.RecordsCount > header.RecordsCount-actualRecords {
				yield(Entry[K]{}, fmt.Errorf(
					"invalid posting count in entry %d",
					entryIndex,
				))
				return
			}

			posting := structs.NewPostingLists(int(entry.RecordsCount))

			if err := posting.UnmarshalBinary(entry.Records); err != nil {
				yield(Entry[K]{}, err)
				return
			}

			if uint64(posting.Len()) != entry.RecordsCount {
				yield(Entry[K]{}, fmt.Errorf(
					"posting length mismatch",
				))
				return
			}

			offset += KeyLengthSize +
				len(entry.Key) +
				RecordsCountSize +
				len(entry.Records)

			actualRecords += entry.RecordsCount

			if !yield(Entry[K]{
				Key:     key,
				Posting: posting,
			}, nil) {
				return
			}
		}

		if offset != len(data) {
			yield(Entry[K]{}, fmt.Errorf(
				"unexpected trailing data",
			))
			return
		}

		if actualRecords != header.RecordsCount {
			yield(Entry[K]{}, fmt.Errorf(
				"records count mismatch",
			))
		}
	}
}

type Entry[K comparable] struct {
	Key     K
	Posting *structs.PostingList
}
