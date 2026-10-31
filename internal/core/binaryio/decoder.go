package binaryio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	uint8Size  = 1
	uint16Size = 2
	uint32Size = 4
	uint64Size = 8
)

type Decoder struct {
	data   []byte
	offset int
}

func NewDecoder(
	data []byte,
) *Decoder {
	return &Decoder{
		data: data,
	}
}

func (d *Decoder) read(length int) ([]byte, error) {
	if length < 0 {
		return nil, fmt.Errorf(
			"negative read length: %d",
			length,
		)
	}

	if d.offset < 0 ||
		d.offset > len(d.data) ||
		length > d.Remaining() {
		return nil, io.ErrUnexpectedEOF
	}

	value := d.data[d.offset : d.offset+length]
	d.offset += length

	return value, nil
}

func (d *Decoder) Uint8() (uint8, error) {
	data, err := d.read(uint8Size)
	if err != nil {
		return 0, err
	}

	return data[0], nil
}

func (d *Decoder) Uint16() (uint16, error) {
	data, err := d.read(uint16Size)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint16(data), nil
}

func (d *Decoder) Uint32() (uint32, error) {
	data, err := d.read(uint32Size)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint32(data), nil
}

func (d *Decoder) Uint64() (uint64, error) {
	data, err := d.read(uint64Size)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint64(data), nil
}

func (d *Decoder) String(
	length int,
) (string, error) {
	data, err := d.read(length)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (d *Decoder) Bytes(length int) ([]byte, error) {
	data, err := d.read(length)
	if err != nil {
		return nil, err
	}

	return bytes.Clone(data), nil
}

func (d *Decoder) BytesView(length int) ([]byte, error) {
	return d.read(length)
}

func (d Decoder) Remaining() int {
	return len(d.data) - d.offset
}

func (d Decoder) Finish() error {
	if d.offset != len(d.data) {
		return fmt.Errorf(
			"unexpected trailing data: %d",
			d.Remaining(),
		)
	}

	return nil
}
