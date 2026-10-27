package snapshot

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index"
)

func testManager(t *testing.T, id domain.RecordID) *index.Manager {
	t.Helper()
	manager, err := index.New(&domain.Scheme{Parameters: []domain.Field{
		{Name: "a-level", FieldType: domain.StringType, IndexType: domain.HashIndex},
		{Name: "b-time", FieldType: domain.IntType, IndexType: domain.RangeIndex},
		{Name: "z-message", FieldType: domain.StringType, IndexType: domain.TextIndex},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Index(id, domain.LogEntry{Values: []domain.Value{
		domain.StringValue("ERROR"), domain.IntValue(100), domain.StringValue("alpha beta"),
	}}); err != nil {
		t.Fatal(err)
	}
	return manager
}

func managerData(t *testing.T, m *index.Manager) map[string][]byte {
	t.Helper()
	data := make(map[string][]byte)
	for snapshot, err := range m.MakeSnapshots() {
		if err != nil {
			t.Fatal(err)
		}
		data[snapshot.Name] = bytes.Clone(snapshot.Data)
	}
	return data
}

func assertManagerData(t *testing.T, m *index.Manager, want map[string][]byte) {
	t.Helper()
	got := managerData(t, m)
	if len(got) != len(want) {
		t.Fatal("index count changed")
	}
	for name, data := range want {
		if !bytes.Equal(data, got[name]) {
			t.Fatalf("index %q changed", name)
		}
	}
}

func rewriteManifest(t *testing.T, s *Store, generation uint32, change func(map[string]any)) {
	t.Helper()
	path := filepath.Join(s.path, fmt.Sprintf("snapshot-%04d", generation), "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	change(m)
	data, err = json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadLatestAndReopenCheckpoint(t *testing.T) {
	s := newTestStore(t)
	for _, id := range []domain.RecordID{1, 42} {
		if err := s.Save(testManager(t, id).MakeSnapshots(), id); err != nil {
			t.Fatal(err)
		}
	}
	// A crashed unfinished save is never selected.
	if err := os.Mkdir(filepath.Join(s.path, ".snapshot-tmp-crash"), 0o700); err != nil {
		t.Fatal(err)
	}
	reopened, err := NewStore(s.path)
	if err != nil {
		t.Fatal(err)
	}
	wantID, wantTime := s.Checkpoint()
	id, updated := reopened.Checkpoint()
	if id != wantID || !updated.Equal(wantTime) {
		t.Fatalf("reopened checkpoint %d %v", id, updated)
	}
	target := testManager(t, 99)
	if err := reopened.Load(target); err != nil {
		t.Fatal(err)
	}
	assertManagerData(t, target, managerData(t, testManager(t, 42)))
	ids, err := target.Exact("a-level", domain.StringValue("ERROR"))
	if err != nil {
		t.Fatal(err)
	}
	if got := slices.Collect(ids); !slices.Equal(got, []domain.RecordID{42}) {
		t.Fatalf("restored query: %v", got)
	}
	if err := reopened.Save(testManager(t, 43).MakeSnapshots(), 43); err != nil {
		t.Fatal(err)
	}
	readGeneration(t, reopened, 3)
}

func TestLoadEmptyStoreAndEmptyGeneration(t *testing.T) {
	s := newTestStore(t)
	target := testManager(t, 99)
	before := managerData(t, target)
	if err := s.Load(target); !errors.Is(err, ErrNoSnapshots) {
		t.Fatalf("expected ErrNoSnapshots, got %v", err)
	}
	assertManagerData(t, target, before)
	if id, updated := s.Checkpoint(); id != 0 || !updated.IsZero() {
		t.Fatal("empty store has a checkpoint")
	}
	if err := s.Load(nil); err == nil {
		t.Fatal("accepted nil manager")
	}
	if err := s.Save(sequence(), 0); err != nil {
		t.Fatal(err)
	}
	reopened, err := NewStore(s.path)
	if err != nil {
		t.Fatal(err)
	}
	empty, err := index.New(&domain.Scheme{})
	if err != nil {
		t.Fatal(err)
	}
	if err := reopened.Load(empty); err != nil {
		t.Fatal(err)
	}
	if id, updated := reopened.Checkpoint(); id != 0 || updated.IsZero() {
		t.Fatal("zero checkpoint confused with no generation")
	}
	if err := reopened.Load(target); err == nil {
		t.Fatal("accepted missing indexes")
	}
	assertManagerData(t, target, before)
}

func TestLoadCorruptionPreservesManagerAndCheckpoint(t *testing.T) {
	cases := map[string]func(*testing.T, *Store){
		"checksum": func(t *testing.T, s *Store) {
			_, items := readGeneration(t, s, 2)
			if len(items) == 0 {
				t.Fatal("no data")
			}
			path := filepath.Join(s.path, "snapshot-0002", "index-0001.bin")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			data[len(data)-1] ^= 1
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"size": func(t *testing.T, s *Store) {
			if err := os.WriteFile(filepath.Join(s.path, "snapshot-0002", "index-0001.bin"), nil, 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"missing index": func(t *testing.T, s *Store) {
			if err := os.Remove(filepath.Join(s.path, "snapshot-0002", "index-0001.bin")); err != nil {
				t.Fatal(err)
			}
		},
		"missing manifest": func(t *testing.T, s *Store) {
			if err := os.Remove(filepath.Join(s.path, "snapshot-0002", "manifest.json")); err != nil {
				t.Fatal(err)
			}
		},
		"index directory": func(t *testing.T, s *Store) {
			path := filepath.Join(s.path, "snapshot-0002", "index-0001.bin")
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(path, 0o700); err != nil {
				t.Fatal(err)
			}
		},
		"index symlink": func(t *testing.T, s *Store) {
			path := filepath.Join(s.path, "snapshot-0002", "index-0001.bin")
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("index-0002.bin", path); err != nil {
				t.Fatal(err)
			}
		},
		"manifest symlink": func(t *testing.T, s *Store) {
			path := filepath.Join(s.path, "snapshot-0002", "manifest.json")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			external := filepath.Join(t.TempDir(), "manifest.json")
			if err := os.WriteFile(external, data, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(external, path); err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, damage := range cases {
		t.Run(name, func(t *testing.T) {
			s := newTestStore(t)
			for _, id := range []domain.RecordID{1, 2} {
				if err := s.Save(testManager(t, id).MakeSnapshots(), id); err != nil {
					t.Fatal(err)
				}
			}
			target := testManager(t, 99)
			before := managerData(t, target)
			beforeID, beforeTime := s.Checkpoint()
			damage(t, s)
			if err := s.Load(target); err == nil {
				t.Fatal("accepted corruption or silently used older generation")
			}
			assertManagerData(t, target, before)
			if id, updated := s.Checkpoint(); id != beforeID || !updated.Equal(beforeTime) {
				t.Fatal("checkpoint changed on failed load")
			}
			if _, err := NewStore(s.path); err == nil {
				t.Fatal("reopened corrupt generation")
			}
		})
	}
}

func TestManifestValidation(t *testing.T) {
	changeEntry := func(fn func(map[string]any)) func(map[string]any) {
		return func(m map[string]any) { fn(m["indexes"].([]any)[0].(map[string]any)) }
	}
	cases := map[string]func(map[string]any){
		"version":            func(m map[string]any) { m["version"] = 2 },
		"generation":         func(m map[string]any) { m["generation"] = 2 },
		"missing checkpoint": func(m map[string]any) { delete(m, "last_indexed_record_id") },
		"null checkpoint":    func(m map[string]any) { m["last_indexed_record_id"] = nil },
		"zero time":          func(m map[string]any) { m["created_at"] = time.Time{} },
		"null indexes":       func(m map[string]any) { m["indexes"] = nil },
		"unknown field":      func(m map[string]any) { m["unexpected"] = true },
		"duplicate name": func(m map[string]any) {
			entries := m["indexes"].([]any)
			entries[1].(map[string]any)["name"] = entries[0].(map[string]any)["name"]
		},
		"duplicate file": func(m map[string]any) {
			entries := m["indexes"].([]any)
			entries[1].(map[string]any)["filename"] = entries[0].(map[string]any)["filename"]
		},
		"traversal":         changeEntry(func(e map[string]any) { e["filename"] = "../index-0001.bin" }),
		"absolute path":     changeEntry(func(e map[string]any) { e["filename"] = "/tmp/index-0001.bin" }),
		"metadata filename": changeEntry(func(e map[string]any) { e["filename"] = "manifest.json" }),
		"bad ordinal":       changeEntry(func(e map[string]any) { e["filename"] = "index-9999.bin" }),
		"empty name":        changeEntry(func(e map[string]any) { e["name"] = "" }),
		"negative size":     changeEntry(func(e map[string]any) { e["size"] = -1 }),
		"missing size":      changeEntry(func(e map[string]any) { delete(e, "size") }),
		"bad digest":        changeEntry(func(e map[string]any) { e["sha256"] = "invalid" }),
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			s := newTestStore(t)
			if err := s.Save(testManager(t, 1).MakeSnapshots(), 1); err != nil {
				t.Fatal(err)
			}
			rewriteManifest(t, s, 1, change)
			target := testManager(t, 99)
			before := managerData(t, target)
			if err := s.Load(target); err == nil {
				t.Fatal("accepted invalid manifest")
			}
			assertManagerData(t, target, before)
			if _, err := NewStore(s.path); err == nil {
				t.Fatal("opened invalid manifest")
			}
		})
	}
	for _, data := range [][]byte{[]byte("{"), []byte("null"), []byte("{} {}"), {0xff}} {
		t.Run(fmt.Sprintf("json-%x", data), func(t *testing.T) {
			if _, err := decodeManifest(data, 1); err == nil {
				t.Fatal("accepted invalid JSON manifest")
			}
		})
	}
}

func TestLoadDecodeFailureIsAtomic(t *testing.T) {
	s := newTestStore(t)
	if err := s.Save(testManager(t, 1).MakeSnapshots(), 1); err != nil {
		t.Fatal(err)
	}
	// Damage the last index but give it a matching size and checksum. Disk-level
	// validation succeeds; the index codec must still reject the data atomically.
	rewriteManifest(t, s, 1, func(m map[string]any) {
		entries := m["indexes"].([]any)
		entry := entries[len(entries)-1].(map[string]any)
		data := []byte{0}
		digest := sha256.Sum256(data)
		entry["size"] = len(data)
		entry["sha256"] = hex.EncodeToString(digest[:])
		if err := os.WriteFile(filepath.Join(s.path, "snapshot-0001", entry["filename"].(string)), data, 0o600); err != nil {
			t.Fatal(err)
		}
	})
	reopened, err := NewStore(s.path)
	if err != nil {
		t.Fatalf("disk-level validation should pass: %v", err)
	}
	target := testManager(t, 99)
	before := managerData(t, target)
	if err := reopened.Load(target); err == nil {
		t.Fatal("accepted invalid index encoding")
	}
	assertManagerData(t, target, before)
}

func TestLoadRejectsIncompatibleScheme(t *testing.T) {
	s := newTestStore(t)
	if err := s.Save(testManager(t, 1).MakeSnapshots(), 1); err != nil {
		t.Fatal(err)
	}
	target, err := index.New(&domain.Scheme{Parameters: []domain.Field{{Name: "other", FieldType: domain.StringType, IndexType: domain.HashIndex}}})
	if err != nil {
		t.Fatal(err)
	}
	before := managerData(t, target)
	if err := s.Load(target); err == nil {
		t.Fatal("accepted incompatible scheme")
	}
	assertManagerData(t, target, before)
}

func TestConcurrentSaveLoadCheckpoint(t *testing.T) {
	s := newTestStore(t)
	if err := s.Save(sequence(), 0); err != nil {
		t.Fatal(err)
	}
	target, err := index.New(&domain.Scheme{})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for range 3 {
				if err := s.Save(sequence(), 0); err != nil {
					t.Error(err)
				}
			}
		})
		wg.Go(func() {
			for range 3 {
				if err := s.Load(target); err != nil {
					t.Error(err)
				}
			}
		})
		wg.Go(func() {
			for range 3 {
				s.Checkpoint()
			}
		})
	}
	wg.Wait()
}

func TestCheckpointPreservesUint64(t *testing.T) {
	s := newTestStore(t)
	want := domain.RecordID(^uint64(0))
	if err := s.Save(sequence(), want); err != nil {
		t.Fatal(err)
	}
	reopened, err := NewStore(s.path)
	if err != nil {
		t.Fatal(err)
	}
	if id, _ := reopened.Checkpoint(); id != want {
		t.Fatalf("lost checkpoint precision: %d", id)
	}
}

func TestManifestRejectsDuplicateJSONFields(t *testing.T) {
	for _, data := range []string{
		`{"version":1,"version":1}`, `{"size":null,"size":0}`,
	} {
		if err := requireFields([]byte(data)); err == nil {
			t.Fatal("accepted duplicate JSON field")
		}
	}
	s := newTestStore(t)
	if err := s.Save(sequence(), 0); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(s.path, "snapshot-0001", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	data = bytes.Replace(data, []byte(`"version": 1`), []byte(`"version": 1,"VERSION": 1`), 1)
	if _, err := decodeManifest(data, 1); err == nil {
		t.Fatal("accepted ambiguous field casing")
	}
}
