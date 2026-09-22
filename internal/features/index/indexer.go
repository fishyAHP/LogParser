package index

import (
	"strings"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/hash_index"
	"fishyAHP/LogParser.git/internal/features/index/set"
	"fishyAHP/LogParser.git/internal/features/index/text"
	"fishyAHP/LogParser.git/internal/features/index/text/words"
	"fishyAHP/LogParser.git/internal/features/index/timestamp"
)

type IndexService struct {
	level     Index[domain.LogLevel]
	component Index[domain.LogComponent]
	pid       Index[domain.PID]
	ip        Index[domain.IP]

	timestamp TimeIndex
	text      TextIndex
}

func New(timeAccuracy time.Duration) *IndexService {
	return &IndexService{
		level:     hash_index.New[domain.LogLevel](),
		component: hash_index.New[domain.LogComponent](),
		pid:       hash_index.New[domain.PID](),
		ip:        hash_index.New[domain.IP](),
		timestamp: timestamp.New(timeAccuracy),
		text:      text.New(),
	}
}

func (i *IndexService) SetTokenizer(tokenizer *words.Tokenizer) {
	t := (i.text).(*text.Index)
	t.Tokenizer = tokenizer
}

func (i *IndexService) Index(record domain.RecordData, entry domain.LogEntry) {
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

	i.timestamp.Add(entry.Timestamp, record)
}

func (i *IndexService) Clear() {
	i.level.Clear()
	i.component.Clear()
	i.pid.Clear()
	i.ip.Clear()
	i.timestamp.Clear()
}

func (i *IndexService) ByLevel(level domain.LogLevel) *set.Set[domain.RecordData] {
	sett, ok := i.level.Get(level)
	if !ok {
		return nil
	}

	return sett
}

func (i *IndexService) ByComponent(component domain.LogComponent) *set.Set[domain.RecordData] {
	sett, ok := i.component.Get(component)
	if !ok {
		return nil
	}

	return sett
}

func (i *IndexService) ByIP(ip domain.IP) *set.Set[domain.RecordData] {
	sett, ok := i.ip.Get(ip)
	if !ok {
		return nil
	}

	return sett
}

func (i *IndexService) ByTimestampRange(from, to time.Time) *set.Set[domain.RecordData] {
	sett, ok := i.timestamp.Range(from, to)
	if !ok {
		return nil
	}

	return sett
}

func (i *IndexService) ByText(text string) *set.Set[domain.RecordData] {
	return i.text.Get(text)
}
