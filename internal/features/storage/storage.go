package features_storage

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
)

// Storage это директория в которую сейчас ведется запись
type Storage struct {
	dir           string
	activeSegment *Segment
}

// New создает или открывает директорию, затем находит последний сегмент.
// После этого определяет его как активный, в случае если он не заполнен слишком сильно.
// Если сегментов нет, то создает новый.
// Если размер последнего сегмента больше максимального заданного значения, то надо создать новый.
func New(path string) (*Storage, error) {
	if _, err := time.Parse("2006-01-02/15", path); err != nil {
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
	// В нашем случае в качестве аргумента передается полный путь формата: 2006-01-02/15
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

	segment, err := newSegment(path, uint32(id))
	if err != nil {
		return nil, fmt.Errorf("create new segment: %w", err)
	}

	if segment.isOverloaded(0) {
		if err = segment.Close(); err != nil {
			return nil, fmt.Errorf("close old segment: %w", err)
		}

		if segment, err = newSegment(path, uint32(id+1)); err != nil {
			return nil, fmt.Errorf("create not overloaded segment: %w", err)
		}
	}

	return &Storage{
		dir:           path,
		activeSegment: segment,
	}, nil
}

func (s *Storage) Write(data []byte) (*domain.RecordData, error) {
	if s.activeSegment.isOverloaded(fileSize(len(data))) {
		if err := s.rotationSegment(
			s.dir,
			s.activeSegment.ID+1,
		); err != nil {
			return nil, fmt.Errorf("write segment: %w", err)
		}
	}

	rd, err := s.activeSegment.Write(data)
	if err != nil {
		return nil, fmt.Errorf("write segment: %w", err)
	}

	return rd, nil
}

func (s *Storage) Read(pointer *domain.RecordData) (data []byte, err error) {
	if err = s.openPointer(pointer); err != nil {
		return nil, fmt.Errorf("read storage: %w", err)
	}

	data, err = s.activeSegment.Read(pointer)
	if err != nil {
		return nil, fmt.Errorf("read storage: %w", err)
	}

	return
}

func (s *Storage) Close() error {
	if err := s.activeSegment.Close(); err != nil {
		return fmt.Errorf("close storage: %w", err)
	}

	return nil
}

func (s *Storage) openPointer(pointer *domain.RecordData) error {
	if filepath.Join(s.dir, strconv.Itoa(int(s.activeSegment.ID))) ==
		filepath.Join(pointer.Pointer.Directory, strconv.Itoa(int(pointer.Pointer.SegmentID))) {
		return nil
	}

	newDir := s.dir
	if s.dir != pointer.Pointer.Directory {
		newDir = pointer.Pointer.Directory
	}

	if err := s.rotationSegment(
		newDir,
		pointer.Pointer.SegmentID,
	); err != nil {
		return fmt.Errorf("open pointer: %w", err)
	}

	return nil
}

func (s *Storage) rotationSegment(dir string, id uint32) error {
	if err := s.activeSegment.Close(); err != nil {
		return fmt.Errorf("rotation segment: %w", err)
	}

	newActiveSegment, err := newSegment(
		dir,
		id,
	)

	if err != nil {
		return fmt.Errorf("rotation segment: %w", err)
	}

	s.activeSegment = newActiveSegment
	s.dir = dir
	return nil
}
