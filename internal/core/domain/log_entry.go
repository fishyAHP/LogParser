package domain

import "time"

// LogEntry логическое представление записи лога
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
