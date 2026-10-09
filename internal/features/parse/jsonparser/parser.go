package jsonparser

import (
	"bytes"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"
	"strconv"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/parse/common"
)

type JSONParser struct {
	scheme *domain.Scheme
	values map[string]jsontext.Value
}

func New(scheme *domain.Scheme) *JSONParser {
	return &JSONParser{
		scheme: scheme,
		values: make(map[string]jsontext.Value, 5),
	}
}

func (jp *JSONParser) Parse(data []byte) (domain.ParsedLog, error) {
	if err := jsonv2.Unmarshal(data, &jp.values); err != nil {
		return domain.ParsedLog{}, fmt.Errorf(
			"unmarshal json: %w",
			err,
		)
	}

	fields := make([]string, 0, len(jp.values))
	for name := range jp.values {
		fields = append(fields, name)
	}

	entry := domain.LogEntry{
		Values: make([]domain.Value, 0, len(jp.scheme.Parameters)),
	}
	for _, field := range jp.scheme.Parameters {
		strVal, ok := jp.values[field.Name]
		if !ok {
			return domain.ParsedLog{}, fmt.Errorf(
				"field %q not found",
				field.Name,
			)
		}

		val, err := decodeValue(strVal, field.FieldType)
		if err != nil {
			return domain.ParsedLog{}, fmt.Errorf(
				"decode field %q: %w",
				field.Name,
				err,
			)
		}

		entry.Values = append(entry.Values, val)
	}

	clear(jp.values)
	return domain.ParsedLog{
		Entry:  entry,
		Fields: fields,
	}, nil
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

func (jp *JSONParser) Project(
	data []byte,
	fields []string,
) (common.Projections, error) {
	decoder := jsontext.NewDecoder(bytes.NewReader(data))

	token, err := decoder.ReadToken()
	if err != nil {
		return nil, fmt.Errorf(
			"start read json: %w",
			err,
		)
	}

	if token.Kind() != '{' {
		return nil, errors.New(
			"expect json object",
		)
	}

	res := make(common.Projections, len(fields))

	for i, field := range fields {
		res[i] = common.ProjectedField{
			Name:  field,
			Value: jsontext.Value("null"),
		}
	}

	for decoder.PeekKind() != '}' {
		token, err := decoder.ReadToken()
		if err != nil {
			return nil, fmt.Errorf(
				"read field name: %w",
				err,
			)
		}
		name := token.String()

		value, err := decoder.ReadValue()
		if err != nil {
			return nil, fmt.Errorf(
				"read field value: %w",
				err,
			)
		}

		for i := range res {
			if res[i].Name == name {
				res[i].Value = value.Clone()
				break
			}
		}
	}

	if _, err := decoder.ReadToken(); err != nil {
		return nil, fmt.Errorf(
			"read end object: %w",
			err,
		)
	}

	return res, nil
}
