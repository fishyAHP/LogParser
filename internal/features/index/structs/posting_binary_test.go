package structs

import (
	"encoding/binary"
	"reflect"
	"slices"
	"testing"

	"fishyAHP/LogParser.git/internal/core/domain"
)

func TestPostingListBinaryRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		ids  []domain.RecordID
	}{
		{
			name: "empty list",
			ids:  nil,
		},
		{
			name: "single record",
			ids:  []domain.RecordID{1},
		},
		{
			name: "multiple records",
			ids:  []domain.RecordID{1, 5, 10, 100},
		},
		{
			name: "large record IDs",
			ids:  []domain.RecordID{1, 1000000, 1 << 63, ^domain.RecordID(0)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := PostingList(tt.ids)
			original := &l

			data, err := original.MarshalBinary()
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			if len(data) != len(tt.ids)*8 {
				t.Fatalf(
					"expected %d bytes, got %d",
					len(tt.ids)*8,
					len(data),
				)
			}

			restored := &PostingList{}

			if err := restored.UnmarshalBinary(data); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			if !slices.Equal(([]domain.RecordID)(*restored), tt.ids) {
				t.Errorf(
					"expected %v, got %v",
					tt.ids,
					restored,
				)
			}
		})
	}
}

func TestPostingListUnmarshalInvalidLength(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "one byte",
			data: make([]byte, 1),
		},
		{
			name: "seven bytes",
			data: make([]byte, 7),
		},
		{
			name: "nine bytes",
			data: make([]byte, 9),
		},
		{
			name: "fifteen bytes",
			data: make([]byte, 15),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PostingList{}

			if err := p.UnmarshalBinary(tt.data); err == nil {
				t.Fatal("expected error for invalid data length")
			}
		})
	}
}

func TestPostingListUnmarshalInvalidOrder(t *testing.T) {
	tests := []struct {
		name string
		ids  []uint64
	}{
		{
			name: "descending order",
			ids:  []uint64{10, 5},
		},
		{
			name: "duplicate IDs",
			ids:  []uint64{5, 5},
		},
		{
			name: "invalid middle element",
			ids:  []uint64{1, 10, 7, 20},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, len(tt.ids)*8)

			for i, id := range tt.ids {
				binary.BigEndian.PutUint64(
					data[i*8:(i+1)*8],
					id,
				)
			}

			p := &PostingList{}

			if err := p.UnmarshalBinary(data); err == nil {
				t.Fatal("expected error for invalid record order")
			}
		})
	}
}

func TestPostingListUnmarshalReplacesState(t *testing.T) {
	l := PostingList([]domain.RecordID{1, 2, 3})
	p := &l

	data := make([]byte, 16)

	binary.BigEndian.PutUint64(data[:8], 10)
	binary.BigEndian.PutUint64(data[8:], 20)

	if err := p.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}

	want := PostingList([]domain.RecordID{10, 20})

	if !reflect.DeepEqual(p, &want) {
		t.Errorf("expected %v, got %v", want, p)
	}
}

func TestPostingListUnmarshalStateAfterError(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "invalid length",
			data: make([]byte, 7),
		},
		{
			name: "invalid order",
			data: func() []byte {
				data := make([]byte, 16)
				binary.BigEndian.PutUint64(data[:8], 10)
				binary.BigEndian.PutUint64(data[8:], 5)
				return data
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := PostingList([]domain.RecordID{1, 2, 3})
			p := &l

			if err := p.UnmarshalBinary(tt.data); err == nil {
				t.Fatal("expected error")
			}

			want := PostingList([]domain.RecordID{1, 2, 3})

			if !reflect.DeepEqual(p, &want) {
				t.Errorf(
					"state changed after error: got %v, want %v",
					p,
					want,
				)
			}
		})
	}
}

func TestPostingListBinaryMillionRecords(t *testing.T) {
	if testing.Short() {
		t.Skip("large serialization test")
	}

	const count = 1_000_000

	l := PostingList(make([]domain.RecordID, count))
	original := &l

	for i := range *original {
		(*original)[i] = domain.RecordID(i + 1)
	}

	data, err := original.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	if len(data) != count*8 {
		t.Fatalf(
			"expected %d bytes, got %d",
			count*8,
			len(data),
		)
	}

	restored := &PostingList{}

	if err := restored.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(([]domain.RecordID)(*original), *restored) {
		t.Fatal("restored posting list differs from original")
	}
}
