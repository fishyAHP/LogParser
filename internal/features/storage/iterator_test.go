package storage

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"fishyAHP/LogParser.git/internal/core/domain"
)

func TestIterator_Next(t *testing.T) {
	dir := t.TempDir()

	s, err := New(dir)
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}
	defer s.Close()

	logs := [][]byte{
		[]byte(`{"level":"INFO","message":"first"}`),
		[]byte(`{"level":"ERROR","message":"second"}`),
		[]byte(`{"level":"DEBUG","message":"third"}`),
	}

	for _, log := range logs {
		if _, err := s.Write(log); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	it := s.Iterator()

	var got [][]byte

	for it.Next() {
		got = append(got, bytes.Clone(it.Data()))
	}

	if err := it.Err(); err != nil {
		t.Fatalf("iterator: %v", err)
	}

	if len(got) != len(logs) {
		t.Fatalf("expected %d recordsCount, got %d", len(logs), len(got))
	}

	for j := range logs {
		if !bytes.Equal(got[j], logs[j]) {
			t.Errorf(
				"id %d: expected %q, got %q",
				j,
				logs[j],
				got[j],
			)
		}
	}
}

func TestIterator_AfterDone(t *testing.T) {
	dir := t.TempDir()

	s, err := New(dir)
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}
	defer s.Close()

	if _, err := s.Write([]byte("hello")); err != nil {
		t.Fatalf("write: %v", err)
	}

	it := s.Iterator()

	if !it.Next() {
		t.Fatalf("expected first id: %v", it.Err())
	}

	if it.Next() {
		t.Fatal("expected iterator to finish")
	}

	if err := it.Err(); err != nil {
		t.Fatalf("unexpected iterator error: %v", err)
	}

	if it.Next() {
		t.Fatal("iterator restarted after completion")
	}

	if err := it.Err(); err != nil {
		t.Fatalf("unexpected iterator error after completion: %v", err)
	}
}

func TestIterator_EmptyStorage(t *testing.T) {
	dir := t.TempDir()

	s, err := New(dir)
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}
	defer s.Close()

	it := s.Iterator()

	if it.Next() {
		t.Fatal("expected empty iterator")
	}

	if err := it.Err(); err != nil {
		t.Fatalf("empty storage should not produce error: %v", err)
	}
}

func TestStorageIteratorFromID(t *testing.T) {
	dir := t.TempDir()

	store, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	for i := 1; i <= 10; i++ {
		data := []byte(fmt.Sprintf("record-%d", i))

		id, err := store.Write(data)
		if err != nil {
			t.Fatal(err)
		}

		if id != domain.RecordID(i) {
			t.Fatalf("got ID %d, want %d", id, i)
		}
	}

	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	// Проверяем именно восстановленные RecordPointer.
	store, err = New(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	tests := []struct {
		name     string
		startID  domain.RecordID
		expected []domain.RecordID
	}{
		{
			name:     "zero record ID",
			startID:  0,
			expected: []domain.RecordID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			name:     "first record",
			startID:  1,
			expected: []domain.RecordID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			name:     "middle record",
			startID:  5,
			expected: []domain.RecordID{5, 6, 7, 8, 9, 10},
		},
		{
			name:     "last record",
			startID:  10,
			expected: []domain.RecordID{10},
		},
		{
			name:     "after last record",
			startID:  11,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iterator, err := store.IteratorFromID(tt.startID)
			if err != nil {
				t.Fatal(err)
			}
			defer iterator.Close()

			var got []domain.RecordID

			for iterator.Next() {
				id := iterator.RecordID()
				got = append(got, id)

				wantData := fmt.Sprintf("record-%d", id)
				if string(iterator.Data()) != wantData {
					t.Fatalf(
						"record %d: got %q, want %q",
						id,
						iterator.Data(),
						wantData,
					)
				}
			}

			if err := iterator.Err(); err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf(
					"start ID %d: got %v, want %v",
					tt.startID,
					got,
					tt.expected,
				)
			}
		})
	}
}
