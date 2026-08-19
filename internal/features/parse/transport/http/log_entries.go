package parse_transport_http

import "time"

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Fields    map[string]string
}

func (l *LogEntry) JSONUnmarshal(data []byte) (n int, err error) {
	return 0, nil
}
