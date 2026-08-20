package domain

import "github.com/google/uuid"

// RecordPointer физическое местоположение записи
type RecordPointer struct {
	Offset    uint64
	Length    uint32
	SegmentID uint32
}

// RecordData объединение физического и логического местоположения записи
type RecordData struct {
	ID      uuid.UUID
	Pointer RecordPointer
}

func NewRecordData(
	length uint32,
	segmentID uint32,
	offset uint64,
	recordID uuid.UUID,
) *RecordData {
	return &RecordData{
		ID: recordID,
		Pointer: RecordPointer{
			Offset:    offset,
			Length:    length,
			SegmentID: segmentID,
		},
	}
}
