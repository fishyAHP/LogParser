package manifest

import (
	"fmt"
	"iter"
	"time"

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
			"create manifest header: %w",
			err,
		)
	}

	data, err := h.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf(
			"header binary marshal: %w",
			err,
		)
	}

	for _, idx := range m.indexes {
		indexData, err := idx.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf(
				"index %q binary marshal: %w",
				idx.name,
				err,
			)
		}

		data = append(data, indexData...)
	}

	return data, nil
}

func (m *Manifest) UnmarshalBinary(
	data []byte,
) error {
	var h header
	if err := h.UnmarshalBinary(
		data,
	); err != nil {
		return fmt.Errorf(
			"manifest header binary unmarshal: %w",
			err,
		)
	}

	remaining := len(data) - headerSize
	if uint64(h.indexCount) > uint64(remaining/44) {
		return fmt.Errorf(
			"index count %d exceeds available data",
			h.indexCount,
		)
	}

	indexes := make([]indexManifest, 0, h.indexCount)
	offset := headerSize

	for range int(h.indexCount) {
		var idx indexManifest

		if err := idx.UnmarshalBinary(
			data[offset:],
		); err != nil {
			return fmt.Errorf(
				"index manifest binary unmarshaling: %w",
				err,
			)
		}

		consumed := 2 + len(idx.name) +
			2 + len(idx.filename) +
			8 + 32

		offset += consumed

		indexes = append(indexes, idx)
	}

	if offset != len(data) {
		return fmt.Errorf(
			"trailing data in manifest: %d bytes",
			len(data)-offset,
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
