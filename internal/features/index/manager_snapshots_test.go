package index

import (
	"bytes"
	"errors"
	"fmt"
	"iter"
	"maps"
	"slices"
	"testing"

	"fishyAHP/LogParser.git/internal/core/domain"
)

func snapshotTestScheme() *domain.Scheme {
	return &domain.Scheme{
		Separator: ',',
		Parameters: []domain.Field{
			{
				Name:      "level",
				FieldType: domain.StringType,
				IndexType: domain.HashIndex,
			},
			{
				Name:      "timestamp",
				FieldType: domain.IntType,
				IndexType: domain.RangeIndex,
			},
			{
				Name:      "message",
				FieldType: domain.StringType,
				IndexType: domain.TextIndex,
			},
			{
				Name:      "source",
				FieldType: domain.StringType,
				IndexType: domain.NoIndex,
			},
		},
	}
}

func snapshotTestManager(t *testing.T) *Manager {
	t.Helper()

	m, err := New(snapshotTestScheme())
	if err != nil {
		t.Fatal(err)
	}

	for _, row := range []struct {
		name  string
		value domain.Value
		id    domain.RecordID
	}{
		{"level", domain.StringValue("INFO"), 1},
		{"level", domain.StringValue("ERROR"), 2},
		{"timestamp", domain.IntValue(100), 1},
		{"timestamp", domain.IntValue(200), 2},
		{"message", domain.StringValue("alpha beta"), 1},
		{"message", domain.StringValue("beta gamma"), 2},
	} {
		if err := m.indexes[row.name].Add(row.value, row.id); err != nil {
			t.Fatalf("add to %q: %v", row.name, err)
		}
	}

	return m
}

func collectSnapshots(t *testing.T, m *Manager) []Snapshot {
	t.Helper()

	var result []Snapshot

	for snapshot, err := range m.MakeSnapshots() {
		if err != nil {
			t.Fatal(err)
		}

		result = append(result, Snapshot{
			Name: snapshot.Name,
			Data: bytes.Clone(snapshot.Data),
		})
	}

	return result
}

func snapshotSequence(items []Snapshot) iter.Seq2[Snapshot, error] {
	return func(yield func(Snapshot, error) bool) {
		for _, item := range items {
			if !yield(item, nil) {
				return
			}
		}
	}
}

func snapshotMap(items []Snapshot) map[string][]byte {
	result := make(map[string][]byte, len(items))
	for _, item := range items {
		result[item.Name] = item.Data
	}
	return result
}

func assertSnapshotsEqual(t *testing.T, want, got []Snapshot) {
	t.Helper()

	a := snapshotMap(want)
	b := snapshotMap(got)

	if !maps.EqualFunc(a, b, bytes.Equal) {
		t.Fatalf("snapshot sets differ: want %d, got %d",
			len(a), len(b))
	}
}

func TestManagerSnapshotsRoundTrip(t *testing.T) {
	source := snapshotTestManager(t)
	snapshots := collectSnapshots(t, source)

	if len(snapshots) != 3 {
		t.Fatalf("got %d snapshots, want 3", len(snapshots))
	}

	restored, err := New(snapshotTestScheme())
	if err != nil {
		t.Fatal(err)
	}

	if err := restored.FromSnapshots(snapshotSequence(snapshots)); err != nil {
		t.Fatal(err)
	}

	assertSnapshotsEqual(
		t,
		snapshots,
		collectSnapshots(t, restored),
	)

	ids, err := restored.Exact("level", domain.StringValue("ERROR"))
	if err != nil {
		t.Fatal(err)
	}

	if got := slices.Collect(ids); !slices.Equal(got, []domain.RecordID{2}) {
		t.Fatalf("restored hash search: %v", got)
	}

	ids, err = restored.Exact("timestamp", domain.IntValue(200))
	if err != nil {
		t.Fatal(err)
	}

	if got := slices.Collect(ids); !slices.Equal(got, []domain.RecordID{2}) {
		t.Fatalf("restored range search: %v", got)
	}
}

func TestManagerSnapshotsMissing(t *testing.T) {
	snapshots := collectSnapshots(t, snapshotTestManager(t))

	for skip := range snapshots {
		t.Run(snapshots[skip].Name, func(t *testing.T) {
			partial := slices.Clone(snapshots)
			partial = slices.Delete(partial, skip, skip+1)

			target, err := New(snapshotTestScheme())
			if err != nil {
				t.Fatal(err)
			}

			if err := target.FromSnapshots(snapshotSequence(partial)); err == nil {
				t.Fatal("accepted incomplete snapshots")
			}
		})
	}
}

func TestManagerSnapshotsDuplicate(t *testing.T) {
	snapshots := collectSnapshots(t, snapshotTestManager(t))
	snapshots = append(snapshots, snapshots[0])

	target, err := New(snapshotTestScheme())
	if err != nil {
		t.Fatal(err)
	}

	if err := target.FromSnapshots(snapshotSequence(snapshots)); err == nil {
		t.Fatal("accepted duplicate snapshot")
	}
}

func TestManagerSnapshotsUnknownField(t *testing.T) {
	snapshots := collectSnapshots(t, snapshotTestManager(t))
	snapshots[0].Name = "unknown"

	target, err := New(snapshotTestScheme())
	if err != nil {
		t.Fatal(err)
	}

	if err := target.FromSnapshots(snapshotSequence(snapshots)); err == nil {
		t.Fatal("accepted unknown field")
	}
}

func TestManagerSnapshotsNoIndexField(t *testing.T) {
	snapshots := collectSnapshots(t, snapshotTestManager(t))
	snapshots[0].Name = "source"

	target, err := New(snapshotTestScheme())
	if err != nil {
		t.Fatal(err)
	}

	if err := target.FromSnapshots(snapshotSequence(snapshots)); err == nil {
		t.Fatal("accepted snapshot for NoIndex field")
	}
}

func TestManagerSnapshotsPreserveOnCorruption(t *testing.T) {
	target := snapshotTestManager(t)
	before := collectSnapshots(t, target)

	snapshots := collectSnapshots(t, snapshotTestManager(t))

	// Corrupt the last snapshot so previous indexes
	// have already been decoded into temporary state.
	last := len(snapshots) - 1
	snapshots[last].Data = snapshots[last].Data[:1]

	if err := target.FromSnapshots(snapshotSequence(snapshots)); err == nil {
		t.Fatal("expected corrupted snapshot error")
	}

	after := collectSnapshots(t, target)
	assertSnapshotsEqual(t, before, after)
}

func TestManagerSnapshotsIteratorError(t *testing.T) {
	sentinel := errors.New("snapshot read failed")

	seq := func(yield func(Snapshot, error) bool) {
		yield(Snapshot{}, sentinel)
	}

	target := snapshotTestManager(t)
	before := collectSnapshots(t, target)

	err := target.FromSnapshots(seq)
	if !errors.Is(err, sentinel) {
		t.Fatalf("got %v, want wrapped sentinel", err)
	}

	assertSnapshotsEqual(
		t,
		before,
		collectSnapshots(t, target),
	)
}

func TestManagerSnapshotsEarlyStop(t *testing.T) {
	m := snapshotTestManager(t)
	received := 0

	for snapshot, err := range m.MakeSnapshots() {
		if err != nil {
			t.Fatal(err)
		}
		if snapshot.Name == "" || len(snapshot.Data) == 0 {
			t.Fatal("invalid snapshot")
		}

		received++
		break
	}

	if received != 1 {
		t.Fatalf("received %d snapshots, want 1", received)
	}
}

func TestManagerSnapshotsEmptySequence(t *testing.T) {
	target := snapshotTestManager(t)
	before := collectSnapshots(t, target)

	empty := func(yield func(Snapshot, error) bool) {}

	if err := target.FromSnapshots(empty); err == nil {
		t.Fatal("accepted empty snapshot sequence")
	}

	assertSnapshotsEqual(t, before, collectSnapshots(t, target))
}

func TestManagerSnapshotsIterationErrorAfterOne(t *testing.T) {
	target := snapshotTestManager(t)
	before := collectSnapshots(t, target)

	snapshots := collectSnapshots(t, target)
	sentinel := fmt.Errorf("disk read failure")

	seq := func(yield func(Snapshot, error) bool) {
		if !yield(snapshots[0], nil) {
			return
		}
		yield(Snapshot{}, sentinel)
	}

	err := target.FromSnapshots(seq)
	if !errors.Is(err, sentinel) {
		t.Fatalf("got %v, want %v", err, sentinel)
	}

	assertSnapshotsEqual(t, before, collectSnapshots(t, target))
}
