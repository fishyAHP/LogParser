package hash

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/common"
	"fishyAHP/LogParser.git/internal/features/index/structs"
)

func testPosting(
	t *testing.T,
	ids ...uint64,
) *structs.PostingList {
	t.Helper()

	data := make([]byte, len(ids)*8)

	for j, id := range ids {
		binary.BigEndian.PutUint64(
			data[j*8:(j+1)*8],
			id,
		)
	}

	posting := structs.NewPostingLists(0)

	if err := posting.UnmarshalBinary(data); err != nil {
		t.Fatalf("create posting: %v", err)
	}

	return posting
}

func testIndexRoundTrip[K common.Key](
	t *testing.T,
	keys []K,
) {
	t.Helper()

	original := New[K]()

	original.idx[keys[0]] = testPosting(t, 1, 3, 5)
	original.idx[keys[1]] = testPosting(t, 2, 4, 6, 8)
	original.count = 7

	data, err := original.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal index: %v", err)
	}

	restored := New[K]()

	if err = restored.UnmarshalBinary(data); err != nil {
		t.Fatalf("unmarshal index: %v", err)
	}

	if original.count != restored.count {
		t.Errorf(
			"count mismatch: want %d, got %d",
			original.count,
			restored.count,
		)
	}

	if !reflect.DeepEqual(original.idx, restored.idx) {
		t.Errorf(
			"index mismatch:\nwant: %#v\ngot: %#v",
			original.idx,
			restored.idx,
		)
	}
}

func TestIndexBinaryRoundTrip(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		testIndexRoundTrip(t, []domain.StringValue{
			"ERROR",
			"Привет, мир!",
		})
	})

	t.Run("int", func(t *testing.T) {
		testIndexRoundTrip(t, []domain.IntValue{
			-100,
			9223372036854775807,
		})
	})

	t.Run("float", func(t *testing.T) {
		testIndexRoundTrip(t, []domain.FloatValue{
			-3.14159,
			123.456,
		})
	})

	t.Run("bool", func(t *testing.T) {
		testIndexRoundTrip(t, []domain.BoolValue{
			true,
			false,
		})
	})

	t.Run("time", func(t *testing.T) {
		testIndexRoundTrip(t, []domain.TimeValue{
			domain.TimeValue(time.Date(
				2025, 1, 1, 0, 0, 0, 0, time.UTC,
			)),
			domain.TimeValue(time.Date(
				2026, 10, 26, 12, 30, 0, 0, time.UTC,
			)),
		})
	})
}

func TestIndexBinaryEmpty(t *testing.T) {
	original := New[domain.StringValue]()

	data, err := original.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	restored := New[domain.StringValue]()

	if err := restored.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}

	if len(restored.idx) != 0 {
		t.Errorf(
			"expected empty index, got %d keys",
			len(restored.idx),
		)
	}

	if restored.count != 0 {
		t.Errorf("expected zero count, got %d", restored.count)
	}
}

func TestIndexUnmarshalTruncated(t *testing.T) {
	original := New[domain.StringValue]()

	original.idx["ERROR"] = testPosting(t, 1, 2, 3)
	original.idx["WARN"] = testPosting(t, 4, 5)
	original.count = 5

	data, err := original.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	for size := range data {
		t.Run(
			fmt.Sprintf("length_%d", size),
			func(t *testing.T) {
				restored := New[domain.StringValue]()

				err := restored.UnmarshalBinary(data[:size])
				if err == nil {
					t.Fatalf(
						"expected error for truncated data of length %d",
						size,
					)
				}
			},
		)
	}
}

func TestIndexUnmarshalPreservesStateOnError(t *testing.T) {
	index := New[domain.StringValue]()

	index.idx["EXISTING"] = testPosting(t, 10, 20, 30)
	index.count = 3

	before := testPosting(t, 10, 20, 30)

	badData := make([]byte, 12)

	binary.BigEndian.PutUint64(badData[:8], 100)
	binary.BigEndian.PutUint32(badData[8:12], 1)

	if err := index.UnmarshalBinary(badData); err == nil {
		t.Fatal("expected unmarshal error")
	}

	if index.count != 3 {
		t.Errorf("index count changed: %d", index.count)
	}

	if len(index.idx) != 1 {
		t.Fatalf("index size changed: %d", len(index.idx))
	}

	posting, ok := index.idx["EXISTING"]
	if !ok {
		t.Fatal("existing key disappeared")
	}

	if !reflect.DeepEqual(posting, before) {
		t.Fatal("existing posting list changed")
	}
}

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

// Формируем snapshot вручную, чтобы проверять
// ситуации, которые обычный MarshalBinary не создаёт.
func makeTestSnapshot(t *testing.T, entries ...BinaryEntry) []byte {
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

func makeTestEntry(t *testing.T, key string, ids ...uint64) BinaryEntry {
	t.Helper()

	posting := testPosting(t, ids...)
	data, err := posting.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	return BinaryEntry{
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
