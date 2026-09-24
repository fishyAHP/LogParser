package features_parse

import (
	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/core/scheme"
	"fishyAHP/LogParser.git/internal/features/tokenizer"
)

type LogParser struct {
	scheme scheme.Scheme
}

func NewParser(s scheme.Scheme) LogParser {
	return LogParser{
		scheme: s,
	}
}

func (p *LogParser) Parse(data []tokenizer.Token) (domain.LogEntry, error) {
	cleanToken := make([]string, 0)
	for _, value := range data {
		if value.Type == tokenizer.TokenString {
			cleanToken = append(cleanToken, value.Value)
		}
	}

	logEntry := domain.LogEntry{
		Fields: make(map[string]string),
	}

	for i, field := range p.scheme.Parameters {
		value := cleanToken[i]

		switch field.FieldType {

		case scheme.IntType:
			_, err := strconv.Atoi(value)
			if err != nil {
				return domain.LogEntry{}, err
			}
			logEntry.Fields[field.Name] = value

		case scheme.TimeType:
			parsedTime, err := time.Parse(time.RFC3339, value)
			if err != nil {
				return domain.LogEntry{}, err
			}
			logEntry.Timestamp = parsedTime

		case scheme.StringType:
			if field.Name == "message" {
				logEntry.Message = value
			} else {
				logEntry.Fields[field.Name] = value
			}
		}
	}
	return logEntry, nil
}

// Parser представляет собой интерфейс, который позволяет с помощью метода Parse
// превратить слайс байт в логическое представление лога и ошибку при невалидном входном слайсе
type Parser interface {
	Parse(data []tokenizer.Token) (domain.LogEntry, error)
}