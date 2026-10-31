package segments

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

const (
	RecordLengthSize = 4 * Byte
	RecordIdSize     = 8 * Byte
	RecordHeaderSize = RecordLengthSize + RecordIdSize
)

// Segment представляет собой файл, в котором сейчас происходит запись или чтение
type Segment struct {
	ID   uint64
	size FileSize
	file *os.File
}

var Magic = []byte{'L', 'G', 'P', 'R', 'S', 'R'}

// NewSegment создает/открывает файл, дает ему номер/название,
// если его не было.
// Также определяет его текущий размер.
func NewSegment(
	dir string,
	id uint64,
	flag int,
) (*Segment, error) {
	path := filepath.Join(
		dir,
		fmt.Sprintf("segment-%04d", id),
	)

	file, err := os.OpenFile(
		path,
		flag,
		0o644,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"open segment: %w",
			err,
		)
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
		if err = ValidateFileHeader(
			file,
		); err != nil {
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

	return &Segment{
		ID:   id,
		size: FileSize(stat.Size()),
		file: file,
	}, nil
}

func writeFileHeader(file *os.File) error {
	n, err := file.Write(Magic)
	if err != nil {
		return fmt.Errorf(
			"write magic: %w",
			err,
		)
	}
	if n != len(Magic) {
		return errors.New(
			"corrupted write magic",
		)
	}

	return nil
}

func ValidateFileHeader(file *os.File) error {
	buf := make([]byte, len(Magic))
	n, err := file.ReadAt(buf, 0)
	if err != nil {
		return fmt.Errorf(
			"read magic: %w",
			err,
		)
	}
	if n != len(Magic) {
		return errors.New(
			"corrupted read magic",
		)
	}

	if !bytes.Equal(buf, Magic) {
		return errors.New(
			"invalid file magic",
		)
	}
	return nil
}

func (s *Segment) IsOverloaded(
	size FileSize,
) bool {
	return float64(s.size+size+RecordHeaderSize)/
		float64(MaxSegmentSize) >=
		loadFactor
}

func (s *Segment) Write(
	data []byte,
	recordID uint64,
) (domain.RecordPointer, error) {
	header := make([]byte, RecordHeaderSize)

	binary.BigEndian.PutUint32(
		header[0:4],
		uint32(len(data)),
	)
	binary.BigEndian.PutUint64(
		header[4:12],
		recordID,
	)

	buf := make([]byte, 0, int(RecordHeaderSize)+len(data))
	buf = append(buf, header...)
	buf = append(buf, data...)

	n, err := s.file.Write(buf)
	if err != nil {
		return domain.RecordPointer{},
			fmt.Errorf(
				"write segment file: %w",
				err,
			)
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

func (s *Segment) Read(
	pointer domain.RecordPointer,
) ([]byte, error) {
	if s.ID != pointer.SegmentID {
		return nil, errors.New(
			"not suitable id data",
		)
	}

	data := make([]byte, pointer.Length)
	if _, err := s.file.ReadAt(
		data,
		int64(pointer.Offset),
	); err != nil {
		return nil, fmt.Errorf(
			"read segment file: %w",
			err,
		)
	}

	return data, nil
}

func (s *Segment) Close() error {
	if err := s.file.Close(); err != nil {
		return fmt.Errorf(
			"close segment file: %w",
			err,
		)
	}

	s.ID = 0
	return nil
}

func (s *Segment) IsOpen() bool {
	return s.ID != 0
}
