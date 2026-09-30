package domain

// RecordPointer физическое местоположение записи
type RecordPointer struct {
	Offset    uint64
	Length    uint32
	SegmentID uint32
	Path      string
}

// RecordData объединение физического и логического местоположения записи
type RecordData struct {
	ID      uint64
	Pointer RecordPointer
}

func NewRecordData(
	length, segmentID uint32,
	offset, recordID uint64,
	path string,
) *RecordData {
	return &RecordData{
		ID: recordID,
		Pointer: RecordPointer{
			Offset:    offset,
			Length:    length,
			SegmentID: segmentID,
			Path:      path,
		},
	}
}
