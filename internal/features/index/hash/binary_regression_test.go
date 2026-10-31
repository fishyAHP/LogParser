package hash

import (
	"bytes"
	"encoding/binary"
	"math"
	"reflect"
	"testing"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/binary"
)

// Формируем snapshot вручную, чтобы проверять
// ситуации, которые обычный MarshalBinary не создаёт.
func makeTestSnapshot(t *testing.T, entries ...indexbin.BinaryEntry) []byte {
	t.Helper()

	var buf bytes.Buffer

	if err := binary.Write(&buf, binary.BigEndian, uint64(0)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(&buf, binary.BigEndian, uint32(len(entries))); err != nil {
		t.Fatal(err)
	}

	for _, entry := range entries {
		data, err := entry.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		buf.Write(data)
	}

	return buf.Bytes()
}

func makeTestEntry(t *testing.T, key string, ids ...uint64) indexbin.BinaryEntry {
	t.Helper()

	posting := testPosting(t, ids...)
	data, err := posting.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	return indexbin.BinaryEntry{
		KeyLength:    uint32(len(key)),
		Key:          []byte(key),
		RecordsCount: uint64(len(ids)),
		Records:      data,
	}
}

func TestIndexBinaryDuplicateKeys(t *testing.T) {
	data := makeTestSnapshot(t,
		makeTestEntry(t, "ERROR", 1, 2),
		makeTestEntry(t, "ERROR", 3, 4),
	)

	index := New[domain.StringValue]()

	if err := index.UnmarshalBinary(data); err == nil {
		t.Fatal("expected duplicate key error")
	}
}

func TestIndexBinaryInvalidEntriesCount(t *testing.T) {
	valid := makeTestSnapshot(t,
		makeTestEntry(t, "ERROR", 1, 2),
	)

	tests := []struct {
		name  string
		count uint32
	}{
		{"too few", 0},
		{"too many", 2},
		{"max uint32", math.MaxUint32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := bytes.Clone(valid)
			binary.BigEndian.PutUint32(data[8:12], tt.count)

			index := New[domain.StringValue]()

			if err := index.UnmarshalBinary(data); err == nil {
				t.Fatalf("expected error for entriesCount=%d", tt.count)
			}
		})
	}
}

func TestIndexBinaryTrailingData(t *testing.T) {
	valid := makeTestSnapshot(t,
		makeTestEntry(t, "ERROR", 1, 2),
	)

	data := append(bytes.Clone(valid), 0xFF, 0xAA)

	index := New[domain.StringValue]()

	if err := index.UnmarshalBinary(data); err == nil {
		t.Fatal("expected trailing data error")
	}
}

func TestIndexBinaryCorruptedPosting(t *testing.T) {
	entry := makeTestEntry(t, "ERROR", 1, 2, 3)

	// Нарушаем строгий порядок RecordID:
	// [1, 2, 3] -> [1, 1, 3].
	binary.BigEndian.PutUint64(entry.Records[8:16], 1)

	data := makeTestSnapshot(t, entry)
	index := New[domain.StringValue]()

	if err := index.UnmarshalBinary(data); err == nil {
		t.Fatal("expected invalid posting order error")
	}
}

func TestIndexBinaryInvalidKeyEncoding(t *testing.T) {
	t.Run("invalid bool", func(t *testing.T) {
		entry := makeTestEntry(t, "X", 1)
		entry.Key = []byte{2}
		entry.KeyLength = 1

		data := makeTestSnapshot(t, entry)
		index := New[domain.BoolValue]()

		if err := index.UnmarshalBinary(data); err == nil {
			t.Fatal("expected invalid bool error")
		}
	})

	t.Run("invalid int length", func(t *testing.T) {
		data := makeTestSnapshot(t,
			makeTestEntry(t, "ABC", 1),
		)

		index := New[domain.IntValue]()

		if err := index.UnmarshalBinary(data); err == nil {
			t.Fatal("expected invalid int length error")
		}
	})

	t.Run("invalid float length", func(t *testing.T) {
		data := makeTestSnapshot(t,
			makeTestEntry(t, "ABC", 1),
		)

		index := New[domain.FloatValue]()

		if err := index.UnmarshalBinary(data); err == nil {
			t.Fatal("expected invalid float length error")
		}
	})
}

func TestIndexBinaryLargePosting(t *testing.T) {
	const count = 100_000

	ids := make([]uint64, count)
	for j := range ids {
		ids[j] = uint64(j + 1)
	}

	original := New[domain.StringValue]()
	original.idx["LARGE"] = testPosting(t, ids...)
	original.count = count

	data, err := original.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	restored := New[domain.StringValue]()
	if err := restored.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}

	if restored.count != original.count {
		t.Fatalf(
			"count mismatch: want %d, got %d",
			original.count, restored.count,
		)
	}

	if !reflect.DeepEqual(original.idx, restored.idx) {
		t.Fatal("large posting mismatch")
	}
}

func TestIndexBinaryPreservesStateOnCorruption(t *testing.T) {
	valid := makeTestSnapshot(t,
		makeTestEntry(t, "NEW", 1, 2),
	)

	invalid := bytes.Clone(valid)
	binary.BigEndian.PutUint32(invalid[8:12], 2)

	index := New[domain.StringValue]()
	index.idx["EXISTING"] = testPosting(t, 10, 20)
	index.count = 2

	expected := New[domain.StringValue]()
	expected.idx["EXISTING"] = testPosting(t, 10, 20)
	expected.count = 2

	if err := index.UnmarshalBinary(invalid); err == nil {
		t.Fatal("expected error")
	}

	if index.count != expected.count ||
		!reflect.DeepEqual(index.idx, expected.idx) {
		t.Fatal("existing index was modified after error")
	}
}
