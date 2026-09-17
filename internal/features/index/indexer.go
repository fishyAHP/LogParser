package index

import (
	"strings"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/timestamp"
)

type Indexer struct {
	level     *Index[domain.LogLevel]
	component *Index[domain.LogComponent]
	pid       *Index[domain.PID]
	timestamp *timestamp.Index
}

func New() *Indexer {
	return &Indexer{
		level:     NewIndex[domain.LogLevel](),
		component: NewIndex[domain.LogComponent](),
		pid:       NewIndex[domain.PID](),
		timestamp: timestamp.New(),
	}
}

func (i *Indexer) Index(record domain.RecordData, entry domain.LogEntry) {
	if strings.TrimSpace(string(entry.Level)) != "" {
		i.level.Add(entry.Level, record)
	}

	if strings.TrimSpace(entry.Component) != "" {
		i.component.Add(entry.Component, record)
	}

	if entry.PID != 0 {
		i.pid.Add(entry.PID, record)
	}

	if err := i.timestamp.Add(entry.Timestamp, record); err != nil {
		// залогируем
	}
}
