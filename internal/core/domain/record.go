package domain

// RecordPointer физическое местоположение записи
type RecordPointer struct {
	Offset    uint64
	Length    uint32
	SegmentID uint64
}

type RecordID uint64
