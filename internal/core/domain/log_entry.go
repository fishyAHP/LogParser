package domain

import "time"

// LogEntry логическое представление записи лога
type LogEntry struct {
	Timestamp time.Time
	Level     LogLevel
	Component LogComponent
	PID       PID
	Message   string
	Other     map[string]string
}

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
