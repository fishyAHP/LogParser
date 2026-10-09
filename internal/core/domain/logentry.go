package domain

import "time"

// ParsedLog all fields with LogEntry
type ParsedLog struct {
	Entry  LogEntry
	Fields []string
}

// LogEntry indexing part of log
type LogEntry struct {
	Values []Value
}

type Value interface {
	isValue()
}

type IntValue int64
type FloatValue float64
type StringValue string
type BoolValue bool
type TimeValue time.Time

func (i IntValue) isValue()    {}
func (f FloatValue) isValue()  {}
func (s StringValue) isValue() {}
func (b BoolValue) isValue()   {}
func (t TimeValue) isValue()   {}
