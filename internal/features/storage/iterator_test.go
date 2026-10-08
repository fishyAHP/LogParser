package storage

import (
	"bytes"
	"testing"
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
