package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"iter"
	"os"
	"path/filepath"

	"fishyAHP/LogParser.git/internal/features/index"
	"fishyAHP/LogParser.git/internal/features/metadata/snapshot/manifest"
)

type reader struct {
	dir string
	m   manifest.Manifest
}

func (r *reader) readManifest() error {
	path := filepath.Join(
		r.dir,
		"manifest.bin",
	)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf(
			"read manifest %s: %w",
			r.dir,
			err,
		)
	}

	if err = r.m.UnmarshalBinary(data); err != nil {
		return fmt.Errorf(
			"manifest binary unmarshal: %w",
			err,
		)
	}

	return nil
}

func (r *reader) readIndex(
	info manifest.IndexInfo,
) ([]byte, error) {
	if info.Filename == "" ||
		info.Filename == "." ||
		info.Filename == ".." ||
		filepath.Base(info.Filename) != info.Filename {
		return nil, fmt.Errorf(
			"invalid index filename %q",
			info.Filename,
		)
	}

	path := filepath.Join(
		r.dir,
		info.Filename,
	)

	stat, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf(
			"get file statistics: %w",
			err,
		)
	}

	if stat.Size() < 0 ||
		uint64(stat.Size()) != uint64(info.Size) {
		return nil, fmt.Errorf(
			"index %q size mismatch: got %d, want %d",
			info.Name,
			stat.Size(),
			info.Size,
		)
	}

	if !stat.Mode().IsRegular() {
		return nil, fmt.Errorf(
			"index %q is not a regular file",
			info.Name,
		)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf(
			"read index %q: %w",
			info.Name,
			err,
		)
	}

	if uint64(len(data)) != uint64(info.Size) {
		return nil, fmt.Errorf(
			"index %q size mismatch: got %d, want %d",
			info.Name,
			len(data),
			info.Size,
		)
	}

	digest := sha256.Sum256(data)
	actualSHA := hex.EncodeToString(digest[:])

	if actualSHA != info.SHA256 {
		return nil, fmt.Errorf(
			"index %q checksum mismatch",
			info.Name,
		)
	}

	return data, nil
}

func (r *reader) snapshots() iter.Seq2[index.Snapshot, error] {
	return func(yield func(index.Snapshot, error) bool) {
		for info := range r.m.Indexes() {
			data, err := r.readIndex(info)
			if err != nil {
				yield(index.Snapshot{}, err)
				return
			}

			if !yield(index.Snapshot{
				Name: info.Name,
				Data: data,
			}, nil) {
				return
			}
		}
	}
}
