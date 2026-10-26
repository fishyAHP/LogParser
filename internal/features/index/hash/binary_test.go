package hash

import (
	"encoding/binary"
	"fmt"
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
