package storage

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"fishyAHP/LogParser.git/internal/core/domain"
)

type FileSize uint64

const (
	Byte           FileSize = 1
	KByte                   = 1024 * Byte
	MByte                   = 1024 * KByte
	MaxSegmentSize          = 100 * MByte

	loadFactor = 87.5 / 100
)

func (f FileSize) String() string {
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
	ID   uint64
	size FileSize
	file *os.File
}

var Magic = []byte{'L', 'G', 'P', 'R', 'S', 'R'}

// newSegment создает/открывает файл, дает ему номер/название,
// если его не было.
// Также определяет его текущий размер.
func newSegment(dir string, id uint64) (*segment, error) {
	// filepath.Join конкатенирует несколько строк в файловый путь.
	// После того как сделаем интеграцию парсера и хранилища уберем эту обработку туда
	path := filepath.Join(
		dir,
		fmt.Sprintf("segment-%04d", id),
	)

	// os.OpenFile открывает нужный файл или создает если его не было,
	// файл открывается для чтения и записи, в конце битовой маски флаг,
	// что указатель записи в файл ставить в его конец.
	file, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_RDWR|os.O_APPEND,
		0o644,
	)
	if err != nil {
		return nil, fmt.Errorf("open segment: %w", err)
	}

	defer func() {
		if err != nil {
			_ = file.Close()
		}
	}()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf(
			"stat segment: %w",
			err,
		)
	}

	if stat.Size() == 0 {
		if err = writeFileHeader(file); err != nil {
			return nil, fmt.Errorf(
				"write header: %w",
				err,
			)
		}
	} else {
		if err = validateFileHeader(file); err != nil {
			return nil, fmt.Errorf(
				"validate header: %w",
				err,
			)
		}
	}

	stat, err = file.Stat()
	if err != nil {
		return nil, fmt.Errorf(
			"stat after header: %w",
			err,
		)
	}

	return &segment{
		ID:   id,
		size: FileSize(stat.Size()),
		file: file,
	}, nil
}

func writeFileHeader(file *os.File) error {
	n, err := file.Write(Magic)
	if err != nil {
		return fmt.Errorf("write magic: %w", err)
	}
	if n != len(Magic) {
		return errors.New("corrupted write magic")
	}

	return nil
}

func validateFileHeader(file *os.File) error {
	buf := make([]byte, len(Magic))
	n, err := file.ReadAt(buf, 0)
	if err != nil {
		return fmt.Errorf(
			"read magic: %w",
			err,
		)
	}
	if n != len(Magic) {
		return errors.New("corrupted read magic")
	}

	if !bytes.Equal(buf, Magic) {
		return errors.New("invalid file magic")
	}
	return nil
}

func (s *segment) isOverloaded(size FileSize) bool {
	return float64(s.size+size)/float64(MaxSegmentSize) >= loadFactor
}

const (
	RecordLengthSize = 4 * Byte
	RecordIdSize     = 8 * Byte
	RecordHeaderSize = RecordLengthSize + RecordIdSize
)

func (s *segment) write(data []byte, recordID uint64) (domain.RecordPointer, error) {
	header := make([]byte, RecordHeaderSize)

	binary.BigEndian.PutUint32(header[0:4], uint32(len(data)))
	binary.BigEndian.PutUint64(header[4:12], recordID)

	buf := make([]byte, 0, int(RecordHeaderSize)+len(data))
	buf = append(buf, header...)
	buf = append(buf, data...)

	n, err := s.file.Write(buf)
	if err != nil {
		return domain.RecordPointer{},
			fmt.Errorf("write segment file: %w", err)
	}
	if n != len(buf) {
		return domain.RecordPointer{}, io.ErrShortWrite
	}

	pointer := domain.RecordPointer{
		Offset:    uint64(s.size + RecordHeaderSize),
		Length:    uint32(len(data)),
		SegmentID: s.ID,
	}
	s.size += FileSize(n)

	return pointer, nil
}

func (s *segment) read(pointer domain.RecordPointer) ([]byte, error) {
	if s.ID != pointer.SegmentID {
		return nil, errors.New("not suitable record data")
	}

	if _, err := s.file.Seek(
		int64(pointer.Offset),
		io.SeekStart,
	); err != nil {
		return nil, fmt.Errorf(
			"seek segment file: %w",
			err,
		)
	}

	data := make([]byte, pointer.Length)
	if _, err := io.ReadFull(s.file, data); err != nil {
		return nil, fmt.Errorf(
			"read segment file: %w",
			err,
		)
	}

	return data, nil
}

func (s *segment) close() error {
	if err := s.file.Close(); err != nil {
		return fmt.Errorf(
			"close segment file: %w",
			err,
		)
	}

	s.ID = 0
	return nil
}

func (s *segment) isOpen() bool {
	return s.ID != 0
}
