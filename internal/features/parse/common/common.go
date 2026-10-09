package common

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"

	"fishyAHP/LogParser.git/internal/core/domain"
)

// Parser представляет собой интерфейс, который позволяет с помощью метода Parse
// превратить слайс байт в логическое представление лога и ошибку при невалидном входном слайсе
type Parser interface {
	Parse(data []byte) (domain.ParsedLog, error)
}

type Projection interface {
	Project([]byte, []string) (Projections, error)
}

type ProjectedField struct {
	Name  string
	Value jsontext.Value
}

type Projections []ProjectedField

func (p Projections) MarshalJSON() ([]byte, error) {
	capacity := 2

	for _, field := range p {
		capacity += len(field.Name) + len(field.Value)
	}

	var buffer bytes.Buffer
	buffer.Grow(capacity)

	buffer.WriteByte('{')

	for i, projection := range p {
		if i > 0 {
			buffer.WriteByte(',')
		}

		name, err := json.Marshal(projection.Name)
		if err != nil {
			return nil, err
		}

		buffer.Write(name)
		buffer.WriteByte(':')
		buffer.Write(projection.Value)
	}

	buffer.WriteByte('}')
	return buffer.Bytes(), nil
}
