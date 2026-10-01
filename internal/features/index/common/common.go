package common

import (
	"errors"

	"fishyAHP/LogParser.git/internal/core/domain"
)

var (
	ErrInvalidType    = errors.New("invalid value type")
	ErrNotFoundRecord = errors.New("not found record")
)

type Key interface {
	domain.Value
	comparable
}
