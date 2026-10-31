package manifest

import (
	"bytes"
	"errors"
	"fmt"
	"math"

	"fishyAHP/LogParser.git/internal/core/binaryio"
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

func (h header) EncodeTo(e *binaryio.Encoder) {
	e.Bytes(h.magic[:])
	e.Uint16(h.version)
	e.Uint32(h.generation)
	e.Uint64(h.lastID)
	e.Uint64(uint64(h.createdAt))
	e.Uint32(h.indexCount)
}

func (h header) MarshalBinary() ([]byte, error) {
	e, err := binaryio.NewEncoder(headerSize)
	if err != nil {
		return nil, err
	}

	h.EncodeTo(e)
	return e.Result(), nil
}

func (h *header) DecodeFrom(d *binaryio.Decoder) error {
	if h == nil {
		return errors.New("nil manifest header")
	}

	magic, err := d.BytesView(magicLen)
	if err != nil {
		return fmt.Errorf("decode magic: %w", err)
	}

	if !bytes.Equal(magic, manifestMagic[:]) {
		return errors.New("invalid magic numbers")
	}

	version, err := d.Uint16()
	if err != nil {
		return err
	}
	if version != Version1 {
		return fmt.Errorf("unsupported manifest version: %d", version)
	}

	generation, err := d.Uint32()
	if err != nil {
		return err
	}
	if generation == 0 {
		return errors.New("manifest generation must be non-zero")
	}

	lastID, err := d.Uint64()
	if err != nil {
		return err
	}

	createdAt, err := d.Uint64()
	if err != nil {
		return err
	}

	indexCount, err := d.Uint32()
	if err != nil {
		return err
	}

	*h = header{
		magic:      manifestMagic,
		version:    version,
		generation: generation,
		lastID:     lastID,
		createdAt:  int64(createdAt),
		indexCount: indexCount,
	}

	return nil
}

func (h *header) UnmarshalBinary(data []byte) error {
	d := binaryio.NewDecoder(data)
	return h.DecodeFrom(d)
}
