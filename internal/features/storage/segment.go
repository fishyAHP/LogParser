package storage

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"fishyAHP/LogParser.git/internal/core/domain"
	"github.com/google/uuid"
)

type fileSize uint64

const (
	Byte           fileSize = 1
	KByte                   = 1024 * Byte
	MByte                   = 1024 * KByte
	MaxSegmentSize          = 100 * MByte

	loadFactor = 87.5 / 100
)

func (f fileSize) String() string {
	switch {
	case f >= MByte:
		return fmt.Sprintf("%.2fmb", float64(f)/float64(MByte))
	case f >= KByte:
		return fmt.Sprintf("%.2fkb", float64(f)/float64(KByte))
	default:
		return fmt.Sprintf("%db", f)
	}
}

// segment представляет собой файл, в который сейчас происходит запись
type segment struct {
	ID   uint32
	Size fileSize
	File *os.File
}

// newSegment создает/открывает файл, дает ему номер/название,
// если его не было.
// Также определяет его текущий размер.
func newSegment(dir string, id uint32) (*segment, error) {
	// filepath.Join конкатенирует несколько строк в файловый путь.
	// После того как сделаем интеграцию парсера и хранилища уберем эту обработку туда
	path := filepath.Join(
		dir,
		fmt.Sprintf("segment-%04d", id),
	)

	// os.OpenFile открывает нужный файл или создает если его не было,
	// файл открывается для чтения и записи, в конце битовой маски флаг,
	// что указатель записи в файл ставить в его конец.
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o644)
	if err != nil {
		return &segment{}, fmt.Errorf("open file: %w", err)
	}

	defer func() {
		if err != nil {
			errClose := file.Close()
			if errClose != nil {
				// когда сделаем логгер то вставим его сюда
				return
			}
		}
	}()

	stat, err := file.Stat()
	if err != nil {
		return &segment{}, fmt.Errorf("stat file: %w", err)
	}

	return &segment{
		ID:   id,
		Size: fileSize(stat.Size()),
		File: file,
	}, nil
}

func (s *segment) isOverloaded(size fileSize) bool {
	return float64(s.Size+size)/float64(MaxSegmentSize) >= loadFactor
}

func (s *segment) Write(data []byte) (*domain.RecordData, error) {
	n, err := s.File.Write(data)
	if err != nil {
		return &domain.RecordData{},
			fmt.Errorf("write segment file: %w", err)
	}

	rd := domain.NewRecordData(
		uint32(n),
		s.ID,
		uint64(s.Size),
		s.File.Name(),
		uuid.New(),
	)
	s.Size += fileSize(n)

	return rd, nil
}

func (s *segment) Read(rd *domain.RecordData) (data []byte, err error) {
	if s.ID != rd.Pointer.SegmentID {
		return nil, errors.New("segment read: not suitable record data")
	}

	if _, err = s.File.Seek(
		int64(rd.Pointer.Offset),
		io.SeekStart,
	); err != nil {
		return nil, fmt.Errorf("seek segment file: %w", err)
	}

	data = make([]byte, rd.Pointer.Length)
	if _, err = io.ReadFull(s.File, data); err != nil {
		return nil, fmt.Errorf("read segment file: %w", err)
	}

	return
}

func (s *segment) Close() error {
	if err := s.File.Close(); err != nil {
		return fmt.Errorf("close segment file: %w", err)
	}

	s.ID = 0
	return nil
}

func (s *segment) IsOpen() bool {
	return s.ID != 0
}
