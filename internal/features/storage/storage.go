package features_storage

import (
	"fishyAHP/LogParser.git/internal/core/domain"
)

// Storage это директория в которую сейчас ведется запись
type Storage struct {
	dir           string
	activeSegment *Segment
}

// New создает или открывает директорию, затем находит последний сегмент.
// После этого определяет его как активный в случае если он не заполнен слишком сильно.
// Если сегментов нет, то создает новый.
func New(path string) (*Storage, error) {
	return nil, nil
}

func (s *Storage) Write(data []byte) (domain.RecordData, error) {
	return s.activeSegment.Write(data)
}

func (s *Storage) Read(pointer domain.RecordData) (data []byte, err error) {
	return nil, nil
}

func (s *Storage) Close() error {
	return nil
}

func (s *Storage) rotationSegment() {

}
