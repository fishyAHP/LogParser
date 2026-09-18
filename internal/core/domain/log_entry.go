package domain

import "time"

// LogEntry логическое представление записи лога
type LogEntry struct {
	Timestamp time.Time
	Level     LogLevel
	Component LogComponent
	PID       PID
	IP        IP
	Message   string
	Other     map[string]string
}

type IP [4]uint8

func (i IP) IsZero() bool {
	var zeroCount int
	for _, ip := range i {
		if ip == 0 {
			zeroCount++
		}
	}

	return zeroCount == 4
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
