package indexbin

import (
	"encoding/binary"
	"reflect"
	"strconv"
	"testing"
)

func TestHeaderRoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		header Header
	}{
		{
			name: "empty",
			header: Header{
				RecordsCount: 0,
				EntriesCount: 0,
			},
		},
		{
			name: "single entry",
			header: Header{
				RecordsCount: 1,
				EntriesCount: 1,
			},
		},
		{
			name: "multiple entries",
			header: Header{
				RecordsCount: 1000,
				EntriesCount: 10,
			},
		},
		{
			name: "max records count",
			header: Header{
				RecordsCount: ^uint64(0),
				EntriesCount: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.header.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary: %v", err)
			}

			if len(data) != HeaderSize {
				t.Fatalf(
					"header size: got %d, want %d",
					len(data),
					HeaderSize,
				)
			}

			// Header.UnmarshalBinary expects a complete
			// index buffer, not just the header, when
			// EntriesCount is greater than zero.
			full := make(
				[]byte,
				HeaderSize+int(tt.header.EntriesCount)*MinEntrySize,
			)
			copy(full, data)

			var restored Header
			if err := restored.UnmarshalBinary(full); err != nil {
				t.Fatalf("UnmarshalBinary: %v", err)
			}

			if restored != tt.header {
				t.Fatalf(
					"header mismatch: got %+v, want %+v",
					restored,
					tt.header,
				)
			}
		})
	}
}

func TestHeaderTruncated(t *testing.T) {
	header := Header{
		RecordsCount: 100,
		EntriesCount: 5,
	}

	data, err := header.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	for size := range HeaderSize {
		t.Run(
			"length_"+strconv.Itoa(size),
			func(t *testing.T) {
				var restored Header

				if err := restored.UnmarshalBinary(
					data[:size],
				); err == nil {
					t.Fatalf(
						"expected error for length %d",
						size,
					)
				}
			},
		)
	}
}

func TestHeaderInvalidEntriesCount(t *testing.T) {
	tests := []struct {
		name         string
		entriesCount uint32
		remaining    int
	}{
		{
			name:         "one entry without data",
			entriesCount: 1,
			remaining:    0,
		},
		{
			name:         "two entries only one fits",
			entriesCount: 2,
			remaining:    MinEntrySize,
		},
		{
			name:         "entry missing one byte",
			entriesCount: 1,
			remaining:    MinEntrySize - 1,
		},
		{
			name:         "maximum entries count",
			entriesCount: ^uint32(0),
			remaining:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, HeaderSize+tt.remaining)

			binary.BigEndian.PutUint32(
				data[RecordsCountSize:HeaderSize],
				tt.entriesCount,
			)

			var header Header
			if err := header.UnmarshalBinary(data); err == nil {
				t.Fatal("expected invalid entries count error")
			}
		})
	}
}

func TestHeaderPreserveOnError(t *testing.T) {
	original := Header{
		RecordsCount: 123,
		EntriesCount: 3,
	}

	data := make([]byte, HeaderSize)
	binary.BigEndian.PutUint64(
		data[:RecordsCountSize],
		999,
	)
	binary.BigEndian.PutUint32(
		data[RecordsCountSize:HeaderSize],
		10,
	)

	header := original

	if err := header.UnmarshalBinary(data); err == nil {
		t.Fatal("expected error")
	}

	if !reflect.DeepEqual(header, original) {
		t.Fatalf(
			"header modified on error: got %+v, want %+v",
			header,
			original,
		)
	}
}
