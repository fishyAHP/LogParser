package ranged

import (
	"errors"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/set"
)

type Key interface {
	domain.Value
	comparable
}

type Index[K Key] struct {
	tree *rbTree[K]
}

var ErrInvalidType = errors.New("invalid type type")

func (i *Index[K]) Range(from, to domain.Value) (*set.Set[domain.RecordData], error) {
	fromVal, ok := from.(K)
	if !ok {
		return nil, ErrInvalidType
	}
	toVal, ok := to.(K)
	if !ok {
		return nil, ErrInvalidType
	}

	records, ok := i.tree.Range(fromVal, toVal)
	if !ok {
		return nil, nil
	}

	s := set.New[domain.RecordData](len(records))
	s.AddMany(records...)

	return s, nil
}

func (i *Index[K]) Exact(value domain.Value) (*set.Set[domain.RecordData], error) {
	key, ok := value.(K)
	if !ok {
		return nil, ErrInvalidType
	}
	res, _ := i.tree.Find(key)
	return res, nil
}

func (i *Index[K]) Add(value domain.Value, record domain.RecordData) error {
	key, ok := value.(K)
	if !ok {
		return ErrInvalidType
	}
	return i.tree.Insert(key, record)
}

func New[K Key](comparator func(K, K) int) *Index[K] {
	return &Index[K]{
		tree: newRBTree(comparator),
	}
}

func (i *Index[K]) Remove(key K) bool {
	return i.tree.Remove(key)
}

func (i *Index[K]) Min() *set.Set[domain.RecordData] {
	if i.tree.Len() == 0 {
		return nil
	}
	if i.tree.Len() == 1 {
		return i.tree.root.records
	}

	minNode := i.tree.Min()

	return minNode.records
}

func (i *Index[K]) Max() *set.Set[domain.RecordData] {
	if i.tree.Len() == 0 {
		return nil
	}
	if i.tree.Len() == 1 {
		return i.tree.root.records
	}

	maxNode := i.tree.Max()

	return maxNode.records
}

func (i *Index[K]) Len() int {
	return i.tree.elemsCount
}

func (i *Index[K]) Delete(key K, value domain.RecordData) bool {
	return i.tree.Delete(key, value)
}

func (i *Index[K]) Clear() {
	i.tree.Clear()
}
