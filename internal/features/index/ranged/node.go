package ranged

import (
	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/set"
)

type color uint

const (
	Red color = iota
	Black
)

type node[K comparable] struct {
	key     K
	records *set.Set[domain.RecordID]

	left, right *node[K]
	parent      *node[K]

	color color
}

func newNode[K comparable](key K, value domain.RecordID) *node[K] {
	s := set.New[domain.RecordID](0)
	s.Add(value)

	return &node[K]{
		key:     key,
		records: s,
		color:   Red,
	}
}

func (n *node[K]) add(value domain.RecordID) bool {
	return n.records.Add(value)
}

func (n *node[K]) uncle() *node[K] {
	parent := n.parent
	grandparent := parent.parent

	if grandparent.left == parent {
		return grandparent.right
	}
	return grandparent.left
}
