package manifest

import (
	"fmt"
	"iter"
	"time"

	"fishyAHP/LogParser.git/internal/core/binaryio"
	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/storage/segments"
)

const (
	Version1 uint16 = iota + 1
	Version2
)

var manifestMagic = [...]byte{'S', 'M', 'N', 'F', 'S', 'T'}

const magicLen = 6

type Manifest struct {
	Version             uint16
	Generation          uint32
	LastIndexedRecordID domain.RecordID
	CreatedAt           time.Time
	indexes             []indexManifest
}

func (m *Manifest) AddIndex(
	name, filename string,
	size segments.FileSize,
	checkSum string,
) error {
	idx, err := newIndex(name, filename, size, checkSum)
	if err != nil {
		return fmt.Errorf(
			"creating index manifest: %w",
			err,
		)
	}

	m.indexes = append(m.indexes, idx)
	return nil
}

func (m *Manifest) IndexCount() int {
	return len(m.indexes)
}

func (m *Manifest) MarshalBinary() ([]byte, error) {
	h, err := m.header()
	if err != nil {
		return nil, fmt.Errorf(
			"create manifest header: %w", err,
		)
	}

	encoder, err := binaryio.NewEncoder(headerSize)
	if err != nil {
		return nil, err
	}

	h.EncodeTo(encoder)

	for _, idx := range m.indexes {
		if err := idx.EncodeTo(encoder); err != nil {
			return nil, fmt.Errorf(
				"encode index %q: %w",
				idx.name, err,
			)
		}
	}

	return encoder.Result(), nil
}

func (m *Manifest) UnmarshalBinary(data []byte) error {
	decoder := binaryio.NewDecoder(data)

	var h header
	if err := h.DecodeFrom(decoder); err != nil {
		return fmt.Errorf(
			"decode manifest header: %w", err,
		)
	}

	const minIndexSize = 2 + 2 + 8 + 32

	if uint64(h.indexCount) >
		uint64(decoder.Remaining()/minIndexSize) {
		return fmt.Errorf(
			"index count %d exceeds available data",
			h.indexCount,
		)
	}

	indexes := make([]indexManifest, 0, h.indexCount)

	for n := range h.indexCount {
		var idx indexManifest

		if err := idx.DecodeFrom(decoder); err != nil {
			return fmt.Errorf(
				"decode index %d: %w", n, err,
			)
		}

		indexes = append(indexes, idx)
	}

	if err := decoder.Finish(); err != nil {
		return fmt.Errorf(
			"finish manifest decoding: %w", err,
		)
	}

	*m = Manifest{
		Version:             h.version,
		Generation:          h.generation,
		LastIndexedRecordID: domain.RecordID(h.lastID),
		CreatedAt:           time.Unix(0, h.createdAt).UTC(),
		indexes:             indexes,
	}

	return nil
}

type IndexInfo struct {
	Name     string
	Filename string
	SHA256   string
	Size     segments.FileSize
}

func (m *Manifest) Indexes() iter.Seq[IndexInfo] {
	return func(yield func(IndexInfo) bool) {
		for _, index := range m.indexes {
			info := IndexInfo{
				Name:     index.name,
				Filename: index.filename,
				SHA256:   index.sha256,
				Size:     index.size,
			}

			if !yield(info) {
				return
			}
		}
	}
}
