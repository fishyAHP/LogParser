package snapshot

import (
	"cmp"
	"errors"
	"fmt"
	"iter"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index"
	"fishyAHP/LogParser.git/internal/features/snapshot/manifest"
)

var (
	ErrNoSnapshots      = errors.New("no snapshots in directory")
	ErrNoValidSnapshots = errors.New("no valid snapshots found")
)

func generationName(generation uint32) string {
	return fmt.Sprintf("snapshot-%04d", generation)
}

type Store struct {
	mtx                 sync.Mutex
	path                string
	lastIndexedRecordID domain.RecordID
	lastUpdate          time.Time
}

func NewStore(
	path string,
) (*Store, error) {
	if err := os.MkdirAll(
		path,
		0o755,
	); err != nil {
		return nil, fmt.Errorf(
			"create snapshot directory '%s': %w",
			path,
			err,
		)
	}

	return &Store{
		path: path,
	}, nil
}

func findLastSnapshot(
	dir string,
) (uint32, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf(
			"read dir: %w",
			err,
		)
	}

	var lastSnapshot uint32

	for _, entry := range entries {
		name := entry.Name()
		withoutPrefix, ok := strings.CutPrefix(
			name,
			"snapshot-",
		)
		if !ok {
			continue
		}

		if !entry.IsDir() {
			return 0, fmt.Errorf(
				"snapshot %q isn't directory: %v",
				name,
				entry.Type(),
			)
		}

		id, err := strconv.ParseUint(
			withoutPrefix,
			10,
			32,
		)
		if err != nil {
			return 0, fmt.Errorf(
				"corrupted number of snapshot: %w",
				err,
			)
		}

		if id == 0 {
			return 0, errors.New(
				"snapshot id cannot be zero",
			)
		}

		generation := uint32(id)
		expect := generationName(generation)
		if entry.Name() != expect {
			return 0, fmt.Errorf(
				"invalid snapshot filename: %s, expected %s",
				entry.Name(),
				expect,
			)
		}

		lastSnapshot = max(lastSnapshot, generation)
	}

	return lastSnapshot, nil
}

func (s *Store) Save(
	snapshots iter.Seq2[index.Snapshot, error],
	lastRecordID domain.RecordID,
) (err error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	generation, err := findLastSnapshot(s.path)
	if err != nil {
		return fmt.Errorf(
			"find last snapshot: %w",
			err,
		)
	}

	if generation == math.MaxUint32 {
		return errors.New(
			"snapshot generation overflow",
		)
	}
	generation++

	w, err := s.writer(
		lastRecordID,
		generation,
	)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, os.RemoveAll(w.tmpDir))
	}()

	for snapshot, snapshotErr := range snapshots {
		if snapshotErr != nil {
			return fmt.Errorf(
				"snapshots error: %w",
				snapshotErr,
			)
		}

		if err = w.writeIndex(
			snapshot.Name,
			snapshot.Data,
		); err != nil {
			return fmt.Errorf(
				"write index: %w",
				err,
			)
		}
	}

	if err = w.writeManifest(); err != nil {
		return fmt.Errorf(
			"write manifest: %w",
			err,
		)
	}

	if err = w.publish(); err != nil {
		return fmt.Errorf(
			"publish manifest with indexes: %w",
			err,
		)
	}

	s.lastIndexedRecordID = lastRecordID
	s.lastUpdate = w.m.CreatedAt

	return nil
}

func (s *Store) writer(
	lastRecordID domain.RecordID,
	generation uint32,
) (*writer, error) {
	tmpDir, err := os.MkdirTemp(
		s.path,
		"temp-snapshot-",
	)
	if err != nil {
		return nil, fmt.Errorf(
			"make tmp dir: %w",
			err,
		)
	}

	finalDir := filepath.Join(
		s.path,
		generationName(generation),
	)

	m := manifest.Manifest{
		Version:             manifest.Version1,
		Generation:          generation,
		LastIndexedRecordID: lastRecordID,
		CreatedAt:           time.Now().UTC(),
	}

	return &writer{
		tmpDir:   tmpDir,
		finalDir: finalDir,
		m:        m,
	}, nil
}

func (s *Store) reader(
	dir string,
) *reader {
	return &reader{
		dir: dir,
	}
}

type Rebuilder interface {
	FromSnapshots(
		snapshots iter.Seq2[index.Snapshot, error],
	) error
}

func (s *Store) Load(rebuilder Rebuilder) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	generations, err := findSnapshotGenerations(s.path)
	if err != nil {
		return fmt.Errorf(
			"find snapshot generations: %w", err,
		)
	}

	if len(generations) == 0 {
		return ErrNoSnapshots
	}

	var failures []error

	for _, generation := range generations {
		m, err := s.loadGeneration(generation, rebuilder)
		if err != nil {
			failures = append(failures, fmt.Errorf(
				"snapshot generation %d: %w",
				generation, err,
			))
			continue
		}

		s.lastIndexedRecordID = m.LastIndexedRecordID
		s.lastUpdate = m.CreatedAt

		s.lastIndexedRecordID = m.LastIndexedRecordID
		s.lastUpdate = m.CreatedAt

		fmt.Printf(
			"Snapshot loaded: generation=%d, checkpoint=%d\n",
			generation,
			m.LastIndexedRecordID,
		)

		if len(failures) > 0 {
			fmt.Printf(
				"Snapshot fallback: %d newer generation(s) rejected\n",
				len(failures),
			)
		}

		return nil
	}

	return errors.Join(
		append([]error{ErrNoValidSnapshots}, failures...)...,
	)
}

func (s *Store) Checkpoint() (domain.RecordID, time.Time) {
	return s.lastIndexedRecordID, s.lastUpdate
}

func findSnapshotGenerations(
	dir string,
) ([]uint32, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf(
			"read snapshot directory: %w",
			err,
		)
	}

	generations := make([]uint32, 0)

	for _, entry := range entries {
		name := entry.Name()

		number, ok := strings.CutPrefix(
			name,
			"snapshot-",
		)
		if !ok {
			continue
		}

		id, err := strconv.ParseUint(number, 10, 32)
		if err != nil ||
			id == 0 {
			return nil, fmt.Errorf(
				"invalid snapshot generation: %q",
				name,
			)
		}

		generation := uint32(id)
		expected := generationName(generation)
		if name != expected ||
			!entry.IsDir() {
			return nil, fmt.Errorf(
				"invalid snapshot directory: %q",
				name,
			)
		}

		generations = append(generations, generation)
	}

	slices.SortFunc(generations, func(a, b uint32) int {
		return cmp.Compare(b, a)
	})

	return generations, nil
}

func (s *Store) loadGeneration(
	generation uint32,
	rebuilder Rebuilder,
) (manifest.Manifest, error) {
	dir := filepath.Join(
		s.path,
		generationName(generation),
	)

	r := s.reader(dir)

	if err := r.readManifest(); err != nil {
		return manifest.Manifest{}, fmt.Errorf(
			"error manifest: %w",
			err,
		)
	}

	if r.m.Generation != generation {
		return manifest.Manifest{}, fmt.Errorf(
			"snapshot generation mismatch: dir %d, manifest %d",
			generation,
			r.m.Generation,
		)
	}

	if err := rebuilder.FromSnapshots(
		r.snapshots(),
	); err != nil {
		return manifest.Manifest{}, fmt.Errorf(
			"rebuild from snapshots: %w",
			err,
		)
	}

	return r.m, nil
}
