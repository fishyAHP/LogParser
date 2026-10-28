package snapshot

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index"
	"fishyAHP/LogParser.git/internal/features/storage/snapshot/manifest"
)

var ErrNoSnapshots = errors.New("no snapshot generations")

func (s *Store) Checkpoint() (domain.RecordID, time.Time) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	return s.lastIndexedRecordID, s.lastUpdate
}

func (s *Store) Load(
	manager *index.Manager,
) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if manager == nil {
		return errors.New(
			"snapshot target manager is nil",
		)
	}

	m, err := s.readLatest(manager.FromSnapshots)
	if err != nil {
		return fmt.Errorf(
			"load snapshots: %w",
			err,
		)
	}

	s.lastIndexedRecordID = m.LastIndexedRecordID
	s.lastUpdate = m.CreatedAt

	return nil
}

func (s *Store) readLatest(
	consume func(
		iter.Seq2[index.Snapshot, error],
	) error,
) (manifest.Manifest, error) {
	last, err := findLastSnapshot(s.path)
	if err != nil {
		return manifest.Manifest{}, fmt.Errorf(
			"find last snapshot: %w",
			err,
		)
	}

	if last == 0 {
		return manifest.Manifest{}, ErrNoSnapshots
	}

	dir := filepath.Join(
		s.path,
		fmt.Sprintf("snapshot-%04d", last),
	)

	root, err := os.OpenRoot(dir)
	if err != nil {
		return manifest.Manifest{}, fmt.Errorf(
			"open snapshot generation %d: %w",
			last,
			err,
		)
	}
	defer func() {
		_ = root.Close()
	}()

	data, err := readRegularFile(
		root,
		"manifest.json",
		-1,
	)
	if err != nil {
		return manifest.Manifest{}, fmt.Errorf(
			"read snapshot manifest: %w",
			err,
		)
	}

	m, err := decodeManifest(data, last)
	if err != nil {
		return manifest.Manifest{}, fmt.Errorf(
			"decode snapshot manifest: %w",
			err,
		)
	}

	seq := func(yield func(index.Snapshot, error) bool) {
		for _, entry := range m.Indexes {
			data, err = readRegularFile(
				root,
				entry.filename,
				int64(entry.size),
			)

			if err != nil {
				yield(index.Snapshot{}, fmt.Errorf(
					"read index %q: %w",
					entry.name,
					err,
				))
				return
			}

			digest := sha256.Sum256(data)
			if hex.EncodeToString(digest[:]) != entry.sha256 {
				err = errors.New(
					"snapshot checksum mismatch",
				)
			}

			if !yield(index.Snapshot{
				Name: entry.name,
				Data: data,
			}, nil) {
				return
			}
		}
	}
	if err = consume(seq); err != nil {
		return manifest.Manifest{}, err
	}
	return m, nil
}

func decodeManifest(
	data []byte,
	generation uint32,
) (manifest.Manifest, error) {
	var m manifest.Manifest

	if !utf8.Valid(data) {
		return m, errors.New(
			"manifest is not valid UTF-8",
		)
	}

	decoder := json.NewDecoder(
		bytes.NewReader(data),
	)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(
		&m,
	); err != nil {
		return m, err
	}

	if err := decoder.Decode(
		&struct{}{},
	); !errors.Is(err, io.EOF) {
		return m, errors.New(
			"trailing data after snapshot manifest",
		)
	}

	if err := requireFields(
		data,
		"version",
		"generation",
		"last_indexed_record_id",
		"created_at",
		"indexes",
	); err != nil {
		return m, err
	}

	if m.Version != manifestVersion {
		return m, fmt.Errorf(
			"unsupported manifest version %d",
			m.Version,
		)
	}

	if m.Generation != generation {
		return m, fmt.Errorf(
			"manifest generation %d does not match directory %d",
			m.Generation,
			generation,
		)
	}

	if m.CreatedAt.IsZero() ||
		m.Indexes == nil {
		return m, errors.New(
			"missing snapshot time or index list",
		)
	}

	var raw struct {
		Indexes []json.RawMessage `json:"indexes"`
	}

	if err := json.Unmarshal(
		data,
		&raw,
	); err != nil {
		return m, fmt.Errorf(
			"json marshal raw: %w",
			err,
		)
	}

	names := make(map[string]struct{}, len(m.Indexes))
	files := make(map[string]struct{}, len(m.Indexes))

	for i, entry := range m.Indexes {
		if err := requireFields(
			raw.Indexes[i],
			"name",
			"filename",
			"size",
			"sha256",
		); err != nil {
			return m, err
		}

		if entry.name == "" ||
			!utf8.ValidString(entry.name) {
			return m, errors.New(
				"invalid snapshot index name",
			)
		}

		if _, ok := names[entry.name]; ok {
			return m, fmt.Errorf(
				"duplicate snapshot index %q",
				entry.name,
			)
		}
		names[entry.name] = struct{}{}

		ordinal, err := strconv.Atoi(
			strings.TrimSuffix(
				strings.TrimPrefix(
					entry.filename,
					"index-",
				),
				".bin"),
		)

		if err != nil ||
			ordinal < 1 ||
			ordinal > len(m.Indexes) ||
			entry.filename != fmt.Sprintf(
				"index-%04d.bin",
				ordinal,
			) {
			return m, fmt.Errorf(
				"invalid index filename %q",
				entry.filename,
			)
		}

		if _, ok := files[entry.filename]; ok {
			return m, fmt.Errorf(
				"duplicate snapshot filename %q",
				entry.filename,
			)
		}
		files[entry.filename] = struct{}{}

		if entry.size < 0 {
			return m, fmt.Errorf(
				"negative index size for %q",
				entry.name,
			)
		}

		digest, err := hex.DecodeString(entry.sha256)
		if err != nil ||
			len(digest) != sha256.Size ||
			entry.sha256 != strings.ToLower(entry.sha256) {
			return m, fmt.Errorf(
				"invalid checksum for %q",
				entry.name,
			)
		}
	}

	return m, nil
}

func requireFields(
	data []byte,
	names ...string,
) error {
	decoder := json.NewDecoder(bytes.NewReader(data))

	token, err := decoder.Token()
	if err != nil ||
		token != json.Delim('{') {
		return errors.New(
			"manifest entry must be an object",
		)
	}

	seen := make(map[string]bool, len(names))
	for decoder.More() {
		token, err = decoder.Token()
		if err != nil {
			return fmt.Errorf(
				"decoder get token: %w",
				err,
			)
		}

		name, ok := token.(string)
		if !ok {
			return errors.New(
				"invalid manifest field name",
			)
		}

		if _, duplicate := seen[name]; duplicate {
			return fmt.Errorf(
				"duplicate manifest field %q",
				name,
			)
		}

		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return fmt.Errorf(
				"decode value: %w",
				err,
			)
		}
		seen[name] = !bytes.Equal(bytes.TrimSpace(value), []byte("null"))
	}

	if _, err = decoder.Token(); err != nil {
		return fmt.Errorf(
			"decoder last token: %w",
			err,
		)
	}

	for _, name := range names {
		if !seen[name] {
			return fmt.Errorf(
				"missing manifest field %q",
				name,
			)
		}

		delete(seen, name)
	}

	for name := range seen {
		return fmt.Errorf(
			"unexpected manifest field %q",
			name,
		)
	}

	return nil
}

func readRegularFile(
	root *os.Root,
	name string,
	expectedSize int64,
) (data []byte, err error) {
	info, err := root.Lstat(name)
	if err != nil {
		return nil, fmt.Errorf(
			"root stat: %w",
			err,
		)
	}

	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf(
			"%q is not a regular file",
			name,
		)
	}

	file, err := root.Open(name)
	if err != nil {
		return nil, fmt.Errorf(
			"open file from root: %w",
			err,
		)
	}
	defer func() {
		_ = file.Close()
	}()

	opened, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf(
			"file %q stat: %w",
			file.Name(),
			err,
		)
	}

	if !opened.Mode().IsRegular() ||
		!os.SameFile(info, opened) {
		return nil, fmt.Errorf(
			"snapshot file %q changed while opening",
			name,
		)
	}

	if expectedSize >= 0 &&
		opened.Size() != expectedSize {
		return nil, fmt.Errorf(
			"snapshot size mismatch for %q: got %d, want %d",
			name,
			opened.Size(),
			expectedSize,
		)
	}

	data, err = io.ReadAll(
		io.NewSectionReader(
			file,
			0,
			opened.Size(),
		),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"read all: %w",
			err,
		)
	}

	after, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if int64(len(data)) != opened.Size() ||
		after.Size() != opened.Size() {
		return nil, fmt.Errorf(
			"snapshot size changed while reading %q",
			name,
		)
	}
	return data, nil
}
