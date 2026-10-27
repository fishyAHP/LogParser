package snapshot

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := NewStore(filepath.Join(t.TempDir(), "snapshots"))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func sequence(items ...index.Snapshot) iter.Seq2[index.Snapshot, error] {
	return func(yield func(index.Snapshot, error) bool) {
		for _, item := range items {
			if !yield(item, nil) {
				return
			}
		}
	}
}

func readGeneration(t *testing.T, s *Store, generation uint32) (manifest, []index.Snapshot) {
	t.Helper()
	dir := filepath.Join(s.path, fmt.Sprintf("snapshot-%04d", generation))
	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if m.Version != manifestVersion || m.Generation != generation || m.CreatedAt.IsZero() {
		t.Fatalf("invalid manifest: %+v", m)
	}
	result := make([]index.Snapshot, 0, len(m.Indexes))
	for _, entry := range m.Indexes {
		if filepath.Base(entry.Filename) != entry.Filename || !strings.HasPrefix(entry.Filename, "index-") {
			t.Fatalf("unsafe filename %q", entry.Filename)
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Filename))
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(data)
		if len(data) != entry.Size || hex.EncodeToString(digest[:]) != entry.SHA256 {
			t.Fatalf("invalid size or digest for %q", entry.Name)
		}
		result = append(result, index.Snapshot{Name: entry.Name, Data: data})
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(result)+1 {
		t.Fatalf("unexpected files: %v", entries)
	}
	return m, result
}

func assertNoTemporaryDirectories(t *testing.T, s *Store) {
	t.Helper()
	entries, err := os.ReadDir(s.path)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".snapshot-tmp-") {
			t.Fatalf("leaked %s", e.Name())
		}
	}
}

func TestSaveManagerRoundTrip(t *testing.T) {
	scheme := &domain.Scheme{Parameters: []domain.Field{
		{Name: "level", FieldType: domain.StringType, IndexType: domain.HashIndex},
		{Name: "time", FieldType: domain.IntType, IndexType: domain.RangeIndex},
		{Name: "message", FieldType: domain.StringType, IndexType: domain.TextIndex},
	}}
	manager, err := index.New(scheme)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Index(42, domain.LogEntry{Values: []domain.Value{
		domain.StringValue("ERROR"), domain.IntValue(100), domain.StringValue("alpha beta"),
	}}); err != nil {
		t.Fatal(err)
	}
	s := newTestStore(t)
	if err := s.Save(manager.MakeSnapshots(), 42); err != nil {
		t.Fatal(err)
	}
	m, snapshots := readGeneration(t, s, 1)
	if m.LastIndexedRecordID != 42 || s.lastIndexedRecordID != 42 || !s.lastUpdate.Equal(m.CreatedAt) {
		t.Fatalf("checkpoint mismatch: %+v", m)
	}
	restored, err := index.New(scheme)
	if err != nil {
		t.Fatal(err)
	}
	if err := restored.FromSnapshots(sequence(snapshots...)); err != nil {
		t.Fatal(err)
	}
	for _, query := range []struct {
		name  string
		value domain.Value
	}{
		{"level", domain.StringValue("ERROR")}, {"time", domain.IntValue(100)},
	} {
		ids, err := restored.Exact(query.name, query.value)
		if err != nil {
			t.Fatal(err)
		}
		if got := slices.Collect(ids); !slices.Equal(got, []domain.RecordID{42}) {
			t.Fatalf("query %s: %v", query.name, got)
		}
	}
	// Decoding the text snapshot above also verifies compatibility with its codec.
	assertNoTemporaryDirectories(t, s)
}

func TestSaveSafeFilenames(t *testing.T) {
	s := newTestStore(t)
	names := []string{"../outside", "/absolute", `a\b`, "manifest.json", "LEVEL", "level", "привет/мир", "x\x00y"}
	items := make([]index.Snapshot, len(names))
	for i, name := range names {
		items[i] = index.Snapshot{Name: name, Data: []byte(name)}
	}
	if err := s.Save(sequence(items...), 7); err != nil {
		t.Fatal(err)
	}
	_, got := readGeneration(t, s, 1)
	if len(got) != len(items) {
		t.Fatal("missing index")
	}
	for _, item := range got {
		if !slices.Contains(names, item.Name) || !bytes.Equal(item.Data, []byte(item.Name)) {
			t.Fatalf("changed index: %+v", item)
		}
	}
	entries, err := os.ReadDir(filepath.Dir(s.path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("wrote outside snapshot directory: %v", entries)
	}
}

func TestSaveGenerationsAndRestart(t *testing.T) {
	s := newTestStore(t)
	if err := s.Save(sequence(index.Snapshot{Name: "a", Data: []byte("old")}), 1); err != nil {
		t.Fatal(err)
	}
	// A leftover from a crash must not be mistaken for a committed generation.
	if err := os.Mkdir(filepath.Join(s.path, ".snapshot-tmp-crash"), 0o700); err != nil {
		t.Fatal(err)
	}
	reopened, err := NewStore(s.path)
	if err != nil {
		t.Fatal(err)
	}
	if err := reopened.Save(sequence(index.Snapshot{Name: "a", Data: []byte("new")}), 2); err != nil {
		t.Fatal(err)
	}
	for generation, want := range map[uint32]string{1: "old", 2: "new"} {
		m, got := readGeneration(t, s, generation)
		if len(got) != 1 || string(got[0].Data) != want || m.LastIndexedRecordID != domain.RecordID(generation) {
			t.Fatalf("generation %d: %+v %v", generation, m, got)
		}
	}
}

func TestSaveFailuresPreserveGeneration(t *testing.T) {
	sentinel := errors.New("iterator failure")
	cases := map[string]iter.Seq2[index.Snapshot, error]{
		"nil":           nil,
		"empty name":    sequence(index.Snapshot{}),
		"invalid UTF-8": sequence(index.Snapshot{Name: string([]byte{0xff})}),
		"duplicate":     sequence(index.Snapshot{Name: "a"}, index.Snapshot{Name: "a"}),
		"iterator": func(yield func(index.Snapshot, error) bool) {
			if yield(index.Snapshot{Name: "a", Data: []byte("partial")}, nil) {
				yield(index.Snapshot{}, sentinel)
			}
		},
	}
	for name, seq := range cases {
		t.Run(name, func(t *testing.T) {
			s := newTestStore(t)
			if err := s.Save(sequence(index.Snapshot{Name: "original", Data: []byte("old")}), 1); err != nil {
				t.Fatal(err)
			}
			before := s.lastUpdate
			err := s.Save(seq, 2)
			if err == nil {
				t.Fatal("expected failure")
			}
			if name == "iterator" && !errors.Is(err, sentinel) {
				t.Fatalf("lost cause: %v", err)
			}
			if s.lastIndexedRecordID != 1 || !s.lastUpdate.Equal(before) {
				t.Fatal("checkpoint changed on failure")
			}
			last, err := findLastSnapshot(s.path)
			if err != nil || last != 1 {
				t.Fatalf("published partial generation: %d %v", last, err)
			}
			_, got := readGeneration(t, s, 1)
			if len(got) != 1 || string(got[0].Data) != "old" {
				t.Fatal("previous generation changed")
			}
			assertNoTemporaryDirectories(t, s)
		})
	}
}

func TestSaveDirectorySyncFailures(t *testing.T) {
	for _, failAt := range []int{1, 2} {
		t.Run(fmt.Sprint(failAt), func(t *testing.T) {
			s := newTestStore(t)
			sentinel := errors.New("fsync failure")
			calls := 0
			err := s.save(sequence(index.Snapshot{Name: "a", Data: []byte("data")}), 9, func(path string) error {
				calls++
				if calls == failAt {
					return sentinel
				}
				return syncDirectory(path)
			})
			if !errors.Is(err, sentinel) {
				t.Fatalf("lost sync error: %v", err)
			}
			if s.lastIndexedRecordID != 0 || !s.lastUpdate.IsZero() {
				t.Fatal("checkpoint changed on sync failure")
			}
			last, err := findLastSnapshot(s.path)
			if err != nil {
				t.Fatal(err)
			}
			if failAt == 1 && last != 0 {
				t.Fatal("published before syncing temporary directory")
			}
			if failAt == 2 {
				if last != 1 {
					t.Fatal("lost visible generation after rename")
				}
				readGeneration(t, s, 1)
			}
			assertNoTemporaryDirectories(t, s)
			// A retry after an uncertain publication must choose a fresh generation.
			if err := s.Save(sequence(), 10); err != nil {
				t.Fatal(err)
			}
			readGeneration(t, s, last+1)
		})
	}
}

func TestSaveWriteAndRenameFailures(t *testing.T) {
	for _, stage := range []string{"write", "rename"} {
		t.Run(stage, func(t *testing.T) {
			s := newTestStore(t)
			seq := func(yield func(index.Snapshot, error) bool) {
				entries, err := os.ReadDir(s.path)
				if err != nil {
					t.Fatal(err)
				}
				for _, e := range entries {
					if strings.HasPrefix(e.Name(), ".snapshot-tmp-") {
						filename := "index-0001.bin"
						if stage == "rename" {
							filename = "manifest.json"
						}
						if stage == "write" {
							if err := os.Mkdir(filepath.Join(s.path, e.Name(), filename), 0o700); err != nil {
								t.Fatal(err)
							}
						} else {
							// Non-empty destination makes rename fail without replacing its contents.
							dir := filepath.Join(s.path, "snapshot-0001")
							if err := os.Mkdir(dir, 0o700); err != nil {
								t.Fatal(err)
							}
							if err := os.WriteFile(filepath.Join(dir, "sentinel"), []byte("keep"), 0o600); err != nil {
								t.Fatal(err)
							}
						}
					}
				}
				yield(index.Snapshot{Name: "a", Data: []byte("data")}, nil)
			}
			if err := s.Save(seq, 1); err == nil {
				t.Fatal("expected filesystem error")
			}
			if s.lastIndexedRecordID != 0 || !s.lastUpdate.IsZero() {
				t.Fatal("checkpoint advanced")
			}
			assertNoTemporaryDirectories(t, s)
			if stage == "rename" {
				data, err := os.ReadFile(filepath.Join(s.path, "snapshot-0001", "sentinel"))
				if err != nil || string(data) != "keep" {
					t.Fatal("destination overwritten")
				}
			}
		})
	}
}

func TestSaveEmptyAndConcurrent(t *testing.T) {
	s := newTestStore(t)
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			if err := s.Save(sequence(), 0); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	for generation := uint32(1); generation <= 4; generation++ {
		m, got := readGeneration(t, s, generation)
		if len(got) != 0 || m.Indexes == nil {
			t.Fatal("invalid empty generation")
		}
	}
	assertNoTemporaryDirectories(t, s)
}

func TestFindLastSnapshotValidation(t *testing.T) {
	for _, name := range []string{"snapshot-0000", "snapshot-1", "snapshot-+001", "snapshot-nope", "snapshot-4294967296"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.Mkdir(filepath.Join(dir, name), 0o700); err != nil {
				t.Fatal(err)
			}
			if _, err := findLastSnapshot(dir); err == nil {
				t.Fatal("accepted invalid generation")
			}
		})
	}
	for _, kind := range []string{"file", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "snapshot-0001")
			var err error
			if kind == "file" {
				err = os.WriteFile(path, nil, 0o600)
			} else {
				err = os.Symlink(t.TempDir(), path)
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := findLastSnapshot(dir); err == nil {
				t.Fatal("accepted non-directory generation")
			}
		})
	}
	s := newTestStore(t)
	if err := os.Mkdir(filepath.Join(s.path, fmt.Sprintf("snapshot-%04d", uint32(math.MaxUint32))), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(sequence(), 0); err == nil {
		t.Fatal("accepted generation overflow")
	}
	assertNoTemporaryDirectories(t, s)
}

func TestSavePublishesOnlyAfterCompleteIteration(t *testing.T) {
	s := newTestStore(t)
	seq := func(yield func(index.Snapshot, error) bool) {
		for _, name := range []string{"first", "second"} {
			if !yield(index.Snapshot{Name: name, Data: []byte(name)}, nil) {
				return
			}
			last, err := findLastSnapshot(s.path)
			if err != nil || last != 0 {
				t.Fatalf("generation visible while still iterating: %d %v", last, err)
			}
		}
	}
	calls := 0
	err := s.save(seq, 2, func(path string) error {
		calls++
		if calls == 1 {
			// All files and the manifest exist before syncing the temporary directory.
			data, err := os.ReadFile(filepath.Join(path, "manifest.json"))
			if err != nil {
				t.Fatal(err)
			}
			var m manifest
			if err := json.Unmarshal(data, &m); err != nil {
				t.Fatal(err)
			}
			if len(m.Indexes) != 2 {
				t.Fatal("incomplete manifest before directory sync")
			}
			last, err := findLastSnapshot(s.path)
			if err != nil || last != 0 {
				t.Fatal("published before temporary directory sync")
			}
		} else {
			if path != s.path {
				t.Fatalf("expected parent directory sync, got %s", path)
			}
			readGeneration(t, s, 1)
			if s.lastIndexedRecordID != 0 || !s.lastUpdate.IsZero() {
				t.Fatal("checkpoint updated before parent sync")
			}
		}
		return syncDirectory(path)
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("directory sync calls: %d", calls)
	}
}
