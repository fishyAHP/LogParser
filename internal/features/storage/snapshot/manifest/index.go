package manifest

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"

	"fishyAHP/LogParser.git/internal/features/storage/segments"
)

type indexManifest struct {
	name     string
	filename string
	size     segments.FileSize
	sha256   string
}

func newIndex(
	name, filename string,
	size segments.FileSize,
	checkSum string,
) (indexManifest, error) {
	idx := indexManifest{
		name:     name,
		filename: filename,
		size:     size,
		sha256:   checkSum,
	}

	if err := idx.validate(); err != nil {
		return indexManifest{}, err
	}

	return idx, nil
}
func (i indexManifest) MarshalBinary() ([]byte, error) {
	if err := i.validate(); err != nil {
		return nil, err
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

func (i *indexManifest) UnmarshalBinary(
	data []byte,
) error {
	if len(data) < 2 {
		return errors.New("data too short for name length")
	}

	lenName := binary.BigEndian.Uint16(
		data[:2],
	)
	offset := 2

	end := offset + int(lenName)
	if len(data) < end {
		return errors.New("data len too short for name")
	}

	name := string(data[offset:end])

	offset = end

	if len(data)-offset < 2 {
		return errors.New("data too short for filename length")
	}

	lenFilename := binary.BigEndian.Uint16(data[offset:])
	offset += 2

	end = offset + int(lenFilename)
	if len(data) < end {
		return errors.New("data len too short for filename")
	}

	filename := string(data[offset:end])

	offset = end

	end = offset + 8
	if len(data) < end {
		return fmt.Errorf("data len too short for index snapshot size")
	}
	size := binary.BigEndian.Uint64(
		data[offset:end],
	)

	offset = end
	end = offset + 32
	if len(data) < end {
		return errors.New("data len too short for SHA256")
	}

	digest := data[offset:end]
	sha := hex.EncodeToString(digest)

	tmp := indexManifest{
		name:     name,
		filename: filename,
		size:     segments.FileSize(size),
		sha256:   sha,
	}

	if err := tmp.validate(); err != nil {
		return err
	}

	*i = tmp
	return nil
}

func (i *indexManifest) validate() error {
	if strings.TrimSpace(i.name) == "" {
		return errors.New(
			"empty index name",
		)
	}

	if strings.TrimSpace(i.filename) == "" {
		return fmt.Errorf(
			"index %q has empty name",
			i.name,
		)
	}

	return nil
}
