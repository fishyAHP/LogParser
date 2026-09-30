package parse

import (
	"fmt"
	"strconv"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/tokenizer"
)

// Parser представляет собой интерфейс, который позволяет с помощью метода Parse
// превратить слайс байт в логическое представление лога и ошибку при невалидном входном слайсе
type Parser interface {
	Parse(data []tokenizer.Token) (domain.LogEntry, error)
}

type LogParser struct {
	scheme domain.Scheme
}

func NewParser(s domain.Scheme) LogParser {
	return LogParser{
		scheme: s,
	}
}

func (p *LogParser) Parse(data []tokenizer.Token) (domain.LogEntry, error) {
	cleanToken := make([]string, 0, len(data)/2)
	for _, value := range data {
		if value.Type == tokenizer.TokenString {
			cleanToken = append(cleanToken, value.Value)
		}
	}
	if len(cleanToken) != len(p.scheme.Parameters) {
		return domain.LogEntry{},
			fmt.Errorf("want %d parameters, got %d", len(p.scheme.Parameters), len(cleanToken))
	}

	logEntry := domain.LogEntry{
		Other: make(map[string]string),
	}
	for i, field := range p.scheme.Parameters {
		value := cleanToken[i]

		switch field.FieldType {
		case domain.IntType:
			_, err := strconv.Atoi(value)
			if err != nil {
				return domain.LogEntry{}, err
			}
			logEntry.Other[field.Name] = value

		case domain.TimeType:
			parsedTime, err := time.Parse(time.RFC3339, value)
			if err != nil {
				return domain.LogEntry{}, err
			}
			logEntry.Timestamp = parsedTime

		case domain.BoolType:
			_, err := strconv.ParseBool(value)
			if err != nil {
				return domain.LogEntry{}, err
			}
			logEntry.Other[field.Name] = value

		case domain.FloatType:
			_, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return domain.LogEntry{}, err
			}
			logEntry.Other[field.Name] = value

		case domain.StringType:
			if field.Name == "message" {
				logEntry.Message = value
			} else {
				logEntry.Other[field.Name] = value
			}
		default:
		}
	}
	return logEntry, nil
}
