package text

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"
	"testing"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/binary"
)

func collectIDs(seq func(func(domain.RecordID) bool)) []domain.RecordID {
	var result []domain.RecordID
	for id := range seq {
		result = append(result, id)
	}
	return result
}

func assertSearch(
	t *testing.T,
	idx *Index,
	query string,
	want []domain.RecordID,
) {
	t.Helper()

	seq, err := idx.Search(domain.StringValue(query))
	if err != nil {
		t.Fatalf("Search(%q): %v", query, err)
	}

	got := collectIDs(seq)
	if !slices.Equal(got, want) {
		t.Fatalf(
			"Search(%q): got %v, want %v",
			query, got, want,
		)
	}
}

func makeSnapshot(t *testing.T) []byte {
	t.Helper()

	idx := New()

	for _, row := range []struct {
		value string
		id    domain.RecordID
	}{
		{"alpha beta", 1},
		{"alpha gamma", 2},
		{"beta gamma", 3},
	} {
		if err := idx.Add(domain.StringValue(row.value), row.id); err != nil {
			t.Fatal(err)
		}
	}

	data, err := idx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestTextIndexBinaryRoundTrip(t *testing.T) {
	data := makeSnapshot(t)

	restored := New()
	if err := restored.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}

	assertSearch(t, restored, "alpha", []domain.RecordID{1, 2})
	assertSearch(t, restored, "beta", []domain.RecordID{1, 3})
	assertSearch(t, restored, "gamma", []domain.RecordID{2, 3})
	assertSearch(t, restored, "alpha beta", []domain.RecordID{1})

	if restored.Len() != 6 {
		t.Fatalf("count: got %d, want 6", restored.Len())
	}

	again, err := restored.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(data, again) {
		t.Fatal("round-trip changed snapshot bytes")
	}
}

func TestTextIndexBinaryEmpty(t *testing.T) {
	idx := New()

	data, err := idx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	if len(data) != indexbin.HeaderSize {
		t.Fatalf(
			"snapshot size: got %d, want %d",
			len(data), indexbin.HeaderSize,
		)
	}

	restored := New()
	if err := restored.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}

	if restored.Len() != 0 || len(restored.invert) != 0 {
		t.Fatal("restored index is not empty")
	}
}

func TestTextIndexBinaryDeterministic(t *testing.T) {
	first := New()
	second := New()

	values := []string{
		"alpha beta",
		"gamma delta",
		"epsilon zeta",
	}

	for j, value := range values {
		if err := first.Add(
			domain.StringValue(value),
			domain.RecordID(j+1),
		); err != nil {
			t.Fatal(err)
		}
	}

	for j := len(values) - 1; j >= 0; j-- {
		if err := second.Add(
			domain.StringValue(values[j]),
			domain.RecordID(j+1),
		); err != nil {
			t.Fatal(err)
		}
	}

	a, err := first.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	b, err := second.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(a, b) {
		t.Fatal("same content produced different snapshots")
	}
}

func TestTextIndexBinaryTruncated(t *testing.T) {
	data := makeSnapshot(t)

	for size := 0; size < len(data); size++ {
		t.Run(fmt.Sprintf("length_%d", size), func(t *testing.T) {
			idx := New()

			if err := idx.UnmarshalBinary(data[:size]); err == nil {
				t.Fatalf("accepted truncated snapshot: %d bytes", size)
			}
		})
	}
}

func TestTextIndexBinaryInvalidCounters(t *testing.T) {
	original := makeSnapshot(t)

	tests := []struct {
		name   string
		mutate func([]byte)
	}{
		{
			name: "records too small",
			mutate: func(data []byte) {
				binary.BigEndian.PutUint64(data[:8], 5)
			},
		},
		{
			name: "records too large",
			mutate: func(data []byte) {
				binary.BigEndian.PutUint64(data[:8], 7)
			},
		},
		{
			name: "entries too small",
			mutate: func(data []byte) {
				binary.BigEndian.PutUint32(data[8:12], 2)
			},
		},
		{
			name: "entries too large",
			mutate: func(data []byte) {
				binary.BigEndian.PutUint32(data[8:12], 4)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := bytes.Clone(original)
			tt.mutate(data)

			idx := New()
			if err := idx.UnmarshalBinary(data); err == nil {
				t.Fatal("expected invalid counters error")
			}
		})
	}
}

func TestTextIndexBinaryDuplicateTokens(t *testing.T) {
	idx := New()
	if err := idx.Add(domain.StringValue("alpha"), 1); err != nil {
		t.Fatal(err)
	}

	data, err := idx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	// Repeat the same serialized entry.
	entry := bytes.Clone(data[indexbin.HeaderSize:])
	data = append(data, entry...)

	// Header now describes two entries and two records.
	binary.BigEndian.PutUint64(data[:8], 2)
	binary.BigEndian.PutUint32(data[8:12], 2)

	restored := New()
	if err := restored.UnmarshalBinary(data); err == nil {
		t.Fatal("duplicate token was accepted")
	}
}

func TestTextIndexBinaryTrailingData(t *testing.T) {
	data := append(makeSnapshot(t), 0xAB)

	idx := New()
	if err := idx.UnmarshalBinary(data); err == nil {
		t.Fatal("trailing byte was accepted")
	}
}

func TestTextIndexBinaryPreserveOnError(t *testing.T) {
	idx := New()
	if err := idx.Add(domain.StringValue("original"), 100); err != nil {
		t.Fatal(err)
	}

	before, err := idx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	corrupted := makeSnapshot(t)
	corrupted = corrupted[:len(corrupted)-1]

	if err := idx.UnmarshalBinary(corrupted); err == nil {
		t.Fatal("expected error")
	}

	after, err := idx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(before, after) {
		t.Fatal("index changed after failed unmarshal")
	}

	assertSearch(t, idx, "original", []domain.RecordID{100})
}

func TestTextIndexBinaryRejectZeroRecordID(t *testing.T) {
	idx := New()
	if err := idx.Add(domain.StringValue("alpha"), 1); err != nil {
		t.Fatal(err)
	}

	data, err := idx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	// Entry = KeyLength(4) + key + RecordsCount(8) + IDs.
	keyLen := binary.BigEndian.Uint32(
		data[indexbin.HeaderSize : indexbin.HeaderSize+4],
	)

	recordOffset := indexbin.HeaderSize +
		indexbin.KeyLengthSize +
		int(keyLen) +
		indexbin.RecordsCountSize

	binary.BigEndian.PutUint64(
		data[recordOffset:recordOffset+8], 0,
	)

	restored := New()
	if err := restored.UnmarshalBinary(data); err == nil {
		t.Fatal("RecordID(0) was accepted")
	}
}

func TestTextIndexBinaryRejectEmptyPosting(t *testing.T) {
	idx := New()
	if err := idx.Add(domain.StringValue("alpha"), 1); err != nil {
		t.Fatal(err)
	}

	data, err := idx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	keyLen := binary.BigEndian.Uint32(
		data[indexbin.HeaderSize : indexbin.HeaderSize+4],
	)

	countOffset := indexbin.HeaderSize +
		indexbin.KeyLengthSize + int(keyLen)

	// Zero record count and remove serialized RecordID.
	binary.BigEndian.PutUint64(
		data[countOffset:countOffset+8], 0,
	)
	binary.BigEndian.PutUint64(data[:8], 0)
	data = data[:countOffset+indexbin.RecordsCountSize]

	restored := New()
	if err := restored.UnmarshalBinary(data); err == nil {
		t.Fatal("empty posting was accepted")
	}
}
