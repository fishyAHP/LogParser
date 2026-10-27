package ranged

import (
	"bytes"
	"cmp"
	"fmt"
	"testing"

	"fishyAHP/LogParser.git/internal/core/domain"
)

func FuzzRangeIndexUnmarshalBinary(f *testing.F) {
	// Valid snapshot.
	source := New(cmp.Compare[domain.IntValue])

	for _, row := range []struct {
		key domain.IntValue
		id  domain.RecordID
	}{
		{10, 1},
		{10, 2},
		{20, 3},
		{30, 4},
	} {
		if err := source.Add(row.key, row.id); err != nil {
			f.Fatal(err)
		}
	}

	valid, err := source.MarshalBinary()
	if err != nil {
		f.Fatal(err)
	}

	// Seed corpus.
	f.Add(valid)
	f.Add(valid[:len(valid)-1])
	f.Add([]byte{})
	f.Add([]byte{0xFF})
	f.Add(bytes.Repeat([]byte{0xFF}, 128))
	f.Add(make([]byte, 12))

	f.Fuzz(func(t *testing.T, data []byte) {
		idx := New(cmp.Compare[domain.IntValue])

		// An existing index must survive failed restoration.
		if err := idx.Add(domain.IntValue(999), 100); err != nil {
			t.Fatal(err)
		}

		before, err := idx.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}

		err = idx.UnmarshalBinary(data)

		if err != nil {
			after, marshalErr := idx.MarshalBinary()
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}

			if !bytes.Equal(before, after) {
				t.Fatal("index modified after failed unmarshal")
			}
			return
		}

		// Every accepted snapshot must be serializable.
		restored, err := idx.MarshalBinary()
		if err != nil {
			t.Fatalf("marshal restored index: %v", err)
		}

		// Its canonical representation must be readable.
		again := New(cmp.Compare[domain.IntValue])

		if err := again.UnmarshalBinary(restored); err != nil {
			t.Fatalf("restored snapshot is unreadable: %v", err)
		}

		if again.tree.elemsCount != idx.tree.elemsCount ||
			again.tree.nodesCount != idx.tree.nodesCount {
			t.Fatal("round-trip counters mismatch")
		}
	})
}

func makeBenchmarkIndex(
	b *testing.B,
	totalRecords int,
	uniqueKeys int,
) *Index[domain.IntValue] {
	b.Helper()

	idx := New(cmp.Compare[domain.IntValue])

	for j := range totalRecords {
		key := domain.IntValue(j % uniqueKeys)
		id := domain.RecordID(j + 1)

		if err := idx.Add(key, id); err != nil {
			b.Fatalf("prepare index: %v", err)
		}
	}

	return idx
}

func BenchmarkRangeIndexMarshalBinary(b *testing.B) {
	scenarios := []struct {
		name    string
		records int
		keys    int
	}{
		{"100k_records_100k_keys", 100_000, 100_000},
		{"100k_records_10k_keys", 100_000, 10_000},
		{"100k_records_100_keys", 100_000, 100},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			idx := makeBenchmarkIndex(
				b,
				scenario.records,
				scenario.keys,
			)

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				data, err := idx.MarshalBinary()
				if err != nil {
					b.Fatal(err)
				}

				b.SetBytes(int64(len(data)))
			}
		})
	}
}

func BenchmarkRangeIndexUnmarshalBinary(b *testing.B) {
	scenarios := []struct {
		name    string
		records int
		keys    int
	}{
		{"100k_records_100k_keys", 100_000, 100_000},
		{"100k_records_10k_keys", 100_000, 10_000},
		{"100k_records_100_keys", 100_000, 100},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			source := makeBenchmarkIndex(
				b,
				scenario.records,
				scenario.keys,
			)

			data, err := source.MarshalBinary()
			if err != nil {
				b.Fatal(err)
			}

			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				restored := New(
					cmp.Compare[domain.IntValue],
				)

				if err := restored.UnmarshalBinary(data); err != nil {
					b.Fatal(err)
				}

				if restored.tree.elemsCount !=
					uint64(scenario.records) {
					b.Fatal(fmt.Errorf(
						"invalid records count: %d",
						restored.tree.elemsCount,
					))
				}
			}
		})
	}
}
