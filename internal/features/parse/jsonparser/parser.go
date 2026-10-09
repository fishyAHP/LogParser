package jsonparser

import (
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"fmt"
	"strconv"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
)

type JSONParser struct {
	scheme *domain.Scheme
}

func New(scheme *domain.Scheme) *JSONParser {
	return &JSONParser{
		scheme: scheme,
	}
}

func (jp *JSONParser) Parse(data []byte) (domain.LogEntry, error) {
	var values map[string]jsontext.Value
	if err := jsonv2.Unmarshal(data, &values); err != nil {
		return domain.LogEntry{}, fmt.Errorf(
			"unmarshal json: %w",
			err,
		)
	}

	res := make([]domain.Value, 0, len(jp.scheme.Parameters))
	for _, field := range jp.scheme.Parameters {
		strVal, ok := values[field.Name]
		if !ok {
			return domain.LogEntry{}, fmt.Errorf(
				"field %q not found",
				field.Name,
			)
		}

		val, err := decodeValue(strVal, field.FieldType)
		if err != nil {
			return domain.LogEntry{}, fmt.Errorf(
				"decode field %q: %w",
				field.Name,
				err,
			)
		}

		res = append(res, val)
	}

	return domain.LogEntry{Values: res}, nil
}

func decodeValue(
	value jsontext.Value,
	typ domain.DataType,
) (domain.Value, error) {
	strVal := value.String()
	switch typ {
	case domain.IntType:
		intVal, err := strconv.ParseInt(strVal, 10, 64)
		if err != nil {
			return nil, fmt.Errorf(
				"parse int: %w",
				err,
			)
		}

		return domain.IntValue(intVal), nil
	case domain.TimeType:
		parsedTime, err := time.Parse(time.RFC3339, strVal[1:len(strVal)-1])
		if err != nil {
			return nil, fmt.Errorf(
				"parse time: %w",
				err,
			)
		}

		return domain.TimeValue(parsedTime), nil
	case domain.BoolType:
		boolVal, err := strconv.ParseBool(strVal)
		if err != nil {
			return nil, fmt.Errorf(
				"parse bool: %w",
				err,
			)
		}

		return domain.BoolValue(boolVal), nil
	case domain.FloatType:
		floatVal, err := strconv.ParseFloat(strVal, 64)
		if err != nil {
			return nil, fmt.Errorf(
				"parse float: %w",
				err,
			)
		}

		return domain.FloatValue(floatVal), nil
	case domain.StringType:
		var parsed string
		if err := jsonv2.Unmarshal(value, &parsed); err != nil {
			return nil, fmt.Errorf(
				"parse string: %w",
				err,
			)
		}

		return domain.StringValue(parsed), nil
	default:
		return nil, fmt.Errorf("unknown type in scheme: %v", typ)
	}
}

type ProjectedField struct {
	Name  string
	Value jsontext.Value
}

func (jp *JSONParser) Projection(
	data []byte,
	fields []string,
) ([]ProjectedField, error) {
	var values map[string]jsontext.Value
	if err := jsonv2.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf(
			"unmarshal json: %w",
			err,
		)
	}

	res := make([]ProjectedField, 0, len(fields))
	for _, field := range fields {
		value, ok := values[field]
		if !ok {
			value = jsontext.Value("null")
		}

		res = append(res, ProjectedField{
			Name:  field,
			Value: value,
		})
	}

	return res, nil
}
