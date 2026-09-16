package index

import (
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
	i.level.Add(entry.Level, record)
	i.component.Add(entry.Component, record)
	i.pid.Add(entry.PID, record)

	if err := i.timestamp.Add(entry.Timestamp, record); err != nil {
		// залогируем
	}
}
