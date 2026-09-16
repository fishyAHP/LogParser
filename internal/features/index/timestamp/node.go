package timestamp

import (
	"slices"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
)

type color uint

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
		color:   Red,
	}
}

func (n *node) add(value domain.RecordData) {
	n.records = slices.Clone(append(n.records, value))
}

func (n *node) uncle() *node {
	parent := n.parent

	if parent.left == n {
		return parent.right
	}
	return parent.left
}
