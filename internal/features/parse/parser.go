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

// Parser представляет собой интерфейс, который позволяет с помощью метода Parse
// превратить слайс байт в логическое представление лога и ошибку при невалидном входном слайсе
type Parser interface {
	Parse(data []tokenizer.Token) (domain.LogEntry, error)