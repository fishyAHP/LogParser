package storage

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/storage/segments"
)

type Iterator interface {
	Next() bool

	Data() []byte
	RecordID() domain.RecordID
	Pointer() domain.RecordPointer

	Err() error
	Close() error
}

type storageIterator struct {
	dir string

	segmentID uint64
	offset    uint64
	file      *os.File

	header  [segments.RecordHeaderSize]byte
	data    []byte
	id      domain.RecordID
	pointer domain.RecordPointer

	err  error
	done bool
}

func (i *storageIterator) Next() bool {
	if i.done || i.err != nil {
		return false
	}

	var err error
	dir := i.dir
	curOffset := i.offset
	curID := i.segmentID

	if i.file == nil {
		curID = 1
		file, err := openFile(dir, curID)
		if err != nil {
			i.err = err
			if errors.Is(err, os.ErrNotExist) {
				i.err = nil
				i.done = true
			}
			return false
		}

		i.file = file
		curOffset = uint64(len(segments.Magic))
	}

	for {
		if n, err := i.file.ReadAt(i.header[:], int64(curOffset)); err != nil {
			if errors.Is(err, io.EOF) && n == 0 {
				err = i.file.Close()
				i.file = nil
				if err != nil {
					i.err = err
					return false
				}
				curID++

				file, err := openFile(dir, curID)
				if err != nil {
					i.err = err
					if errors.Is(err, os.ErrNotExist) {
						i.err = nil
						i.done = true
					}
					return false
				}

				curOffset = uint64(len(segments.Magic))
				i.file = file
				continue
			}
			i.err = err
			return false
		}
		break
	}

	length := binary.BigEndian.Uint32(i.header[0:4])
	if segments.FileSize(length)+segments.RecordHeaderSize >= segments.MaxSegmentSize {
		i.err = ErrRecordTooLarge
		return false
	}

	recordID := binary.BigEndian.Uint64(i.header[4:])

	if cap(i.data) < int(length) {
		i.data = slices.Grow(i.data, int(length)-len(i.data))
	}
	i.data = i.data[:int(length)]

	curOffset += uint64(segments.RecordHeaderSize)
	if _, err = i.file.ReadAt(
		i.data,
		int64(curOffset),
	); err != nil {
		i.err = err
		return false
	}

	i.segmentID = curID
	i.offset = curOffset + uint64(length)

	i.id = domain.RecordID(recordID)
	i.pointer = domain.RecordPointer{
		Offset:    curOffset,
		Length:    length,
		SegmentID: curID,
	}

	return true
}

func openFile(dir string, curID uint64) (*os.File, error) {
	path := filepath.Join(
		dir,
		fmt.Sprintf(
			"segment-%04d",
			curID,
		),
	)
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf(
			"open file: %w",
			err,
		)
	}

	err = segments.ValidateFileHeader(file)
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf(
			"validate header: %w",
			err,
		)
	}

	return file, nil
}

func (i *storageIterator) Data() []byte {
	return i.data
}

func (i *storageIterator) RecordID() domain.RecordID {
	return i.id
}

func (i *storageIterator) Pointer() domain.RecordPointer {
	return i.pointer
}

func (i *storageIterator) Err() error {
	if i.err != nil {
		return i.err
	}
	return nil
}

func (i *storageIterator) Close() error {
	if i.file == nil {
		return nil
	}

	err := i.file.Close()
	i.file = nil
	i.done = true

	if err != nil {
		return fmt.Errorf(
			"close file: %w",
			err,
		)
	}
	return nil
}
