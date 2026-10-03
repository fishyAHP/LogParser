package parse

import (
	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/parse/jsonparser"
)

// Parser представляет собой интерфейс, который позволяет с помощью метода Parse
// превратить слайс байт в логическое представление лога и ошибку при невалидном входном слайсе
type Parser interface {
	Parse(data []byte) (domain.LogEntry, error)
}

type LogParser struct {
	parser Parser
}

func NewParser(s *domain.Scheme, format domain.Format) *LogParser {
	var parser Parser
	switch format {
	case domain.JSON:
		parser = jsonparser.New(s)
	}
	return &LogParser{
		parser: parser,
	}
}

func (p *LogParser) Parse(data []byte) (domain.LogEntry, error) {
	return p.parser.Parse(data)
}
