package ranged

import (
	"bytes"
	"cmp"
	"encoding/binary"
	"iter"
	"reflect"
	"slices"
	"testing"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/binary"
	"fishyAHP/LogParser.git/internal/features/index/common"
)

// Collect RecordIDs from an iterator.
func collectIDs(seq iter.Seq[domain.RecordID]) []domain.RecordID {
	var result []domain.RecordID

	if seq != nil {
		for id := range seq {
			result = append(result, id)
		}
	}

	return result
}

func checkExact[K common.Key](
	t *testing.T,
	idx *Index[K],
	key K,
	want []domain.RecordID,
) {
	t.Helper()

	seq, err := idx.Exact(key)
	if err != nil {
		t.Fatalf("Exact(%v): %v", key, err)
	}

	got := collectIDs(seq)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf(
			"Exact(%v): got %v, want %v",
			key, got, want,
		)
	}
}

func testRoundTrip[K common.Key](
	t *testing.T,
	compare func(K, K) int,
	keys []K,
	records [][]domain.RecordID,
) {
	t.Helper()

	original := New(compare)

	for n, key := range keys {
		for _, id := range records[n] {
			if err := original.Add(key, id); err != nil {
				t.Fatalf("Add(%v, %d): %v", key, id, err)
			}
		}
	}

	data, err := original.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	restored := New(compare)
	if err := restored.UnmarshalBinary(data); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}

	if restored.tree.elemsCount != original.tree.elemsCount {
		t.Fatalf(
			"elements count: got %d, want %d",
			restored.tree.elemsCount,
			original.tree.elemsCount,
		)
	}

	if restored.tree.nodesCount != original.tree.nodesCount {
		t.Fatalf(
			"nodes count: got %d, want %d",
			restored.tree.nodesCount,
			original.tree.nodesCount,
		)
	}

	for n, key := range keys {
		checkExact(t, restored, key, records[n])
	}

	// A second serialization must preserve the same
	// logical representation of the RBTree.
	second, err := restored.MarshalBinary()
	if err != nil {
		t.Fatalf("second MarshalBinary: %v", err)
	}

	if !bytes.Equal(data, second) {
		t.Fatal("snapshot changed after round-trip")
	}
}

func TestRangeIndexBinaryRoundTrip(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		testRoundTrip(
			t,
			cmp.Compare[domain.IntValue],
			[]domain.IntValue{-100, 0, 42},
			[][]domain.RecordID{
				{1, 3, 9},
				{2},
				{4, 5},
			},
		)
	})

	t.Run("float", func(t *testing.T) {
		testRoundTrip(
			t,
			cmp.Compare[domain.FloatValue],
			[]domain.FloatValue{-3.5, 0.25, 99.75},
			[][]domain.RecordID{
				{1, 2},
				{3},
				{4, 8},
			},
		)
	})

	t.Run("string", func(t *testing.T) {
		testRoundTrip(
			t,
			cmp.Compare[domain.StringValue],
			[]domain.StringValue{"", "alpha", "beta"},
			[][]domain.RecordID{
				{1},
				{2, 3},
				{4},
			},
		)
	})

	t.Run("bool", func(t *testing.T) {
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

		testRoundTrip(
			t,
			compare,
			[]domain.BoolValue{false, true},
			[][]domain.RecordID{
				{1, 2},
				{3, 4},
			},
		)
	})

	t.Run("time", func(t *testing.T) {
		compare := func(a, b domain.TimeValue) int {
			return time.Time(a).Compare(time.Time(b))
		}

		first := domain.TimeValue(
			time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		)
		second := domain.TimeValue(
			time.Date(2025, 6, 15, 12, 30, 0, 0, time.UTC),
		)

		testRoundTrip(
			t,
			compare,
			[]domain.TimeValue{first, second},
			[][]domain.RecordID{
				{1, 5},
				{2, 3},
			},
		)
	})
}

func TestRangeIndexBinaryEmpty(t *testing.T) {
	idx := New(cmp.Compare[domain.IntValue])

	data, err := idx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	if len(data) != indexbin.HeaderSize {
		t.Fatalf(
			"empty snapshot size: got %d, want %d",
			len(data), indexbin.HeaderSize,
		)
	}

	restored := New(cmp.Compare[domain.IntValue])
	if err := restored.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}

	if restored.tree.root != nil ||
		restored.tree.elemsCount != 0 ||
		restored.tree.nodesCount != 0 {
		t.Fatal("restored empty tree contains data")
	}
}

func TestRangeIndexBinaryRangeAndExtremum(t *testing.T) {
	idx := New(cmp.Compare[domain.IntValue])

	for _, row := range []struct {
		key domain.IntValue
		id  domain.RecordID
	}{
		{10, 1},
		{20, 2},
		{20, 3},
		{30, 4},
		{40, 5},
	} {
		if err := idx.Add(row.key, row.id); err != nil {
			t.Fatal(err)
		}
	}

	data, err := idx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	restored := New(cmp.Compare[domain.IntValue])
	if err := restored.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}

	from := &domain.Bound{
		Value:     domain.IntValue(20),
		Inclusive: true,
	}
	to := &domain.Bound{
		Value:     domain.IntValue(30),
		Inclusive: true,
	}

	seq, err := restored.Range(from, to)
	if err != nil {
		t.Fatal(err)
	}

	got := collectIDs(seq)
	slices.Sort(got)

	if !slices.Equal(got, []domain.RecordID{2, 3, 4}) {
		t.Fatalf("Range: got %v, want [2 3 4]", got)
	}

	source := func(yield func(domain.RecordID) bool) {
		for _, id := range []domain.RecordID{1, 2, 3, 4, 5} {
			if !yield(id) {
				return
			}
		}
	}

	minValue, err := restored.Min(source)
	if err != nil {
		t.Fatal(err)
	}
	if minValue != domain.IntValue(10) {
		t.Fatalf("Min: got %v, want 10", minValue)
	}

	maxValue, err := restored.Max(source)
	if err != nil {
		t.Fatal(err)
	}
	if maxValue != domain.IntValue(40) {
		t.Fatalf("Max: got %v, want 40", maxValue)
	}
}

// A valid snapshot used for corruption tests.
func makeTestSnapshot(t *testing.T) []byte {
	t.Helper()

	idx := New(cmp.Compare[domain.IntValue])

	for _, row := range []struct {
		key domain.IntValue
		id  domain.RecordID
	}{
		{10, 1},
		{10, 2},
		{20, 3},
		{30, 4},
	} {
		if err := idx.Add(row.key, row.id); err != nil {
			t.Fatal(err)
		}
	}

	data, err := idx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestRangeIndexBinaryTruncated(t *testing.T) {
	data := makeTestSnapshot(t)

	for size := 0; size < len(data); size++ {
		t.Run("truncated", func(t *testing.T) {
			restored := New(cmp.Compare[domain.IntValue])

			if err := restored.UnmarshalBinary(
				data[:size],
			); err == nil {
				t.Fatalf(
					"expected error for truncated length %d",
					size,
				)
			}
		})
	}
}

func TestRangeIndexBinaryPreserveOnError(t *testing.T) {
	idx := New(cmp.Compare[domain.IntValue])

	if err := idx.Add(domain.IntValue(999), 100); err != nil {
		t.Fatal(err)
	}

	before, err := idx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	data := makeTestSnapshot(t)
	data = data[:len(data)-1]

	if err := idx.UnmarshalBinary(data); err == nil {
		t.Fatal("expected error")
	}

	after, err := idx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(before, after) {
		t.Fatal("index modified after failed restore")
	}

	checkExact(
		t, idx,
		domain.IntValue(999),
		[]domain.RecordID{100},
	)
}

func TestRangeIndexBinaryInvalidCounters(t *testing.T) {
	original := makeTestSnapshot(t)

	tests := []struct {
		name   string
		mutate func([]byte)
	}{
		{
			name: "records too small",
			mutate: func(data []byte) {
				binary.BigEndian.PutUint64(data[:8], 3)
			},
		},
		{
			name: "records too large",
			mutate: func(data []byte) {
				binary.BigEndian.PutUint64(data[:8], 5)
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

			idx := New(cmp.Compare[domain.IntValue])

			if err := idx.UnmarshalBinary(data); err == nil {
				t.Fatal("expected invalid counters error")
			}
		})
	}
}

func TestRangeIndexBinaryTrailingData(t *testing.T) {
	data := makeTestSnapshot(t)
	data = append(data, 0xAB, 0xCD)

	idx := New(cmp.Compare[domain.IntValue])

	if err := idx.UnmarshalBinary(data); err == nil {
		t.Fatal("expected trailing data error")
	}
}

func TestRangeIndexBinaryDuplicateKeys(t *testing.T) {
	idx := New(cmp.Compare[domain.IntValue])

	if err := idx.Add(domain.IntValue(10), 1); err != nil {
		t.Fatal(err)
	}

	data, err := idx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	// Duplicate the encoded entry and increase
	// the number of entries in the header.
	entry := bytes.Clone(data[indexbin.HeaderSize:])
	data = append(data, entry...)

	binary.BigEndian.PutUint32(
		data[indexbin.RecordsCountSize:indexbin.HeaderSize],
		2,
	)
	binary.BigEndian.PutUint64(
		data[:indexbin.RecordsCountSize],
		2,
	)

	restored := New(cmp.Compare[domain.IntValue])

	if err = restored.UnmarshalBinary(data); err == nil {
		t.Fatal("expected duplicate key error")
	}
}
