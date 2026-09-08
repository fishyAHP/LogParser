package timestamp

import (
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
)

type Color int

const (
	Black Color = iota
	Red
)

type Node struct {
	key     time.Time
	records []domain.RecordData

	left, right *Node
	parent      *Node

	color Color
}
