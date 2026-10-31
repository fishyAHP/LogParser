package indexbin

import (
	"fmt"
	"io"
	"iter"
	"math"

	"fishyAHP/LogParser.git/internal/core/binaryio"
	"fishyAHP/LogParser.git/internal/features/index/structs"
)

type Decoder[K comparable] struct {
	decoder *binaryio.Decoder

	header Header
	read   uint32
	total  uint64

	decodeKey func([]byte) (K, error)
}

func NewDecoder[K comparable](
	data []byte,
	decodeKey func([]byte) (K, error),
) (*Decoder[K], error) {
	var header Header

	if err := header.UnmarshalBinary(data); err != nil {
		return nil, err
	}

	return &Decoder[K]{
		decoder:   binaryio.NewDecoder(data[HeaderSize:]),
		header:    header,
		decodeKey: decodeKey,
	}, nil
}

func (d *Decoder[K]) Next() (
	K,
	*structs.PostingList,
	error,
) {
	var zero K

	if d.read == d.header.EntriesCount {
		return zero, nil, io.EOF
	}

	keyLength, err := d.decoder.Uint32()
	if err != nil {
		return zero, nil, fmt.Errorf(
			"decode key length: %w", err,
		)
	}

	if uint64(keyLength) > uint64(d.decoder.Remaining()) {
		return zero, nil, io.ErrUnexpectedEOF
	}

	keyData, err := d.decoder.BytesView(int(keyLength))
	if err != nil {
		return zero, nil, err
	}

	key, err := d.decodeKey(keyData)
	if err != nil {
		return zero, nil, fmt.Errorf("decode key: %w", err)
	}

	recordsCount, err := d.decoder.Uint64()
	if err != nil {
		return zero, nil, err
	}

	if recordsCount == 0 ||
		d.total > d.header.RecordsCount ||
		recordsCount > d.header.RecordsCount-d.total ||
		recordsCount > uint64(d.decoder.Remaining()/8) {
		return zero, nil, fmt.Errorf(
			"invalid records count: %d", recordsCount,
		)
	}

	recordsData, err := d.decoder.BytesView(int(recordsCount * 8))
	if err != nil {
		return zero, nil, err
	}

	posting := structs.NewPostingLists(int(recordsCount))

	if err := posting.UnmarshalBinary(recordsData); err != nil {
		return zero, nil, fmt.Errorf(
			"decode posting: %w", err,
		)
	}

	if uint64(posting.Len()) != recordsCount {
		return zero, nil, fmt.Errorf("posting length mismatch")
	}

	d.total += recordsCount
	d.read++

	return key, posting, nil
}

func (d *Decoder[K]) Finish() error {
	if d.read != d.header.EntriesCount {
		return fmt.Errorf(
			"entries mismatch: expected=%d, actual=%d",
			d.header.EntriesCount,
			d.read,
		)
	}

	if d.total != d.header.RecordsCount {
		return fmt.Errorf(
			"records mismatch: expected=%d, actual=%d",
			d.header.RecordsCount,
			d.total,
		)
	}

	return d.decoder.Finish()
}

func (d *Decoder[K]) Header() Header {
	return d.header
}

func MarshalIndex[K comparable](
	recordsCount uint64,
	entriesCount uint32,
	entries iter.Seq2[K, *structs.PostingList],
	encodeKey func(K) ([]byte, error),
) ([]byte, error) {
	encoder, err := binaryio.NewEncoder(HeaderSize)
	if err != nil {
		return nil, fmt.Errorf(
			"create index encoder: %w",
			err,
		)
	}

	header := Header{
		RecordsCount: recordsCount,
		EntriesCount: entriesCount,
	}

	headerData, err := header.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf(
			"marshal index header: %w",
			err,
		)
	}

	encoder.Bytes(headerData)

	var (
		actualRecords uint64
		actualEntries uint64
	)

	for key, posting := range entries {
		actualEntries++

		if actualEntries > uint64(entriesCount) {
			return nil, fmt.Errorf(
				"entries count exceeds header: %d",
				actualEntries,
			)
		}

		if posting == nil || posting.Len() == 0 {
			return nil, fmt.Errorf("empty posting list")
		}

		keyData, err := encodeKey(key)
		if err != nil {
			return nil, fmt.Errorf("encode key: %w", err)
		}

		if uint64(len(keyData)) > math.MaxUint32 {
			return nil, fmt.Errorf(
				"key length overflow: %d",
				len(keyData),
			)
		}

		postingCount := uint64(posting.Len())

		if actualRecords > recordsCount ||
			postingCount > recordsCount-actualRecords {
			return nil, fmt.Errorf(
				"records count exceeds header",
			)
		}

		encoder.Uint32(uint32(len(keyData)))
		encoder.Bytes(keyData)
		encoder.Uint64(postingCount)

		if err := posting.EncodeTo(encoder); err != nil {
			return nil, fmt.Errorf(
				"encode posting: %w",
				err,
			)
		}

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

	return encoder.Result(), nil
}

func UnmarshalIndex[K comparable](
	data []byte,
	decodeKey func([]byte) (K, error),
	insert func(K, *structs.PostingList) error,
) (Header, error) {
	decoder, err := NewDecoder(data, decodeKey)
	if err != nil {
		return Header{}, err
	}

	header := decoder.Header()

	for range header.EntriesCount {
		key, posting, err := decoder.Next()
		if err != nil {
			return Header{}, err
		}

		if err = insert(key, posting); err != nil {
			return Header{}, fmt.Errorf(
				"insert entry: %w", err,
			)
		}
	}

	if err = decoder.Finish(); err != nil {
		return Header{}, err
	}

	return header, nil
}
