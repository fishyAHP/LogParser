package snapshot

import (
	"errors"
	"fmt"
	"iter"
	"math"
	"os"
	"path/filepath"
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
		fmt.Sprintf("snapshot-%04d", generation),
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
