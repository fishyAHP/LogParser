package snapshot

import (
	"bytes"
	"errors"
	"iter"
	"os"
	"path/filepath"
	"testing"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index"
	"fishyAHP/LogParser.git/internal/features/storage/snapshot/manifest"
)

func snapshotSeq(items ...index.Snapshot) iter.Seq2[index.Snapshot, error] {
	return func(yield func(index.Snapshot, error) bool) {
		for _, item := range items {
			if !yield(item, nil) {
				return
			}
		}
	}
}

func TestStoreSave(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	first := []byte("hash index data")
	second := []byte("range index data")

	err = store.Save(snapshotSeq(
		index.Snapshot{Name: "hash", Data: first},
		index.Snapshot{Name: "range", Data: second},
	), domain.RecordID(100))
	if err != nil {
		t.Fatal(err)
	}

	dir := filepath.Join(store.path, "snapshot-0001")

	for _, tc := range []struct {
		filename string
		want     []byte
	}{
		{"1.bin", first},
		{"2.bin", second},
	} {
		got, err := os.ReadFile(filepath.Join(dir, tc.filename))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, tc.want) {
			t.Errorf("file %s: unexpected content", tc.filename)
		}
	}

	data, err := os.ReadFile(filepath.Join(dir, "manifest.bin"))
	if err != nil {
		t.Fatal(err)
	}

	var m manifest.Manifest
	if err := m.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}

	if m.Generation != 1 ||
		m.LastIndexedRecordID != domain.RecordID(100) ||
		len(m.indexes) != 2 {
		t.Fatalf("unexpected manifest: %+v", m)
	}

	if store.lastIndexedRecordID != domain.RecordID(100) ||
		!store.lastUpdate.Equal(m.CreatedAt) {
		t.Fatal("store state does not match manifest")
	}
}

func TestStoreSaveMultipleGenerations(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	for _, id := range []domain.RecordID{10, 20} {
		err := store.Save(snapshotSeq(
			index.Snapshot{
				Name: "hash",
				Data: []byte("snapshot data"),
			},
		), id)
		if err != nil {
			t.Fatal(err)
		}
	}

	for _, name := range []string{
		"snapshot-0001",
		"snapshot-0002",
	} {
		if info, err := os.Stat(filepath.Join(store.path, name)); err != nil {
			t.Fatal(err)
		} else if !info.IsDir() {
			t.Fatalf("%s is not a directory", name)
		}
	}

	if store.lastIndexedRecordID != 20 {
		t.Fatalf("last record ID = %d, want 20", store.lastIndexedRecordID)
	}
}

func TestStoreSaveSnapshotError(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	expectedErr := errors.New("snapshot generation failed")

	broken := func(yield func(index.Snapshot, error) bool) {
		if !yield(index.Snapshot{
			Name: "hash",
			Data: []byte("partial data"),
		}, nil) {
			return
		}
		yield(index.Snapshot{}, expectedErr)
	}

	err = store.Save(broken, 100)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("got error %v, want %v", err, expectedErr)
	}

	entries, err := os.ReadDir(store.path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty snapshot directory, got %d entries", len(entries))
	}

	if store.lastIndexedRecordID != 0 || !store.lastUpdate.IsZero() {
		t.Fatal("store state changed after failed save")
	}
}
