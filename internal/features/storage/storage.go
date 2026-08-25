package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
)

// Storage это описание системы управления сегментами, или файлами,
// в которые записываются логи или откуда они читаются.
type Storage struct {
	dir string

	readSegment  *segment
	writeSegment *segment

	readMtx  sync.Mutex
	writeMtx sync.Mutex
}

// New создает или открывает директорию, затем находит последний сегмент.
// После этого определяет его как активный, в случае если он не заполнен слишком сильно.
// Если сегментов нет, то создает новый.
// Если размер последнего сегмента больше максимального заданного значения, то надо создать новый.
func New(path string) (*Storage, error) {
	if _, err := time.Parse(
		"storage/logs/2006-01-02/15",
		path[len(path)-len("storage/logs/2006-01-02/15"):],
	); err != nil {
		return nil, fmt.Errorf("time parse path: %w", err)
	}

	// os.MkdirAll открывает нужную директорию или создает все директории на указанном пути,
	// если их не было. Права дает создателю все возможности, а остальным возможность читать
	// и выполнять
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

	writer, err := newSegment(path, uint32(id))
	if err != nil {
		return nil, fmt.Errorf("create new writer: %w", err)
	}

	if writer.isOverloaded(0) {
		if err = writer.Close(); err != nil {
			return nil, fmt.Errorf("close old writer: %w", err)
		}

		if writer, err = newSegment(path, uint32(id+1)); err != nil {
			return nil, fmt.Errorf("create not overloaded writer: %w", err)
		}
	}

	return &Storage{
		dir:          path,
		writeSegment: writer,
		readSegment:  &segment{},
	}, nil
}

func (s *Storage) Write(data []byte) (*domain.RecordData, error) {
	s.writeMtx.Lock()
	defer s.writeMtx.Unlock()

	if s.writeSegment.isOverloaded(FileSize(len(data))) {
		if err := s.rotationSegment(); err != nil {
			return nil, fmt.Errorf("write segment: %w", err)
		}
	}

	rd, err := s.writeSegment.Write(data)
	if err != nil {
		return nil, fmt.Errorf("write segment: %w", err)
	}

	return rd, nil
}

func (s *Storage) Read(pointer *domain.RecordData) (data []byte, err error) {
	s.readMtx.Lock()
	defer s.readMtx.Unlock()

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

	return nil
}

func (s *Storage) openPointer(pointer *domain.RecordData) error {
	if s.readSegment.IsOpen() {
		if filepath.Join(s.dir, strconv.Itoa(int(s.readSegment.ID))) ==
			filepath.Join(pointer.Pointer.Path, strconv.Itoa(int(pointer.Pointer.SegmentID))) {
			return nil
		}

		if err := s.readSegment.Close(); err != nil {
			return fmt.Errorf("open pointer: %w", err)
		}
	}

	newReader, err := newSegment(
		pointer.Pointer.Path,
		pointer.Pointer.SegmentID,
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
	// Узкое место, если тут будет ошибка, то у нас останется только закрытый сегмент для записи
	if err != nil {
		return fmt.Errorf("rotation segment: %w", err)
	}

	s.writeSegment = newWriter
	return nil
}
