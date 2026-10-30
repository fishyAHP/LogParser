package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"fishyAHP/LogParser.git/internal/features/metadata/snapshot/manifest"
	"fishyAHP/LogParser.git/internal/features/storage/segments"
)

type writer struct {
	tmpDir   string
	finalDir string
	m        manifest.Manifest
}

func (w *writer) writeIndex(
	name string,
	index []byte,
) error {
	filename := fmt.Sprintf(
		"%d.bin",
		w.m.IndexCount()+1,
	)
	path := filepath.Join(
		w.tmpDir,
		filename,
	)

	if err := writeSyncedFile(
		path,
		index,
	); err != nil {
		return err
	}

	digest := sha256.Sum256(index)
	checkSum := hex.EncodeToString(digest[:])

	if err := w.m.AddIndex(
		name,
		filename,
		segments.FileSize(len(index)),
		checkSum,
	); err != nil {
		return fmt.Errorf(
			"add index to manifest: %w",
			err,
		)
	}

	return nil
}

func (w *writer) writeManifest() error {
	data, err := w.m.MarshalBinary()
	if err != nil {
		return fmt.Errorf(
			"manifest binary marshal: %w",
			err,
		)
	}

	path := filepath.Join(w.tmpDir, "manifest.bin")

	return writeSyncedFile(path, data)
}

func (w *writer) publish() error {
	if err := syncDirectory(w.tmpDir); err != nil {
		return fmt.Errorf(
			"sync temporary snapshot dir: %w",
			err,
		)
	}

	if err := os.Rename(
		w.tmpDir,
		w.finalDir,
	); err != nil {
		return fmt.Errorf(
			"rename dir from %q to %q: %w",
			w.tmpDir,
			w.finalDir,
			err,
		)
	}

	if err := syncDirectory(
		filepath.Dir(w.finalDir),
	); err != nil {
		return fmt.Errorf(
			"sync snapshot directory: %w",
			err,
		)
	}

	return nil
}

func writeSyncedFile(
	path string,
	data []byte,
) (err error) {
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
		err = errors.Join(err, file.Close())
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
	dir, err := os.Open(path)
	if err != nil {
		return fmt.Errorf(
			"open directory %q: %w",
			path,
			err,
		)
	}
	defer func() {
		err = errors.Join(err, dir.Close())
	}()

	if err = dir.Sync(); err != nil {
		return fmt.Errorf(
			"sync directory %q: %w",
			path,
			err,
		)
	}

	return nil
}
