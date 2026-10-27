package snapshot

import (
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
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index"
)

type Store struct {
	mtx                 sync.Mutex
	path                string
	lastIndexedRecordID domain.RecordID
	lastUpdate          time.Time
}

const manifestVersion = 1

type manifest struct {
	Version             int             `json:"version"`
	Generation          uint32          `json:"generation"`
	LastIndexedRecordID domain.RecordID `json:"last_indexed_record_id"`
	CreatedAt           time.Time       `json:"created_at"`
	Indexes             []manifestIndex `json:"indexes"`
}

type manifestIndex struct {
	Name     string `json:"name"`
	Filename string `json:"filename"`
	Size     int    `json:"size"`
	SHA256   string `json:"sha256"`
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
	lastIndexedRecordID domain.RecordID,
) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	return s.save(
		snapshots,
		lastIndexedRecordID,
		syncDirectory,
	)
}

func (s *Store) save(
	snapshots iter.Seq2[index.Snapshot, error],
	lastIndexedRecordID domain.RecordID,
	syncDir func(string) error,
) (err error) {
	if snapshots == nil {
		return errors.New(
			"snapshot iterator is nil",
		)
	}

	last, err := findLastSnapshot(s.path)
	if err != nil {
		return fmt.Errorf(
			"find last snapshot: %w",
			err,
		)
	}

	if last == math.MaxUint32 {
		return errors.New(
			"snapshot generation overflow",
		)
	}

	tmp, err := os.MkdirTemp(
		s.path,
		".snapshot-tmp-",
	)
	if err != nil {
		return fmt.Errorf(
			"create temporary snapshot directory: %w",
			err,
		)
	}
	defer func() {
		if cleanupErr := os.RemoveAll(
			tmp,
		); cleanupErr != nil {
			err = errors.Join(
				err,
				fmt.Errorf(
					"remove temporary snapshot: %w",
					cleanupErr,
				),
			)
		}
	}()

	m := manifest{
		Version:             manifestVersion,
		Generation:          last + 1,
		LastIndexedRecordID: lastIndexedRecordID,
		CreatedAt:           time.Now().UTC(),
		Indexes:             make([]manifestIndex, 0),
	}
	seen := make(map[string]struct{})

	for snapshot, iterErr := range snapshots {
		if iterErr != nil {
			return fmt.Errorf(
				"make index snapshot: %w",
				iterErr,
			)
		}

		if snapshot.Name == "" ||
			!utf8.ValidString(snapshot.Name) {
			return errors.New(
				"snapshot index name must be non-empty valid UTF-8",
			)
		}

		if _, ok := seen[snapshot.Name]; ok {
			return fmt.Errorf(
				"duplicate snapshot index %q",
				snapshot.Name,
			)
		}

		seen[snapshot.Name] = struct{}{}

		filename := fmt.Sprintf(
			"index-%04d.bin",
			len(m.Indexes)+1,
		)

		if err = writeSyncedFile(
			filepath.Join(tmp, filename),
			snapshot.Data,
		); err != nil {
			return fmt.Errorf(
				"write snapshot index %q: %w",
				snapshot.Name,
				err,
			)
		}

		digest := sha256.Sum256(snapshot.Data)
		m.Indexes = append(m.Indexes, manifestIndex{
			Name:     snapshot.Name,
			Filename: filename,
			Size:     len(snapshot.Data),
			SHA256:   hex.EncodeToString(digest[:]),
		})
	}

	slices.SortFunc(
		m.Indexes,
		func(a, b manifestIndex) int {
			return strings.Compare(a.Name, b.Name)
		},
	)

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf(
			"encode snapshot manifest: %w",
			err,
		)
	}

	if err = writeSyncedFile(
		filepath.Join(tmp, "manifest.json"),
		data,
	); err != nil {
		return fmt.Errorf(
			"write snapshot manifest: %w",
			err,
		)
	}
	if err = syncDir(tmp); err != nil {
		return fmt.Errorf(
			"sync temporary snapshot directory: %w",
			err,
		)
	}

	final := filepath.Join(
		s.path,
		fmt.Sprintf("snapshot-%04d", m.Generation),
	)

	if err = os.Rename(tmp, final); err != nil {
		return fmt.Errorf(
			"publish snapshot generation: %w",
			err,
		)
	}

	if err = syncDir(s.path); err != nil {
		return fmt.Errorf(
			"sync published snapshot directory: %w",
			err,
		)
	}

	s.lastIndexedRecordID = lastIndexedRecordID
	s.lastUpdate = m.CreatedAt

	return nil
}

func writeSyncedFile(path string, data []byte) error {
	file, err := os.OpenFile(
		path,
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		0o644,
	)
	if err != nil {
		return fmt.Errorf(
			"open file %q: %w",
			path,
			err,
		)
	}
	defer func() {
		_ = file.Close()
	}()

	if _, err = file.Write(data); err != nil {
		return fmt.Errorf(
			"file write data: %w",
			err,
		)
	}

	return file.Sync()
}

func syncDirectory(path string) (err error) {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf(
			"open directory %q: %w",
			path,
			err,
		)
	}

	defer func() {
		err = errors.Join(err, file.Close())
	}()

	return file.Sync()
}
