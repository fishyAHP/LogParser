package indexbin

import "testing"

func TestBinaryEntryInvalidData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "empty",
			data: nil,
		},
		{
			name: "short key length",
			data: []byte{0, 0, 0},
		},
		{
			name: "key length exceeds data",
			data: []byte{0, 0, 0, 100},
		},
		{
			name: "missing records count",
			data: []byte{
				0, 0, 0, 1,
				'A',
			},
		},
		{
			name: "missing records",
			data: []byte{
				0, 0, 0, 1,
				'A',
				0, 0, 0, 0, 0, 0, 0, 2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var entry BinaryEntry

			if err := entry.UnmarshalBinary(tt.data); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestBinaryEntryMarshalInvalid(t *testing.T) {
	tests := []struct {
		name  string
		entry BinaryEntry
	}{
		{
			name: "incorrect key length",
			entry: BinaryEntry{
				KeyLength: 10,
				Key:       []byte("ABC"),
			},
		},
		{
			name: "incorrect records count",
			entry: BinaryEntry{
				KeyLength:    1,
				Key:          []byte("A"),
				RecordsCount: 2,
				Records:      make([]byte, 8),
			},
		},
		{
			name: "partial record",
			entry: BinaryEntry{
				KeyLength:    1,
				Key:          []byte("A"),
				RecordsCount: 1,
				Records:      make([]byte, 9),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.entry.MarshalBinary(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestBinaryEntryEmptyRoundTrip(t *testing.T) {
	original := BinaryEntry{}

	data, err := original.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	if len(data) != 12 {
		t.Fatalf("expected 12 bytes, got %d", len(data))
	}

	var restored BinaryEntry
	if err := restored.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}

	if restored.KeyLength != 0 ||
		restored.RecordsCount != 0 ||
		len(restored.Key) != 0 ||
		len(restored.Records) != 0 {
		t.Fatalf("unexpected entry: %+v", restored)
	}
}
