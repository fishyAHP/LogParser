package indexbin

import (
	"fmt"

	"fishyAHP/LogParser.git/internal/core/binaryio"
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
	encoder, err := binaryio.NewEncoder(HeaderSize)
	if err != nil {
		return nil, err
	}

	encoder.Uint64(h.RecordsCount)
	encoder.Uint32(h.EntriesCount)

	return encoder.Result(), nil
}

func (h *Header) UnmarshalBinary(
	data []byte,
) error {
	decoder := binaryio.NewDecoder(data)

	recordsCount, err := decoder.Uint64()
	if err != nil {
		return fmt.Errorf(
			"decode records count: %w",
			err,
		)
	}

	entriesCount, err := decoder.Uint32()
	if err != nil {
		return fmt.Errorf(
			"decode entries count: %w",
			err,
		)
	}

	tmp := Header{
		RecordsCount: recordsCount,
		EntriesCount: entriesCount,
	}

	if uint64(tmp.EntriesCount) >
		uint64(decoder.Remaining())/MinEntrySize {
		return fmt.Errorf(
			"invalid entries count: %d",
			tmp.EntriesCount,
		)
	}

	*h = tmp
	return nil
}
