package storage

import (
	"bytes"
	"testing"

	"fishyAHP/LogParser.git/internal/core/domain"
)

func TestSegment_WriteRead(t *testing.T) {
	dir := t.TempDir()

	seg, err := newSegment(dir, 1)
	if err != nil {
		t.Fatalf("newSegment: %v", err)
	}
	defer seg.close()

	want := []byte(`{"level":"INFO","message":"hello"}`)

	record, err := seg.write(want, 1)
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := seg.read(record)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if !bytes.Equal(got, want) {
		t.Fatalf(
			"unexpected data:\nwant: %q\ngot:  %q",
			want,
			got,
		)
	}

	if record.Pointer.Length != uint32(len(want)) {
		t.Fatalf(
			"unexpected id length: want %d, got %d",
			len(want),
			record.Pointer.Length,
		)
	}
}

func TestSegment_WriteReadMultiple(t *testing.T) {
	dir := t.TempDir()

	seg, err := newSegment(dir, 1)
	if err != nil {
		t.Fatalf("newSegment: %v", err)
	}
	defer seg.close()

	input := [][]byte{
		[]byte(`{"level":"INFO","message":"first"}`),
		[]byte(`{"level":"DEBUG","message":"second message"}`),
		[]byte(`{"level":"ERROR","message":"third"}`),
	}

	records := make([]*domain.RecordData, 0, len(input))

	for i, data := range input {
		record, err := seg.write(data, uint64(i+1))
		if err != nil {
			t.Fatalf("write: %v", err)
		}

		records = append(records, record)
	}

	for i, record := range records {
		got, err := seg.read(record)
		if err != nil {
			t.Fatalf("read id %d: %v", i, err)
		}

		if !bytes.Equal(got, input[i]) {
			t.Fatalf(
				"id %d mismatch:\nwant: %q\ngot:  %q",
				i,
				input[i],
				got,
			)
		}
	}
}
