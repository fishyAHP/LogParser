package timestamp

import (
	"slices"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
)

type color int

const (
	Red color = iota
	Black
)

type node struct {
	key     time.Time
	records []domain.RecordData

	left, right *node
	parent      *node

	color color
}

func newNode(key time.Time, value domain.RecordData) *node {
	return &node{
		key:     key,
		records: []domain.RecordData{value},
		color:   Black,
	}
}

func (n *node) add(value domain.RecordData) {
	n.records = slices.Clone(append(n.records, value))
}
