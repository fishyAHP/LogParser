package features_parse

import "fishyAHP/LogParser.git/internal/core/domain"

// Parser представляет собой интерфейс, который позволяет с помощью метода Parse
// превратить слайс байт в логическое представление лога и ошибку при невалидном входном слайсе
type Parser interface {
	Parse(data []byte) (domain.LogEntry, error)
}
