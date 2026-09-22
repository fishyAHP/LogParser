package timestamp

import (
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/set"
)

type color uint

const (
	Red color = iota
	Black
)

type node struct {
	key     time.Time
	records *set.Set[domain.RecordData]

	left, right *node
	parent      *node

	color color
}

func newNode(key time.Time, value domain.RecordData) *node {
	s := set.New[domain.RecordData](0)
	s.Add(value)

	return &node{
		key:     key,
		records: s,
		color:   Red,
	}
}

func (n *node) add(value domain.RecordData) bool {
	return n.records.Add(value)
}

func (n *node) uncle() *node {
	parent := n.parent
	grandparent := parent.parent

	if grandparent.left == parent {
		return grandparent.right
	}
	return grandparent.left
}
