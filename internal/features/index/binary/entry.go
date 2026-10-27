package indexbin

import (
	"bytes"
	"encoding/binary"
	"fmt"
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
	if err := b.validate(); err != nil {
		return nil, fmt.Errorf(
			"binary entry validate: %w",
			err,
		)
	}

	var buf bytes.Buffer
	buf.Grow(
		KeyLengthSize +
			len(b.Key) +
			RecordsCountSize +
			len(b.Records),
	)

	if err := binary.Write(
		&buf,
		binary.BigEndian,
		b.KeyLength,
	); err != nil {
		return nil, fmt.Errorf(
			"binary write key length: %w",
			err,
		)
	}

	if _, err := buf.Write(b.Key); err != nil {
		return nil, fmt.Errorf(
			"buffer write key: %w",
			err,
		)
	}

	if err := binary.Write(
		&buf,
		binary.BigEndian,
		b.RecordsCount,
	); err != nil {
		return nil, fmt.Errorf(
			"binary write records count: %w",
			err,
		)
	}

	if _, err := buf.Write(b.Records); err != nil {
		return nil, fmt.Errorf(
			"buffer write records: %w",
			err,
		)
	}

	return buf.Bytes(), nil
}

func (b *BinaryEntry) UnmarshalBinary(
	data []byte,
) error {
	dataLength := len(data)
	if KeyLengthSize > dataLength {
		return fmt.Errorf(
			"key size must be shorter than data: %d",
			KeyLengthSize,
		)
	}

	keyLength := binary.BigEndian.Uint32(
		data[:KeyLengthSize],
	)

	if uint64(keyLength) > uint64(dataLength-KeyLengthSize) {
		return fmt.Errorf(
			"invalid ley length: %d",
			keyLength,
		)
	}
	keyEnd := KeyLengthSize + int(keyLength)
	if dataLength-keyEnd < RecordsCountSize {
		return fmt.Errorf(
			"truncated records count: %d",
			keyEnd+dataLength,
		)
	}

	key := data[KeyLengthSize:keyEnd]
	recordsCountEnd := keyEnd + RecordsCountSize

	if recordsCountEnd > len(data) {
		return fmt.Errorf(
			"end of records count more than len data: %d",
			recordsCountEnd,
		)
	}

	recordsCount := binary.BigEndian.Uint64(
		data[keyEnd:recordsCountEnd],
	)

	if recordsCount > uint64(
		(dataLength-recordsCountEnd)/8,
	) {
		return fmt.Errorf(
			"invalid records count: %d",
			recordsCount,
		)
	}

	recordsEnd := recordsCountEnd + int(recordsCount)*8
	records := data[recordsCountEnd:recordsEnd]

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
