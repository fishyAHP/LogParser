package index

import (
	"strings"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/hash_index"
	"fishyAHP/LogParser.git/internal/features/index/timestamp"
)

type Indexer struct {
	level     Index[domain.LogLevel]
	component Index[domain.LogComponent]
	pid       Index[domain.PID]
	ip        Index[domain.IP]
	timestamp *timestamp.Index
}

func New(timeAccuracy time.Duration) *Indexer {
	return &Indexer{
		level:     hash_index.NewIndex[domain.LogLevel](),
		component: hash_index.NewIndex[domain.LogComponent](),
		pid:       hash_index.NewIndex[domain.PID](),
		ip:        hash_index.NewIndex[domain.IP](),
		timestamp: timestamp.New(timeAccuracy),
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

	if !entry.IP.IsZero() {
		i.ip.Add(entry.IP, record)
	}

	if err := i.timestamp.Add(entry.Timestamp, record); err != nil {
		// залогируем
	}
}

func (i *Indexer) Clear() {
	i.level.Clear()
	i.component.Clear()
	i.pid.Clear()
	i.ip.Clear()
	i.timestamp.Clear()
}
