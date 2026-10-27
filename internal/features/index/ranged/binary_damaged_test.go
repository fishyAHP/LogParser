package ranged

import (
	"bytes"
	"cmp"
	"encoding/binary"
	"testing"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/binary"
)

func TestRangeIndexBinaryMalformedEntries(t *testing.T) {
	original := makeTestSnapshot(t)

	// The first entry contains an IntValue key (8 bytes)
	// and a posting list with RecordIDs {1, 2}.
	entryOffset := indexbin.HeaderSize
	keyLengthOffset := entryOffset
	keyLength := binary.BigEndian.Uint32(
		original[keyLengthOffset : keyLengthOffset+4],
	)

	recordsCountOffset := entryOffset +
		indexbin.KeyLengthSize + int(keyLength)

	recordsOffset := recordsCountOffset +
		indexbin.RecordsCountSize

	tests := []struct {
		name   string
		mutate func([]byte)
	}{
		{
			name: "key length exceeds buffer",
			mutate: func(data []byte) {
				binary.BigEndian.PutUint32(
					data[keyLengthOffset:keyLengthOffset+4],
					^uint32(0),
				)
			},
		},
		{
			name: "empty posting list",
			mutate: func(data []byte) {
				binary.BigEndian.PutUint64(
					data[recordsCountOffset:recordsOffset],
					0,
				)
			},
		},
		{
			name: "records count exceeds buffer",
			mutate: func(data []byte) {
				binary.BigEndian.PutUint64(
					data[recordsCountOffset:recordsOffset],
					^uint64(0),
				)
			},
		},
		{
			name: "duplicate record ids",
			mutate: func(data []byte) {
				// Change {1, 2} into {1, 1}.
				copy(
					data[recordsOffset+8:recordsOffset+16],
					data[recordsOffset:recordsOffset+8],
				)
			},
		},
		{
			name: "unsorted record ids",
			mutate: func(data []byte) {
				// Change {1, 2} into {2, 1}.
				first := bytes.Clone(
					data[recordsOffset : recordsOffset+8],
				)
				second := bytes.Clone(
					data[recordsOffset+8 : recordsOffset+16],
				)

				copy(data[recordsOffset:recordsOffset+8], second)
				copy(data[recordsOffset+8:recordsOffset+16], first)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := bytes.Clone(original)
			tt.mutate(data)

			idx := New(cmp.Compare[domain.IntValue])

			if err := idx.UnmarshalBinary(data); err == nil {
				t.Fatal("expected error for corrupted snapshot")
			}
		})
	}
}

func TestRangeIndexBinaryInvalidBoolKey(t *testing.T) {
	compare := func(a, b domain.BoolValue) int {
		switch {
		case a == b:
			return 0
		case !bool(a):
			return -1
		default:
			return 1
		}
	}

	idx := New(compare)

	if err := idx.Add(domain.BoolValue(true), 1); err != nil {
		t.Fatal(err)
	}

	data, err := idx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	keyOffset := indexbin.HeaderSize +
		indexbin.KeyLengthSize

	// BoolValue must be encoded as 0 or 1.
	data[keyOffset] = 0xFF

	restored := New(compare)

	if err := restored.UnmarshalBinary(data); err == nil {
		t.Fatal("expected invalid boolean key error")
	}
}

func TestRangeIndexBinaryPreserveOnCorruption(t *testing.T) {
	original := makeTestSnapshot(t)

	tests := []struct {
		name   string
		mutate func([]byte) []byte
	}{
		{
			name: "truncated last entry",
			mutate: func(data []byte) []byte {
				return data[:len(data)-1]
			},
		},
		{
			name: "wrong records count",
			mutate: func(data []byte) []byte {
				binary.BigEndian.PutUint64(
					data[:indexbin.RecordsCountSize],
					999,
				)
				return data
			},
		},
		{
			name: "trailing bytes",
			mutate: func(data []byte) []byte {
				return append(data, 0xAA)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := New(cmp.Compare[domain.IntValue])

			if err := idx.Add(domain.IntValue(999), 100); err != nil {
				t.Fatal(err)
			}
			if err := idx.Add(domain.IntValue(1000), 101); err != nil {
				t.Fatal(err)
			}

			before, err := idx.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}

			corrupted := tt.mutate(bytes.Clone(original))

			if err := idx.UnmarshalBinary(corrupted); err == nil {
				t.Fatal("expected error")
			}

			after, err := idx.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(before, after) {
				t.Fatal("index state changed after failed restore")
			}

			checkExact(
				t, idx,
				domain.IntValue(999),
				[]domain.RecordID{100},
			)
			checkExact(
				t, idx,
				domain.IntValue(1000),
				[]domain.RecordID{101},
			)
		})
	}
}
