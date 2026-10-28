package manifest

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
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
	Indexes             []indexManifest
}

type indexManifest struct {
	name     string
	filename string
	size     segments.FileSize
	sha256   string
}

func (i *indexManifest) MarshalBinary() ([]byte, error) {
	if i == nil {
		return nil, errors.New("nil index manifest")
	}

	var buf bytes.Buffer

	for _, str := range []string{
		i.name,
		i.filename,
	} {
		if len(str) > math.MaxUint16 {
			return nil, fmt.Errorf(
				"string too long: %d bytes",
				len(str),
			)
		}

		if err := binary.Write(
			&buf,
			binary.BigEndian,
			uint16(len(str)),
		); err != nil {
			return nil, fmt.Errorf(
				"binary write string len: %w",

				err,
			)
		}

		if _, err := buf.Write(
			[]byte(str),
		); err != nil {
			return nil, fmt.Errorf(
				"buffer write string: %w",
				err,
			)
		}
	}

	if err := binary.Write(
		&buf,
		binary.BigEndian,
		uint64(i.size),
	); err != nil {
		return nil, fmt.Errorf(
			"binary write file size: %w",
			err,
		)
	}

	digest, err := hex.DecodeString(i.sha256)
	if err != nil {
		return nil, fmt.Errorf(
			"decode SHA256: %w",
			err,
		)
	}

	if len(digest) != sha256.Size {
		return nil, fmt.Errorf(
			"invalid SHA256 length: %d",
			len(digest),
		)
	}

	if _, err = buf.Write(digest); err != nil {
		return nil, fmt.Errorf(
			"buffer write SHA256: %w",
			err,
		)
	}

	return buf.Bytes(), nil
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

	for _, idx := range m.Indexes {
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

}
