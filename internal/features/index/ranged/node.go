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
	records *records

	left, right *node[K]
	parent      *node[K]

	color color
}

func newNode[K comparable](key K, value domain.RecordID) *node[K] {
	s := structs.NewSet[domain.RecordID](0)
	s.Add(value)

	return &node[K]{
		key:     key,
		records: newRecs(value),
		color:   Red,
	}
}

func (n *node[K]) add(value domain.RecordID) bool {
	return n.records.add(value)
}

func (n *node[K]) uncle() *node[K] {
	parent := n.parent
	grandparent := parent.parent

	if grandparent.left == parent {
		return grandparent.right
	}
	return grandparent.left
}

type records struct {
	single domain.RecordID
	many   *structs.PostingList
}

func newRecs(id domain.RecordID) *records {
	return &records{
		single: id,
	}
}

func (r *records) add(id domain.RecordID) bool {
	if r.single == 0 &&
		r.many == nil {
		r.single = id
		return true
	}

	if r.many == nil {
		r.many = structs.NewPostingLists(2)
		r.many.Add(r.single)
		r.single = 0
	}

	return r.many.Add(id)
}

func (r *records) count() int {
	if r.single == 0 &&
		r.many == nil {
		return 0
	}

	if r.many == nil {
		return 1
	}
	return r.many.Len()
}

func (r *records) toPosting() *structs.PostingList {
	if r.single == 0 &&
		r.many == nil {
		return &structs.PostingList{}
	}

	if r.many == nil {
		post := structs.NewPostingLists(1)
		post.Add(r.single)
		return post
	}

	return r.many
}

func (r *records) slice() []domain.RecordID {
	if r.single == 0 &&
		r.many == nil {
		return nil
	}

	if r.many == nil {
		return []domain.RecordID{r.single}
	}
	return r.many.Slice()
}
