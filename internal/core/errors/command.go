package coreerrors

import "errors"

var (
	ErrIndexNotFound       = errors.New("index wasn't found")
	ErrUnsupportedOperator = errors.New("unsupported operator")
	ErrUnknownIndexType    = errors.New("unknown index type in field")
	ErrNotGrouper          = errors.New("index doesn't support grouping")
)
