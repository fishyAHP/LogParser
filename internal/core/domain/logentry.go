package domain

import (
	"encoding/json/v2"
	"time"
)

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
	Type() DataType
}

type IntValue int64
type FloatValue float64
type StringValue string
type BoolValue bool
type TimeValue time.Time

func (IntValue) Type() DataType {
	return IntType
}
func (FloatValue) Type() DataType {
	return FloatType
}
func (StringValue) Type() DataType {
	return StringType
}
func (BoolValue) Type() DataType {
	return BoolType
}
func (TimeValue) Type() DataType {
	return TimeType
}

func (t TimeValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(t).Format(time.RFC3339Nano))
}
