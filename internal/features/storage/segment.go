package features_storage

import (
	"fmt"
	"os"

	"fishyAHP/LogParser.git/internal/core/domain"
	"github.com/google/uuid"
)

type fileSize uint64

const (
	Byte           fileSize = 1
	KByte                   = 1024 * Byte
	MByte                   = 1024 * KByte
	MaxSegmentSize          = 100 * MByte
)

func (f fileSize) String() string {
	switch {
	case f <= MaxSegmentSize:
		return fmt.Sprintf("%.2fmb", float64(f)/float64(MByte))
	case f < MByte:
		return fmt.Sprintf("%.2fkb", float64(f)/float64(KByte))
	case f < KByte:
		return fmt.Sprintf("%.2fb", float64(f)/float64(Byte))
	default:
		return fmt.Sprintf("%db", f)
	}
}

// Segment представляет собой файл, в который сейчас происходит запись
type Segment struct {
	ID   uint32
	Size fileSize
	File *os.File
}

// newSegment создает/открывает файл, дает ему номер/название,
// если его не было.
// Также определяет текущий размер, если он больше максимального, то необходимо создать новый
func newSegment(path string, id uint32) (*Segment, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o644)
	if err != nil {
		return &Segment{}, fmt.Errorf("open file: %w", err)
	}

	defer func() {
		if errClose := file.Close(); errClose != nil {
			// когда сделаем свой логгер то заменим простое возвращение на запись лога
			return
		}
	}()

	stat, err := file.Stat()
	if err != nil {
		return &Segment{}, fmt.Errorf("stat file: %w", err)
	}

	return &Segment{
		ID:   id,
		Size: fileSize(stat.Size()),
		File: file,
	}, nil
}

func (s *Segment) Write(data []byte) (domain.RecordData, error) {
	if _, err := s.File.Write(data); err != nil {
		return domain.RecordData{}, fmt.Errorf("write file: %w", err)
	}

	rm := domain.NewRecordData(
		uint32(len(data)),
		s.ID,
		uint64(s.Size),
		uuid.New(),
	)

	s.Size += fileSize(rm.Pointer.Length)

	return rm, nil
}

func (s *Segment) Read(pointer domain.RecordData) (data []byte, err error) {
	return nil, nil
}

func (s *Segment) Close() error {
	return nil
}
