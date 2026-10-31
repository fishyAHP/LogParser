package binaryio

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type Encoder struct {
	data []byte
	done bool
}

func NewEncoder(
	capacity int,
) (*Encoder, error) {
	if capacity < 0 {
		return nil, fmt.Errorf(
			"negative encoder capacity: %d",
			capacity,
		)
	}

	return &Encoder{
		data: make([]byte, 0, capacity),
	}, nil
}

func (e *Encoder) Uint8(v uint8) {
	e.data = append(e.data, v)
}

func (e *Encoder) Uint16(v uint16) {
	e.data = binary.BigEndian.AppendUint16(e.data, v)
}

func (e *Encoder) Uint32(v uint32) {
	e.data = binary.BigEndian.AppendUint32(e.data, v)
}

func (e *Encoder) Uint64(v uint64) {
	e.data = binary.BigEndian.AppendUint64(e.data, v)
}

func (e *Encoder) String(v string) {
	e.data = append(e.data, v...)
}

func (e *Encoder) Bytes(v []byte) {
	e.data = append(e.data, v...)
}

func (e Encoder) Result() []byte {
	return bytes.Clone(e.data)
}
