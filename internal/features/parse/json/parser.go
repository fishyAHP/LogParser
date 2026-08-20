package features_parser_json

import "fishyAHP/LogParser.git/internal/core/domain"

type JSONParser struct {
}

// Parse принимает слайс байт на выходе должна дать лог.
// Если json не валидный то возвращаешь ошибку, можешь сделать отдельный файл для описания типа ошибки.
// Можешь сделать промежуточную структуру для декодирования json'а.
func (jp *JSONParser) Parse(data []byte) (domain.LogEntry, error) {
	return domain.LogEntry{}, nil
}
