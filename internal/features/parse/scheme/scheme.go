package scheme

import (
	"errors"
	"fmt"
	"strings"
)

type DataType uint8

const (
	Invalid DataType = iota
	String
	Int
	Bool
	Float
	Time
	Array
	Map
	Object
)

type Field struct {
	Name         string
	FieldType    DataType
	ObjectFields []Field
	Elements     *Field
	Map          *[2]Field
}

type Scheme struct {
	Separator  rune
	Parameters []Field
}

func (s *Scheme) Validate() error {
	if s.Separator == 0 {
		return errors.New("separator mustn't be 0")
	}

	if len(s.Parameters) == 0 {
		return errors.New("parameters length must be more than zero")
	}

	m := make(map[string]struct{})
	for _, field := range s.Parameters {
		if err := field.Validate(); err != nil {
			return fmt.Errorf("schema validate parameters: %w", err)
		}

		if _, ok := m[field.Name]; ok {
			return errors.New("field name must be unique")
		}

		m[field.Name] = struct{}{}
	}

	return nil
}

func (f *Field) Validate() error {
	if strings.TrimSpace(f.Name) == "" {
		return errors.New("name must be not void")
	}

	m := map[DataType]struct{}{
		String: {}, Int: {}, Float: {},
		Bool: {}, Time: {}, Object: {},
		Array: {}, Map: {}, Invalid: {},
	}

	if _, ok := m[f.FieldType]; !ok {
		return errors.New("field type must be in initialized types")
	}

	switch f.FieldType {
	case Invalid:
		return errors.New("field type must be not invalid")
	case Object:
		if len(f.ObjectFields) == 0 {
			return errors.New("object fields must have more than zero")
		}

		if f.Elements != nil {
			return errors.New("in object type elements must be nil")
		}

		for _, field := range f.ObjectFields {
			if err := field.Validate(); err != nil {
				return fmt.Errorf("validate field object type: %w", err)
			}
		}
	case Array:
		if len(f.ObjectFields) != 0 {
			return errors.New("in slice length of object fields must be zero")
		}
		if f.Elements == nil {
			return errors.New("in slice elements must be non nil")
		}

		if err := f.Elements.Validate(); err != nil {
			return fmt.Errorf("validate field slice type: %w", err)
		}
	case Map:
		if len(f.ObjectFields) != 0 {
			return errors.New("in map length of object fields must be zero")
		}
		if f.Elements != nil {
			return errors.New("in map elements must be nil")
		}

		for _, field := range f.Map {
			if err := field.Validate(); err != nil {
				return fmt.Errorf("validate field map type: %w", err)
			}
		}
	default:
		break
	}
	return nil
}
