package parse

import (
	"fmt"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/parse/common"
	"fishyAHP/LogParser.git/internal/features/parse/jsonparser"
)

type LogParser struct {
	parser common.Parser
}

func NewParser(
	s *domain.Scheme,
	format domain.Format,
) *LogParser {
	var parser common.Parser
	switch format {
	case domain.JSON:
		parser = jsonparser.New(s)
	}
	return &LogParser{
		parser: parser,
	}
}

func (p *LogParser) Parse(
	data []byte,
) (domain.ParsedLog, error) {
	return p.parser.Parse(data)
}

func (p *LogParser) Project(
	data []byte,
	fields []string,
) (common.Projections, error) {
	projection, ok := p.parser.(common.Projection)
	if !ok {
		return nil, fmt.Errorf(
			"unsupported projections: %T",
			p.parser,
		)
	}

	return projection.Project(data, fields)
}
