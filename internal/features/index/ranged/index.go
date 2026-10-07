package ranged

import (
	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/structs"

	"fishyAHP/LogParser.git/internal/features/index/common"
)

type Index[K common.Key] struct {
	tree *rbTree[K]
}
type localBound[K common.Key] struct {
	value     K
	inclusive bool
}

func (i *Index[K]) Range(
	from, to *common.Bound,
) (*structs.Set[domain.RecordID], error) {
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
			inclusive: from.Inclusive,
		}

	}
	if to != nil {
		toVal, ok := to.Value.(K)
		if !ok {
			return nil, common.ErrInvalidType
		}
		toBound = &localBound[K]{
			value:     toVal,
			inclusive: to.Inclusive,
		}
	}

	records, ok := i.tree.innerRange(fromBound, toBound)
	if !ok {
		return nil, common.ErrNotFoundRecord
	}

	s := structs.NewSet[domain.RecordID](len(records))
	s.AddMany(records...)

	return s, nil
}

func (i *Index[K]) Exact(
	value domain.Value,
) (*structs.Set[domain.RecordID], error) {
	key, ok := value.(K)
	if !ok {
		return nil, common.ErrInvalidType
	}

	res, ok := i.tree.find(key)
	if !ok {
		return nil, common.ErrNotFoundRecord
	}
	return res, nil
}

func (i *Index[K]) Add(
	value domain.Value,
	record domain.RecordID,
) error {
	key, ok := value.(K)
	if !ok {
		return common.ErrInvalidType
	}
	return i.tree.insert(key, record)
}

func New[K common.Key](comparator func(K, K) int) *Index[K] {
	return &Index[K]{
		tree: newRBTree(comparator),
	}
}

func (i *Index[K]) Remove(
	value domain.Value,
	data domain.RecordID,
) error {
	key, ok := value.(K)
	if !ok {
		return common.ErrInvalidType
	}

	deleted := i.tree.remove(key, data)
	if !deleted {
		return common.ErrNotFoundRecord
	}
	return nil
}

func (i *Index[K]) Min() *structs.Set[domain.RecordID] {
	if i.tree.len() == 0 {
		return nil
	}
	if i.tree.len() == 1 {
		return i.tree.root.records
	}

	minNode := i.tree.min()

	return minNode.records
}

func (i *Index[K]) Max() *structs.Set[domain.RecordID] {
	if i.tree.len() == 0 {
		return nil
	}
	if i.tree.len() == 1 {
		return i.tree.root.records
	}

	maxNode := i.tree.max()

	return maxNode.records.Clone()
}

func (i *Index[K]) Len() int {
	return i.tree.elemsCount
}

func (i *Index[K]) Clear() {
	i.tree.clear()
}
