package domain

import (
	"errors"
	"fmt"
	"strings"
)

type Scheme struct {
	Separator  rune
	Parameters []Field
}

type Field struct {
	Name      string
	FieldType DataType
	IndexType IndexType
}

type DataType uint8

const (
	Invalid DataType = iota
	StringType
	IntType
	BoolType
	FloatType
	TimeType
)

type IndexType uint8

const (
	NoIndex IndexType = iota
	HashIndex
	TextIndex
	RangeIndex
)

func (s *Scheme) Validate() error {
	if s.Separator == 0 {
		return errors.New("separator mustn't be 0")
	}

	if len(s.Parameters) == 0 {
		return errors.New("parameters length must be more than zero")
	}

	m := make(map[string]struct{})
	for _, field := range s.Parameters {
		if err := field.validate(); err != nil {
			return fmt.Errorf("schema validate parameters: %w", err)
		}

		if _, ok := m[field.Name]; ok {
			return errors.New("field name must be unique")
		}

		m[field.Name] = struct{}{}
	}

	return nil
}

func (f *Field) validate() error {
	if strings.TrimSpace(f.Name) == "" {
		return errors.New("name must be not void")
	}

	m := map[DataType]struct{}{
		StringType: {}, IntType: {}, FloatType: {},
		BoolType: {}, TimeType: {},
	}

	if _, ok := m[f.FieldType]; !ok {
		return errors.New("field type must be in initialized types")
	}

	return nil
}
