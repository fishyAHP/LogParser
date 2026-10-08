package storage

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"fishyAHP/LogParser.git/internal/core/domain"
)

// Storage это описание системы управления сегментами, или файлами,
// в которые записываются логи или откуда они читаются.
type Storage struct {
	dir          string
	recordsCount uint64
	records      []domain.RecordPointer

	readSegment  *segment
	writeSegment *segment

	readMtx  sync.Mutex
	writeMtx sync.Mutex

	closed bool
}

var (
	readFlag  = os.O_RDONLY
	writeFlag = os.O_CREATE | os.O_RDWR | os.O_APPEND
)

var (
	ErrRecordTooLarge = errors.New("id too large")
	ErrClosedStorage  = errors.New("closed storage")
	ErrRecordNotFound = errors.New("id not found")
)

func New(path string) (*Storage, error) {
	err := os.MkdirAll(path, 0o755)
	if err != nil {
		return nil, fmt.Errorf(
			"mk dir all: %w",
			err,
		)
	}

	storage := &Storage{
		dir:         path,
		records:     make([]domain.RecordPointer, 0),
		readSegment: &segment{},
	}

	if err = storage.recover(); err != nil {
		return nil, err
	}

	id, err := findLastSegment(path)
	if err != nil {
		return nil, err
	}

	writer, err := newSegment(
		path,
		id,
		writeFlag,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create new writer: %w",
			err,
		)
	}

	if writer.isOverloaded(0) {
		if err = writer.close(); err != nil {
			return nil, fmt.Errorf(
				"close old writer: %w",
				err,
			)
		}

		if writer, err = newSegment(
			path,
			id+1,
			writeFlag,
		); err != nil {
			return nil, fmt.Errorf(
				"create not overloaded writer: %w",
				err,
			)
		}
	}

	storage.writeSegment = writer

	return storage, nil
}

func findLastSegment(dir string) (uint64, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf(
			"read dir: %w",
			err,
		)
	}

	segmentsID := make(map[uint64]struct{})
	var maxID uint64

	for _, entry := range entries {
		name := entry.Name()
		withoutPrefix, ok := strings.CutPrefix(
			name,
			"segment-",
		)
		if !ok {
			continue
		}

		if !entry.Type().IsRegular() {
			return 0, fmt.Errorf(
				"segment %q isn't regular file: %v",
				name,
				entry.Type(),
			)
		}

		id, err := strconv.ParseUint(withoutPrefix, 10, 64)
		if err != nil {
			return 0, fmt.Errorf(
				"corrupted number of segment: %w",
				err,
			)
		}

		if id == 0 {
			return 0, errors.New(
				"segment id cannot be zero",
			)
		}

		expectedName := fmt.Sprintf("segment-%04d", id)

		if entry.Name() != expectedName {
			return 0, fmt.Errorf(
				"invalid segment filename: %s, expected %s",
				entry.Name(),
				expectedName,
			)
		}

		if _, ok = segmentsID[id]; ok {
			return 0, errors.New(
				"segment id must be unique",
			)
		}
		segmentsID[id] = struct{}{}
		maxID = max(maxID, id)
	}

	if maxID == 0 {
		return 1, nil
	}

	for id := range maxID {
		if _, ok := segmentsID[id+1]; !ok {
			return 0, fmt.Errorf(
				"missing segment-%04d",
				id+1,
			)
		}
	}

	return maxID, nil
}

func (s *Storage) recover() error {
	iterator := s.Iterator()
	defer func() {
		_ = iterator.Close()
	}()

	for iterator.Next() {
		id := iterator.RecordID()
		pointer := iterator.Pointer()

		expectID := domain.RecordID(len(s.records) + 1)
		if id != expectID {
			return fmt.Errorf(
				"invalid record sequence: want %d, got %d",
				expectID,
				id,
			)
		}

		s.records = append(s.records, pointer)
		s.recordsCount = uint64(id)
	}

	if err := iterator.Err(); err != nil {
		return fmt.Errorf(
			"iterator: %w",
			err,
		)
	}
	return nil
}

func (s *Storage) Write(data []byte) (domain.RecordID, error) {
	s.writeMtx.Lock()
	defer s.writeMtx.Unlock()

	if s.closed {
		return 0, ErrClosedStorage
	}

	sizeData := FileSize(len(data))
	if sizeData+RecordHeaderSize >= MaxSegmentSize {
		return 0, ErrRecordTooLarge
	}

	if s.writeSegment.isOverloaded(FileSize(len(data))) {
		if err := s.rotationSegment(); err != nil {
			return 0, fmt.Errorf(
				"write segment: %w",
				err,
			)
		}
	}

	pointer, err := s.writeSegment.write(data, s.recordsCount+1)
	if err != nil {
		return 0, fmt.Errorf(
			"write segment: %w",
			err,
		)
	}

	s.recordsCount++
	s.records = append(s.records, pointer)

	return domain.RecordID(s.recordsCount), nil
}

func (s *Storage) ReadByID(id domain.RecordID) ([]byte, error) {
	if 0 >= id ||
		id > domain.RecordID(len(s.records)) {
		return nil, ErrRecordNotFound
	}

	return s.read(s.records[id-1])
}

func (s *Storage) read(pointer domain.RecordPointer) (data []byte, err error) {
	s.readMtx.Lock()
	defer s.readMtx.Unlock()
	if s.closed {
		return nil, ErrClosedStorage
	}

	if err = s.openPointer(pointer); err != nil {
		return nil, fmt.Errorf(
			"open pointer: %w",
			err,
		)
	}

	data, err = s.readSegment.read(pointer)
	if err != nil {
		return nil, fmt.Errorf(
			"read segment: %w",
			err,
		)
	}

	return
}

func (s *Storage) Close() error {
	s.readMtx.Lock()
	defer s.readMtx.Unlock()

	if s.readSegment.isOpen() {
		if err := s.readSegment.close(); err != nil {
			return fmt.Errorf(
				"close read segment: %w",
				err,
			)
		}
	}

	s.writeMtx.Lock()
	defer s.writeMtx.Unlock()

	if s.writeSegment.isOpen() {
		if err := s.writeSegment.close(); err != nil {
			return fmt.Errorf(
				"close write segment: %w",
				err,
			)
		}
	}

	s.recordsCount = 0
	s.closed = true

	return nil
}

func (s *Storage) openPointer(pointer domain.RecordPointer) error {
	if s.readSegment.isOpen() {
		if s.readSegment.ID == pointer.SegmentID {
			return nil
		}

		if err := s.readSegment.close(); err != nil {
			return fmt.Errorf(
				"open pointer: %w",
				err,
			)
		}
	}

	newReader, err := newSegment(
		s.dir,
		pointer.SegmentID,
		readFlag,
	)
	if err != nil {
		return fmt.Errorf(
			"open pointer: %w",
			err,
		)
	}

	s.readSegment = newReader
	return nil
}

func (s *Storage) rotationSegment() error {
	nextID := s.writeSegment.ID + 1
	if err := s.writeSegment.close(); err != nil {
		return fmt.Errorf(
			"rotation segment: %w",
			err,
		)
	}

	newWriter, err := newSegment(
		s.dir,
		nextID,
		writeFlag,
	)
	// TODO если тут будет ошибка, то у нас останется только закрытый сегмент для записи
	if err != nil {
		return fmt.Errorf(
			"rotation segment: %w",
			err,
		)
	}

	s.writeSegment = newWriter
	return nil
}

func (s *Storage) Iterator() *Iterator {
	return &Iterator{
		dir: s.dir,
	}
}
