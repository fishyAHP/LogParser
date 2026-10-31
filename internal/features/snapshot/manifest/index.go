package manifest

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"

	"fishyAHP/LogParser.git/internal/core/binaryio"
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

func (i indexManifest) EncodeTo(e *binaryio.Encoder) error {
	if err := i.validate(); err != nil {
		return err
	}

	for _, value := range []string{i.name, i.filename} {
		if len(value) > math.MaxUint16 {
			return fmt.Errorf(
				"string too long: %d bytes",
				len(value),
			)
		}

		e.Uint16(uint16(len(value)))
		e.String(value)
	}

	e.Uint64(uint64(i.size))

	digest, err := hex.DecodeString(i.sha256)
	if err != nil {
		return fmt.Errorf("decode SHA256: %w", err)
	}

	if len(digest) != sha256.Size {
		return fmt.Errorf(
			"invalid SHA256 length: %d",
			len(digest),
		)
	}

	e.Bytes(digest)
	return nil
}

func (i indexManifest) MarshalBinary() ([]byte, error) {
	e, err := binaryio.NewEncoder(0)
	if err != nil {
		return nil, err
	}

	if err := i.EncodeTo(e); err != nil {
		return nil, err
	}

	return e.Result(), nil
}

func (i *indexManifest) DecodeFrom(
	d *binaryio.Decoder,
) error {
	nameLength, err := d.Uint16()
	if err != nil {
		return fmt.Errorf("decode name length: %w", err)
	}

	name, err := d.String(int(nameLength))
	if err != nil {
		return fmt.Errorf("decode name: %w", err)
	}

	filenameLength, err := d.Uint16()
	if err != nil {
		return fmt.Errorf("decode filename length: %w", err)
	}

	filename, err := d.String(int(filenameLength))
	if err != nil {
		return fmt.Errorf("decode filename: %w", err)
	}

	size, err := d.Uint64()
	if err != nil {
		return fmt.Errorf("decode size: %w", err)
	}

	digest, err := d.BytesView(sha256.Size)
	if err != nil {
		return fmt.Errorf("decode SHA256: %w", err)
	}

	tmp := indexManifest{
		name:     name,
		filename: filename,
		size:     segments.FileSize(size),
		sha256:   hex.EncodeToString(digest),
	}

	if err := tmp.validate(); err != nil {
		return err
	}

	*i = tmp
	return nil
}

func (i *indexManifest) UnmarshalBinary(
	data []byte,
) error {
	d := binaryio.NewDecoder(data)
	return i.DecodeFrom(d)
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
