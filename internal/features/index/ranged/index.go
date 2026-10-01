package ranged

import (
	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/set"

	"fishyAHP/LogParser.git/internal/features/index/common"
)

type Index[K common.Key] struct {
	tree *rbTree[K]
}

type localBound[K common.Key] struct {
	value     K
	exclusive bool
}

func (i *Index[K]) Range(
	from, to *common.Bound,
) (*set.Set[domain.RecordData], error) {
	var (
		fromBound *localBound[K]
		toBound   *localBound[K]
	)
	if from != nil {
		fromVal, ok := from.Value.(K)
		if !ok {
			return nil, common.ErrInvalidType
		}
		fromBound = &localBound[K]{
			value:     fromVal,
			exclusive: from.Exclusive,
		}

	}
	if to != nil {
		toVal, ok := to.Value.(K)
		if !ok {
			return nil, common.ErrInvalidType
		}
		toBound = &localBound[K]{
			value:     toVal,
			exclusive: to.Exclusive,
		}
	}

	records, ok := i.tree.Range(fromBound, toBound)
	if !ok {
		return nil, common.ErrNotFoundRecord
	}

	s := set.New[domain.RecordData](len(records))
	s.AddMany(records...)

	return s, nil
}

func (i *Index[K]) Exact(
	value domain.Value,
) (*set.Set[domain.RecordData], error) {
	key, ok := value.(K)
	if !ok {
		return nil, common.ErrInvalidType
	}

	res, ok := i.tree.Find(key)
	if !ok {
		return nil, common.ErrNotFoundRecord
	}
	return res, nil
}

func (i *Index[K]) Add(
	value domain.Value,
	record domain.RecordData,
) error {
	key, ok := value.(K)
	if !ok {
		return common.ErrInvalidType
	}
	return i.tree.Insert(key, record)
}

func New[K common.Key](comparator func(K, K) int) *Index[K] {
	return &Index[K]{
		tree: newRBTree(comparator),
	}
}

func (i *Index[K]) Remove(
	value domain.Value,
	data domain.RecordData,
) error {
	key, ok := value.(K)
	if !ok {
		return common.ErrInvalidType
	}

	deleted := i.tree.Remove(key, data)
	if !deleted {
		return common.ErrNotFoundRecord
	}
	return nil
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

func (i *Index[K]) Clear() {
	i.tree.Clear()
}
