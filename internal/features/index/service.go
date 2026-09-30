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

type Service struct {
	level     Index[domain.LogLevel]
	component Index[domain.LogComponent]
	pid       Index[domain.PID]
	ip        Index[domain.IP]

	timestamp TimeIndex
	text      TextIndex
}

func New(timeAccuracy time.Duration) *Service {
	return &Service{
		level:     hash_index.New[domain.LogLevel](),
		component: hash_index.New[domain.LogComponent](),
		pid:       hash_index.New[domain.PID](),
		ip:        hash_index.New[domain.IP](),
		timestamp: timestamp.New(timeAccuracy),
		text:      text.New(),
	}
}

func (s *Service) SetTokenizer(tokenizer *words.Tokenizer) {
	t := (s.text).(*text.Index)
	t.Tokenizer = tokenizer
}

func (s *Service) Index(record domain.RecordData, entry domain.LogEntry) {
	if strings.TrimSpace(string(entry.Level)) != "" {
		s.level.Add(entry.Level, record)
	}

	if strings.TrimSpace(entry.Component) != "" {
		s.component.Add(entry.Component, record)
	}

	if entry.PID != 0 {
		s.pid.Add(entry.PID, record)
	}

	if !entry.IP.IsZero() {
		s.ip.Add(entry.IP, record)
	}

	s.timestamp.Add(entry.Timestamp, record)
}

func (s *Service) Clear() {
	s.level.Clear()
	s.component.Clear()
	s.pid.Clear()
	s.ip.Clear()
	s.timestamp.Clear()
}

func (s *Service) ByLevel(level domain.LogLevel) *set.Set[domain.RecordData] {
	sett, ok := s.level.Get(level)
	if !ok {
		return nil
	}

	return sett
}

func (s *Service) ByComponent(component domain.LogComponent) *set.Set[domain.RecordData] {
	sett, ok := s.component.Get(component)
	if !ok {
		return nil
	}

	return sett
}

func (s *Service) ByIP(ip domain.IP) *set.Set[domain.RecordData] {
	sett, ok := s.ip.Get(ip)
	if !ok {
		return nil
	}

	return sett
}

func (s *Service) ByTimestampRange(from, to time.Time) *set.Set[domain.RecordData] {
	sett, ok := s.timestamp.Range(from, to)
	if !ok {
		return nil
	}

	return sett
}

func (s *Service) ByText(text string) *set.Set[domain.RecordData] {
	return s.text.Get(text)
}
