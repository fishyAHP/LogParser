package snapshot

import (
	"errors"
	"fmt"
	"iter"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index"
	"fishyAHP/LogParser.git/internal/features/storage/snapshot/manifest"
)

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

	store := &Store{
		path: path,
	}

	m, err := store.readLatest(
		func(snapshots iter.Seq2[index.Snapshot, error]) error {
			for _, err := range snapshots {
				if err != nil {
					return err
				}
			}
			return nil
		},
	)
	if errors.Is(err, ErrNoSnapshots) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf(
			"open snapshot store: %w",
			err,
		)
	}

	store.lastIndexedRecordID = m.LastIndexedRecordID
	store.lastUpdate = m.CreatedAt

	return store, nil
}

func findLastSnapshot(dir string) (uint32, error) {
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

		expectedName := fmt.Sprintf(
			"snapshot-%04d",
			id,
		)

		if entry.Name() != expectedName {
			return 0, fmt.Errorf(
				"invalid snapshot filename: %s, expected %s",
				entry.Name(),
				expectedName,
			)
		}

		lastSnapshot = max(lastSnapshot, uint32(id))
	}

	return lastSnapshot, nil
}

func (s *Store) Save(
	snapshots iter.Seq2[index.Snapshot, error],
	lastRecordID domain.RecordID,
) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	generation, err := s.nextGeneration()
	if err != nil {
		return fmt.Errorf(
			"create new manifest generation: %w",
			err,
		)
	}

	metadata, err := s.writeGeneration(
		generation,
		snapshots,
		lastRecordID,
	)
	if err != nil {
		return fmt.Errorf(
			"write generation snapshots: %w",
			err,
		)
	}

	s.lastIndexedRecordID = metadata
	s.lastUpdate = metadata.
}

func (s *Store) nextGeneration() (manifest.Manifest, error) {

}



