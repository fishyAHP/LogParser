package hash

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/common"
	"fishyAHP/LogParser.git/internal/features/index/structs"
)

type BinaryEntry struct {
	KeyLength    uint32
	Key          []byte
	RecordsCount uint64
	Records      []byte
}

const (
	keyLengthSize = 4
	entriesCountSize
	recordsCountSize = 8
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
		keyLengthSize +
			len(b.Key) +
			recordsCountSize +
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
	if keyLengthSize > dataLength {
		return fmt.Errorf(
			"key size must be shorter than data: %d",
			keyLengthSize,
		)
	}

	keyLength := binary.BigEndian.Uint32(
		data[:keyLengthSize],
	)

	if uint64(keyLength) > uint64(dataLength-keyLengthSize) {
		return fmt.Errorf(
			"invalid ley length: %d",
			keyLength,
		)
	}
	keyEnd := keyLengthSize + keyLength
	if int(keyEnd) >= dataLength {
		return fmt.Errorf(
			"key must be shorter than data: %d",
			keyEnd,
		)
	}

	key := data[keyLengthSize:keyEnd]
	recordsCountEnd := keyEnd + recordsCountSize

	if int(recordsCountEnd) > len(data) {
		return fmt.Errorf(
			"end of records count more than len data: %d",
			recordsCountEnd,
		)
	}

	recordsCount := binary.BigEndian.Uint64(
		data[keyEnd:recordsCountEnd],
	)

	if recordsCount > uint64(
		(dataLength-int(recordsCountEnd))/8,
	) {
		return fmt.Errorf(
			"invalid records count: %d",
			recordsCount,
		)
	}

	recordsSize := recordsCount * 8
	recordsEnd := uint64(recordsCountEnd) + recordsSize

	if int(recordsEnd) > dataLength {
		return fmt.Errorf(
			"records must be equal to len data: %d",
			recordsEnd,
		)
	}

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

func (i *Index[K]) MarshalBinary() ([]byte, error) {
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
		keyData, err := encodeKey(key)
		if err != nil {
			return nil, err
		}

		postData, err := posting.MarshalBinary()
		if err != nil {
			return nil, err
		}

		entry := BinaryEntry{
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
	if len(data) < recordsCountSize {
		return fmt.Errorf(
			"records count overflow data: %d",
			len(data),
		)
	}

	tmp := New[K]()
	count := binary.BigEndian.Uint64(
		data[:recordsCountSize],
	)
	offset := recordsCountSize

	if len(data) < offset+entriesCountSize {
		return fmt.Errorf(
			"entries count overflow data: %d and %d",
			len(data),
			offset+entriesCountSize,
		)
	}
	entriesCount := binary.BigEndian.Uint32(
		data[offset : offset+entriesCountSize],
	)

	if int(entriesCount) != len(tmp.idx) {
		return fmt.Errorf(
			"entries count not consistently: %d and %d",
			entriesCount,
			len(tmp.idx),
		)
	}

	offset += entriesCountSize
	for idx := offset; idx < len(data); {
		var entry BinaryEntry

		if err := entry.UnmarshalBinary(
			data[idx:],
		); err != nil {
			return fmt.Errorf(
				"entry binary unmarshal: %w",
				err,
			)
		}

		key, err := decodeKey[K](
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
		size := keyLengthSize +
			len(entry.Key) +
			recordsCountSize +
			len(entry.Records)

		idx += size
	}

	*i = Index[K]{
		idx:   tmp.idx,
		count: count,
	}

	return nil
}

func encodeKey[K common.Key](
	key K,
) ([]byte, error) {
	switch k := any(key).(type) {
	case domain.StringValue:
		return []byte(k), nil

	case domain.IntValue:
		data := make([]byte, 8)
		binary.BigEndian.PutUint64(
			data,
			uint64(k),
		)

		return data, nil

	case domain.FloatValue:
		data := make([]byte, 8)
		binary.BigEndian.PutUint64(
			data,
			math.Float64bits(float64(k)),
		)

		return data, nil

	case domain.TimeValue:
		t := time.Time(k)
		data, err := t.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf(
				"time marshal binary: %w",
				err,
			)
		}

		return data, nil
	case domain.BoolValue:
		if k {
			return []byte{1}, nil
		}
		return []byte{0}, nil

	default:
		return nil, fmt.Errorf(
			"unsupported key type: %T",
			key,
		)
	}
}

func decodeKey[K common.Key](
	data []byte,
) (K, error) {
	var zero K

	switch k := any(zero).(type) {
	case domain.StringValue:
		return any(domain.StringValue(data)).(K), nil

	case domain.IntValue:
		if len(data) != 8 {
			return zero, fmt.Errorf(
				"invalid int key length: %d",
				len(data),
			)
		}

		value := domain.IntValue(
			binary.BigEndian.Uint64(data),
		)
		return any(value).(K), nil

	case domain.FloatValue:
		if len(data) != 8 {
			return zero, fmt.Errorf(
				"invalid float key value: %d",
				len(data),
			)
		}

		bits := binary.BigEndian.Uint64(data)
		value := domain.FloatValue(
			math.Float64frombits(bits),
		)

		return any(value).(K), nil

	case domain.TimeValue:
		var t time.Time

		if err := t.UnmarshalBinary(
			data,
		); err != nil {
			return zero, fmt.Errorf(
				"time binary unmarshal: %w",
				err,
			)
		}

		return any(domain.TimeValue(t)).(K), nil

	case domain.BoolValue:
		if len(data) != 1 {
			return zero, fmt.Errorf(
				"invalid bool key length: %d",
				len(data),
			)
		}

		switch data[0] {
		case 0:
			return any(domain.BoolValue(false)).(K), nil
		case 1:
			return any(domain.BoolValue(true)).(K), nil
		default:
			return zero, fmt.Errorf(
				"unexpected bool key value: %d",
				data[0],
			)
		}
	default:
		return zero, fmt.Errorf(
			"unsupported key type: %T",
			k,
		)
	}
}
