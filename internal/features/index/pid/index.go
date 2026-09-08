package pid

import "fishyAHP/LogParser.git/internal/core/domain"

type PID = uint32

type Index struct {
	index map[PID][]domain.RecordData
}
