package storage

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"fishyAHP/LogParser.git/internal/core/domain"
)

type Iterator struct {
	dir string

	segmentID uint64
	offset    uint64
	file      *os.File

	data    []byte
	id      domain.RecordID
	pointer domain.RecordPointer

	err  error
	done bool
}

func (i *Iterator) Next() bool {
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
		curOffset = uint64(len(Magic))
	}

	buf := make([]byte, RecordHeaderSize)
	for {
		if n, err := i.file.ReadAt(buf, int64(curOffset)); err != nil {
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

				curOffset = uint64(len(Magic))
				i.file = file
				continue
			}
			i.err = err
			return false
		}
		break
	}

	length := binary.BigEndian.Uint32(buf[0:4])
	if FileSize(length)+RecordHeaderSize >= MaxSegmentSize {
		i.err = ErrRecordTooLarge
		return false
	}

	recordID := binary.BigEndian.Uint64(buf[4:])

	buf = make([]byte, length)

	curOffset += uint64(RecordHeaderSize)
	if _, err = i.file.ReadAt(
		buf,
		int64(curOffset),
	); err != nil {
		i.err = err
		return false
	}

	i.segmentID = curID
	i.offset = curOffset + uint64(length)

	i.data = buf
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

	err = validateFileHeader(file)
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf(
			"validate header: %w",
			err,
		)
	}

	return file, nil
}

func (i *Iterator) Data() []byte {
	return i.data
}

func (i *Iterator) RecordID() domain.RecordID {
	return i.id
}

func (i *Iterator) Pointer() domain.RecordPointer {
	return i.pointer
}

func (i *Iterator) Err() error {
	if i.err != nil {
		return i.err
	}
	return nil
}

func (i *Iterator) Close() error {
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
