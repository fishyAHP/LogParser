package indexbin

import (
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"fishyAHP/LogParser.git/internal/core/binaryio"
	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/common"
)

func EncodeKey[K common.Key](
	key K,
) ([]byte, error) {
	switch k := any(key).(type) {
	case domain.StringValue:
		return []byte(k), nil

	case domain.IntValue:
		return binary.BigEndian.AppendUint64(
			nil, uint64(k),
		), nil

	case domain.FloatValue:
		return binary.BigEndian.AppendUint64(
			nil, math.Float64bits(float64(k)),
		), nil

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

func DecodeKey[K common.Key](
	data []byte,
) (K, error) {
	var zero K

	switch k := any(zero).(type) {
	case domain.StringValue:
		return any(domain.StringValue(data)).(K), nil

	case domain.IntValue:
		decoder := binaryio.NewDecoder(data)

		value, err := decoder.Uint64()
		if err != nil {
			return zero, fmt.Errorf(
				"decode int key: %w",
				err,
			)
		}

		if err = decoder.Finish(); err != nil {
			return zero, fmt.Errorf(
				"invalid int key: %w",
				err,
			)
		}
		return any(domain.IntValue(value)).(K), nil

	case domain.FloatValue:
		decoder := binaryio.NewDecoder(data)

		bits, err := decoder.Uint64()
		if err != nil {
			return zero, fmt.Errorf(
				"decode float key: %w",
				err,
			)
		}

		if err = decoder.Finish(); err != nil {
			return zero, fmt.Errorf(
				"invalid float key: %w",
				err,
			)
		}
		return any(
			domain.FloatValue(math.Float64frombits(bits)),
		).(K), nil

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
