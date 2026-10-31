package indexbin

import (
	"fmt"

	"fishyAHP/LogParser.git/internal/core/binaryio"
)

type BinaryEntry struct {
	KeyLength    uint32
	Key          []byte
	RecordsCount uint64
	Records      []byte
}

const (
	KeyLengthSize = 4
	EntriesCountSize
	RecordsCountSize = 8
)

func (b BinaryEntry) validate() error {
	if uint64(len(b.Key)) !=
		uint64(b.KeyLength) {
		return fmt.Errorf(
			"len of key not consistently: %d and %d",
			len(b.Key),
			b.KeyLength,
		)
	}

	if len(b.Records)%8 != 0 ||
		uint64(len(b.Records)/8) !=
			b.RecordsCount {
		return fmt.Errorf(
			"records count not consistently: %d and %d",
			uint64(len(b.Records)/8),
			b.RecordsCount,
		)
	}

	return nil
}

func (b BinaryEntry) MarshalBinary() ([]byte, error) {
	encoder, err := binaryio.NewEncoder(
		KeyLengthSize +
			len(b.Key) +
			RecordsCountSize +
			len(b.Records),
	)
	if err != nil {
		return nil, err
	}

	if err := b.EncodeTo(encoder); err != nil {
		return nil, err
	}

	return encoder.Result(), nil
}

func (b *BinaryEntry) UnmarshalBinary(
	data []byte,
) error {
	decoder := binaryio.NewDecoder(data)

	keyLength, err := decoder.Uint32()
	if err != nil {
		return fmt.Errorf(
			"decode key length: %w",
			err,
		)
	}

	if uint64(keyLength) > uint64(decoder.Remaining()) {
		return fmt.Errorf(
			"invalid ley length: %d",
			keyLength,
		)
	}

	key, err := decoder.BytesView(int(keyLength))
	if err != nil {
		return fmt.Errorf(
			"decode key: %w",
			err,
		)
	}

	recordsCount, err := decoder.Uint64()
	if err != nil {
		return fmt.Errorf(
			"decoder records count: %w",
			err,
		)
	}

	if recordsCount > uint64(decoder.Remaining()/8) {
		return fmt.Errorf(
			"invalid records count: %d",
			recordsCount,
		)
	}

	records, err := decoder.BytesView(int(recordsCount * 8))
	if err != nil {
		return fmt.Errorf(
			"decode records: %w",
			err,
		)
	}

	tmp := BinaryEntry{
		KeyLength:    keyLength,
		Key:          key,
		RecordsCount: recordsCount,
		Records:      records,
	}

	if err := tmp.validate(); err != nil {
		return fmt.Errorf(
			"binary entry validate: %w",
			err,
		)
	}

	*b = tmp

	return nil
}

func (b BinaryEntry) EncodeTo(
	encoder *binaryio.Encoder,
) error {
	if err := b.validate(); err != nil {
		return fmt.Errorf(
			"binary entry validate: %w",
			err,
		)
	}

	encoder.Uint32(b.KeyLength)
	encoder.Bytes(b.Key)
	encoder.Uint64(b.RecordsCount)
	encoder.Bytes(b.Records)

	return nil
}
