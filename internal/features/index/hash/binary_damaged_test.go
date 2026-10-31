package hash

import (
	"bytes"
	"encoding/binary"
	"testing"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/binary"
)

func TestHashIndexBinaryInvalidRecordsCount(t *testing.T) {
	idx := New[domain.IntValue]()

	if err := idx.Add(domain.IntValue(10), 1); err != nil {
		t.Fatal(err)
	}
	if err := idx.Add(domain.IntValue(10), 2); err != nil {
		t.Fatal(err)
	}
	if err := idx.Add(domain.IntValue(20), 3); err != nil {
		t.Fatal(err)
	}

	original, err := idx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		count uint64
	}{
		{"less than actual", 2},
		{"greater than actual", 4},
		{"maximum value", ^uint64(0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := bytes.Clone(original)

			binary.BigEndian.PutUint64(
				data[:indexbin.RecordsCountSize],
				tt.count,
			)

			restored := New[domain.IntValue]()

			if err := restored.UnmarshalBinary(data); err == nil {
				t.Fatal("expected records count mismatch")
			}
		})
	}
}
