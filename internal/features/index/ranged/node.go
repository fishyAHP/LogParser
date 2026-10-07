package ranged

import (
	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/structs"
)

type color uint

const (
	Red color = iota
	Black
)

type node[K comparable] struct {
	key     K
	records *structs.Set[domain.RecordID]

	left, right *node[K]
	parent      *node[K]

	color color
}

func newNode[K comparable](key K, value domain.RecordID) *node[K] {
	s := structs.NewSet[domain.RecordID](0)
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
