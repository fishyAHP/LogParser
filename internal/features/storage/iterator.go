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
	storage *Storage

	segmentID uint64
	offset    uint64
	file      *os.File

	data   []byte
	record domain.RecordID

	err  error
	done bool
}

func (i *Iterator) Next() bool {
	if i.done || i.err != nil {
		return false
	}

	var err error
	dir := i.storage.dir
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
	i.record = domain.RecordID(recordID)

	if recordID > i.storage.recordsCount {
		i.storage.recordsCount = recordID
	}
	i.storage.records[i.record] = domain.RecordPointer{
		Offset:    curOffset,
		Length:    length,
		SegmentID: curID,
		Path:      dir,
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

	return file, nil
}

func (i *Iterator) Data() []byte {
	return i.data
}

func (i *Iterator) Record() domain.RecordID {
	return i.record
}

func (i *Iterator) Err() error {
	if i.err != nil {
		return i.err
	}
	return nil
}
