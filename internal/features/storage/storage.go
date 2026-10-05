package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"fishyAHP/LogParser.git/internal/core/domain"
)

// Storage это описание системы управления сегментами, или файлами,
// в которые записываются логи или откуда они читаются.
type Storage struct {
	dir          string
	recordsCount uint64
	records      map[domain.RecordID]domain.RecordPointer

	readSegment  *segment
	writeSegment *segment

	readMtx  sync.Mutex
	writeMtx sync.Mutex

	closed bool
}

var (
	ErrClosedStorage  = errors.New("closed storage")
	ErrRecordNotFound = errors.New("record not found")
)

func New(path string) (*Storage, error) {
	err := os.MkdirAll(path, 0o755)
	if err != nil {
		return nil, fmt.Errorf("mk dir all: %w", err)
	}

	// os.ReadDir возвращает отсортированный слайс директорий или файлов.
	// В нашем случае в качестве аргумента передается путь формата: 2006-01-02/15
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	id := 1
	if len(entries) != 0 {
		// Последний элемент должен иметь название с самым большим индексом,
		// так как слайс был отсортирован по названиям
		last := entries[len(entries)-1]
		id, err = strconv.Atoi(last.Name()[8:])

		if err != nil {
			return nil, fmt.Errorf("conversion string to int(atoi): %w", err)
		}
	}

	writer, err := newSegment(path, uint64(id))
	if err != nil {
		return nil, fmt.Errorf("create new writer: %w", err)
	}

	if writer.isOverloaded(0) {
		if err = writer.Close(); err != nil {
			return nil, fmt.Errorf("close old writer: %w", err)
		}

		if writer, err = newSegment(path, uint64(id+1)); err != nil {
			return nil, fmt.Errorf("create not overloaded writer: %w", err)
		}
	}

	return &Storage{
		dir:          path,
		records:      make(map[domain.RecordID]domain.RecordPointer),
		writeSegment: writer,
		readSegment:  &segment{},
	}, nil
}

func (s *Storage) Write(data []byte) (domain.RecordID, error) {
	s.writeMtx.Lock()
	defer s.writeMtx.Unlock()

	if s.closed {
		return 0, ErrClosedStorage
	}

	if s.writeSegment.isOverloaded(FileSize(len(data))) {
		if err := s.rotationSegment(); err != nil {
			return 0, fmt.Errorf("write segment: %w", err)
		}
	}

	pointer, err := s.writeSegment.Write(data, s.recordsCount+1)
	if err != nil {
		return 0, fmt.Errorf("write segment: %w", err)
	}

	s.recordsCount++
	s.records[domain.RecordID(s.recordsCount)] = pointer

	return domain.RecordID(s.recordsCount), nil
}

func (s *Storage) ReadByID(id domain.RecordID) ([]byte, error) {
	pointer, ok := s.records[id]
	if !ok {
		return nil, ErrRecordNotFound
	}

	return s.read(pointer)
}

func (s *Storage) read(pointer domain.RecordPointer) (data []byte, err error) {
	s.readMtx.Lock()
	defer s.readMtx.Unlock()
	if s.closed {
		return nil, ErrClosedStorage
	}

	if err = s.openPointer(pointer); err != nil {
		return nil, fmt.Errorf("read storage: %w", err)
	}

	data, err = s.readSegment.Read(pointer)
	if err != nil {
		return nil, fmt.Errorf("read storage: %w", err)
	}

	return
}

func (s *Storage) Close() error {
	s.readMtx.Lock()
	defer s.readMtx.Unlock()

	if s.readSegment.IsOpen() {
		if err := s.readSegment.Close(); err != nil {
			return fmt.Errorf("close read segment: %w", err)
		}
	}

	s.writeMtx.Lock()
	defer s.writeMtx.Unlock()

	if s.writeSegment.IsOpen() {
		if err := s.writeSegment.Close(); err != nil {
			return fmt.Errorf("close write segment: %w", err)
		}
	}

	s.recordsCount = 0
	s.closed = true

	return nil
}

func (s *Storage) openPointer(pointer domain.RecordPointer) error {
	if s.readSegment.IsOpen() {
		if filepath.Join(s.dir, strconv.Itoa(int(s.readSegment.ID))) ==
			filepath.Join(pointer.Path, strconv.Itoa(int(pointer.SegmentID))) {
			return nil
		}

		if err := s.readSegment.Close(); err != nil {
			return fmt.Errorf("open pointer: %w", err)
		}
	}

	newReader, err := newSegment(
		pointer.Path,
		pointer.SegmentID,
	)
	if err != nil {
		return fmt.Errorf("open pointer: %w", err)
	}

	s.readSegment = newReader
	return nil
}

func (s *Storage) rotationSegment() error {
	oldID := s.writeSegment.ID
	if err := s.writeSegment.Close(); err != nil {
		return fmt.Errorf("rotation segment: %w", err)
	}

	newWriter, err := newSegment(
		s.dir,
		oldID+1,
	)
	// TODO если тут будет ошибка, то у нас останется только закрытый сегмент для записи
	if err != nil {
		return fmt.Errorf("rotation segment: %w", err)
	}

	s.writeSegment = newWriter
	return nil
}

func (s *Storage) Iterator() *Iterator {
	return &Iterator{
		storage: s,
	}
}
