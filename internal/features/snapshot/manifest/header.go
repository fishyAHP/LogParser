package manifest

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

type header struct {
	magic      [magicLen]byte
	version    uint16
	generation uint32
	lastID     uint64
	createdAt  int64
	indexCount uint32
}

const headerSize = 6 + 2 + 4 + 8 + 8 + 4

func (m *Manifest) header() (header, error) {
	if m.Version != Version1 {
		return header{}, fmt.Errorf(
			"unsupported manifest version: %d",
			m.Version,
		)
	}

	if m.Generation == 0 {
		return header{}, errors.New(
			"manifest generation must be non-zero",
		)
	}

	if m.CreatedAt.IsZero() {
		return header{}, errors.New(
			"manifest creation time must be non-zero",
		)
	}

	if len(m.indexes) > math.MaxUint32 {
		return header{}, fmt.Errorf(
			"too many indexes in manifest: %d",
			len(m.indexes),
		)
	}

	return header{
		magic:      manifestMagic,
		version:    m.Version,
		generation: m.Generation,
		lastID:     uint64(m.LastIndexedRecordID),
		createdAt:  m.CreatedAt.UnixNano(),
		indexCount: uint32(len(m.indexes)),
	}, nil
}

func (h header) MarshalBinary() ([]byte, error) {
	data := make([]byte, headerSize)
	copy(data, h.magic[:])

	offset := len(manifestMagic)
	binary.BigEndian.PutUint16(
		data[offset:],
		h.version,
	)

	offset += 2
	binary.BigEndian.PutUint32(
		data[offset:],
		h.generation,
	)

	offset += 4
	binary.BigEndian.PutUint64(
		data[offset:],
		h.lastID,
	)

	offset += 8
	binary.BigEndian.PutUint64(
		data[offset:],
		uint64(h.createdAt),
	)

	offset += 8
	binary.BigEndian.PutUint32(
		data[offset:],
		h.indexCount,
	)

	return data, nil
}

func (h *header) UnmarshalBinary(
	data []byte,
) error {
	if h == nil {
		return errors.New(
			"nil manifest header",
		)
	}

	if len(data) < headerSize {
		return fmt.Errorf(
			"manifest header too short got %d, want %d",
			len(data),
			headerSize,
		)
	}

	var magic [magicLen]byte
	copy(magic[:], data[:magicLen])

	if magic != manifestMagic {
		return errors.New(
			"invalid magic numbers",
		)
	}

	offset := len(magic)
	version := binary.BigEndian.Uint16(
		data[offset:],
	)

	if version != Version1 {
		return fmt.Errorf(
			"unsupported manifest version: %d",
			version,
		)
	}

	offset += 2
	generation := binary.BigEndian.Uint32(
		data[offset:],
	)
	if generation == 0 {
		return errors.New(
			"manifest version must be non-zero",
		)
	}

	offset += 4
	lastID := binary.BigEndian.Uint64(
		data[offset:],
	)

	offset += 8
	createdAt := int64(binary.BigEndian.Uint64(
		data[offset:],
	))

	offset += 8
	indexCount := binary.BigEndian.Uint32(
		data[offset:],
	)

	*h = header{
		magic:      magic,
		version:    version,
		generation: generation,
		lastID:     lastID,
		createdAt:  createdAt,
		indexCount: indexCount,
	}

	return nil
}
