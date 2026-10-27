package indexbin

import (
	"encoding/binary"
	"fmt"
)

const (
	HeaderSize   = RecordsCountSize + EntriesCountSize
	MinEntrySize = KeyLengthSize + RecordsCountSize
)

type Header struct {
	RecordsCount uint64
	EntriesCount uint32
}

func (h *Header) MarshalBinary() ([]byte, error) {
	data := make([]byte, HeaderSize)

	binary.BigEndian.PutUint64(
		data[:RecordsCountSize],
		h.RecordsCount,
	)

	binary.BigEndian.PutUint32(
		data[RecordsCountSize:HeaderSize],
		h.EntriesCount,
	)

	return data, nil
}

func (h *Header) UnmarshalBinary(
	data []byte,
) error {
	if len(data) < HeaderSize {
		return fmt.Errorf(
			"index header too short: got %d, want %d",
			len(data),
			HeaderSize,
		)
	}

	tmp := Header{
		RecordsCount: binary.BigEndian.Uint64(
			data[:RecordsCountSize],
		),
		EntriesCount: binary.BigEndian.Uint32(
			data[RecordsCountSize:HeaderSize],
		),
	}

	if uint64(tmp.EntriesCount) >
		uint64(len(data)-HeaderSize)/MinEntrySize {
		return fmt.Errorf(
			"invalid entries count: %d",
			tmp.EntriesCount,
		)
	}

	*h = tmp

	return nil
}
