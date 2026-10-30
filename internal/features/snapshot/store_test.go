package snapshot

import (
	"bytes"
	"errors"
	"iter"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index"
	"fishyAHP/LogParser.git/internal/features/snapshot/manifest"
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
		m.IndexCount() != 2 {
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

type testRebuilder struct {
	snapshots []index.Snapshot
}

func (r *testRebuilder) FromSnapshots(
	snapshots iter.Seq2[index.Snapshot, error],
) error {
	for snapshot, err := range snapshots {
		if err != nil {
			return err
		}

		r.snapshots = append(r.snapshots, snapshot)
	}

	return nil
}

func TestStoreSaveLoad(t *testing.T) {
	dir := t.TempDir()

	store := &Store{path: dir}

	expected := []index.Snapshot{
		{
			Name: "hash",
			Data: []byte("hash index data"),
		},
		{
			Name: "range",
			Data: []byte("range index data"),
		},
		{
			Name: "text",
			Data: []byte("text index data"),
		},
	}

	snapshots := func(yield func(index.Snapshot, error) bool) {
		for _, snapshot := range expected {
			if !yield(snapshot, nil) {
				return
			}
		}
	}

	const lastRecordID = 100

	if err := store.Save(snapshots, lastRecordID); err != nil {
		t.Fatalf("save snapshots: %v", err)
	}

	// Имитируем перезапуск приложения.
	restoredStore := &Store{path: dir}

	rebuilder := &testRebuilder{}

	if err := restoredStore.Load(rebuilder); err != nil {
		t.Fatalf("load snapshots: %v", err)
	}

	if !reflect.DeepEqual(rebuilder.snapshots, expected) {
		t.Errorf(
			"restored snapshots mismatch:\ngot:  %#v\nwant: %#v",
			rebuilder.snapshots,
			expected,
		)
	}

	if restoredStore.lastIndexedRecordID != lastRecordID {
		t.Errorf(
			"last indexed record ID: got %d, want %d",
			restoredStore.lastIndexedRecordID,
			lastRecordID,
		)
	}
}

func TestStoreLoadCorruptedIndex(t *testing.T) {
	dir := t.TempDir()
	store := &Store{path: dir}

	snapshots := func(yield func(index.Snapshot, error) bool) {
		yield(index.Snapshot{
			Name: "hash",
			Data: []byte("original index data"),
		}, nil)
	}

	if err := store.Save(snapshots, 100); err != nil {
		t.Fatal(err)
	}

	snapshotDir := filepath.Join(dir, "snapshot-0001")

	r := &reader{dir: snapshotDir}
	if err := r.readManifest(); err != nil {
		t.Fatal(err)
	}

	for info := range r.m.Indexes() {
		path := filepath.Join(snapshotDir, info.Filename)

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}

		// Меняем один байт, сохраняя размер файла.
		data[0] ^= 0xFF

		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}

		break
	}

	rebuilder := &testRebuilder{}

	err := store.Load(rebuilder)
	if err == nil {
		t.Fatal("expected checksum error, got nil")
	}

	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestStoreLoadLatestGeneration(t *testing.T) {
	dir := t.TempDir()
	store := &Store{path: dir}

	makeSnapshots := func(data string) iter.Seq2[index.Snapshot, error] {
		return func(yield func(index.Snapshot, error) bool) {
			yield(index.Snapshot{
				Name: "hash",
				Data: []byte(data),
			}, nil)
		}
	}

	if err := store.Save(makeSnapshots("generation 1"), 10); err != nil {
		t.Fatal(err)
	}

	if err := store.Save(makeSnapshots("generation 2"), 20); err != nil {
		t.Fatal(err)
	}

	restored := &Store{path: dir}
	rebuilder := &testRebuilder{}

	if err := restored.Load(rebuilder); err != nil {
		t.Fatal(err)
	}

	if len(rebuilder.snapshots) != 1 {
		t.Fatalf(
			"expected 1 snapshot, got %d",
			len(rebuilder.snapshots),
		)
	}

	if got := string(rebuilder.snapshots[0].Data); got != "generation 2" {
		t.Errorf("got %q, want %q", got, "generation 2")
	}

	if restored.lastIndexedRecordID != 20 {
		t.Errorf(
			"last record ID: got %d, want 20",
			restored.lastIndexedRecordID,
		)
	}
}

func TestStoreLoadMissingIndex(t *testing.T) {
	dir := t.TempDir()
	store := &Store{path: dir}

	snapshots := func(yield func(index.Snapshot, error) bool) {
		yield(index.Snapshot{
			Name: "hash",
			Data: []byte("index data"),
		}, nil)
	}

	if err := store.Save(snapshots, 10); err != nil {
		t.Fatal(err)
	}

	snapshotDir := filepath.Join(dir, "snapshot-0001")

	r := &reader{dir: snapshotDir}
	if err := r.readManifest(); err != nil {
		t.Fatal(err)
	}

	removed := false

	for info := range r.m.Indexes() {
		path := filepath.Join(snapshotDir, info.Filename)

		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}

		removed = true
		break
	}

	if !removed {
		t.Fatal("manifest contains no indexes")
	}

	rebuilder := &testRebuilder{}

	if err := store.Load(rebuilder); err == nil {
		t.Fatal("expected load error for missing index")
	}
}

func TestStoreLoadDoesNotUpdateCheckpointOnError(t *testing.T) {
	dir := t.TempDir()
	store := &Store{path: dir}

	snapshots := func(yield func(index.Snapshot, error) bool) {
		if !yield(index.Snapshot{
			Name: "hash",
			Data: []byte("hash data"),
		}, nil) {
			return
		}

		yield(index.Snapshot{
			Name: "range",
			Data: []byte("range data"),
		}, nil)
	}

	if err := store.Save(snapshots, 100); err != nil {
		t.Fatal(err)
	}

	snapshotDir := filepath.Join(dir, "snapshot-0001")
	r := &reader{dir: snapshotDir}

	if err := r.readManifest(); err != nil {
		t.Fatal(err)
	}

	i := 0
	for info := range r.m.Indexes() {
		i++
		if i != 2 {
			continue
		}

		path := filepath.Join(snapshotDir, info.Filename)

		if err := os.WriteFile(path, []byte("broken"), 0600); err != nil {
			t.Fatal(err)
		}

		break
	}

	if i < 2 {
		t.Fatal("expected at least two indexes")
	}

	restored := &Store{path: dir}
	rebuilder := &testRebuilder{}

	if err := restored.Load(rebuilder); err == nil {
		t.Fatal("expected load error")
	}

	if restored.lastIndexedRecordID != 0 {
		t.Errorf(
			"checkpoint changed after failed load: %d",
			restored.lastIndexedRecordID,
		)
	}
}
