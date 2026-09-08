package level

import "fishyAHP/LogParser.git/internal/core/domain"

type LogLevel string

const (
	DEBUG LogLevel = "DEBUG"
	INFO           = "INFO"
	WARN           = "WARN"
	ERROR          = "ERROR"
	FATAL          = "FATAL"
)

type Index struct {
	index map[LogLevel][]domain.RecordData
}
