package domain

import "time"

// LogEntry логическое представление записи лога
type LogEntry struct {
	Timestamp time.Time
	Message   string
	Fields    map[string]string
}
