package index

import (
	"fishyAHP/LogParser.git/internal/features/index/timestamp"
)

type PID = uint32
type LogComponent = string

type LogLevel string

const (
	Debug LogLevel = "DEBUG"
	Info  LogLevel = "INFO"
	Warn  LogLevel = "WARN"
	Error LogLevel = "ERROR"
	Fatal LogLevel = "FATAL"
)

type Indexer struct {
	level     *Index[LogLevel]
	component *Index[LogComponent]
	pid       *Index[PID]
	timestamp *timestamp.Index
}
