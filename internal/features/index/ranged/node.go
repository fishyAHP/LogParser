package ranged

import (
	"iter"

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
	return &node[K]{
		key:     key,
		records: newRecs(value),
		color:   Red,
	}
}

func newNodeWithPosting[K comparable](
	key K,
	posting *structs.PostingList,
) *node[K] {
	rec := &records{}

	if posting.Len() == 1 {
		rec.single = posting.At(0)
	} else {
		rec.many = posting
	}

	return &node[K]{
		key:     key,
		records: rec,
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

func (r *records) isEmpty() bool {
	return r.single == 0 &&
		r.many == nil
}

func (r *records) add(id domain.RecordID) bool {
	if r.isEmpty() {
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
	if r.isEmpty() {
		return 0
	}

	if r.many == nil {
		return 1
	}
	return r.many.Len()
}

func (r *records) at(idx int) domain.RecordID {
	if r.isEmpty() {
		return 0
	}
	if r.many == nil {
		return r.single
	}
	return r.many.At(idx)
}

func (r *records) toPosting() *structs.PostingList {
	if r.isEmpty() {
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
	if r.isEmpty() {
		return nil
	}

	if r.many == nil {
		return []domain.RecordID{r.single}
	}
	return r.many.Slice()
}

func (r *records) contains(id domain.RecordID) bool {
	if r.isEmpty() {
		return false
	}

	if r.many == nil {
		return r.single == id
	}
	return r.many.Contains(id)
}

func (r *records) iterator() iter.Seq[domain.RecordID] {
	return func(yield func(domain.RecordID) bool) {
		if r.isEmpty() {
			return
		}

		if r.many == nil {
			yield(r.single)
			return
		}
		for id := range r.many.Iterator() {
			if !yield(id) {
				return
			}
		}
	}
}
