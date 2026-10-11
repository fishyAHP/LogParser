package ranged

import (
	"fmt"
	"slices"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/structs"

	"fishyAHP/LogParser.git/internal/features/index/common"
)

type Index[K common.Key] struct {
	tree *rbTree[K]
}

func New[K common.Key](comparator func(K, K) int) *Index[K] {
	return &Index[K]{
		tree: newRBTree(comparator),
	}
}

type localBound[K common.Key] struct {
	value     K
	inclusive bool
}

func (i *Index[K]) Range(
	from, to *common.Bound,
) (*structs.PostingList, error) {
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

	res, ok := i.tree.innerRange(fromBound, toBound)
	if !ok {
		return nil, common.ErrRecordNotFound
	}

	slices.Sort(res)
	return structs.NewPostingFromSorted(res), nil
}

func (i *Index[K]) Exact(
	value domain.Value,
) (*structs.PostingList, error) {
	key, ok := value.(K)
	if !ok {
		return nil, common.ErrInvalidType
	}

	res, ok := i.tree.find(key)
	if !ok {
		return nil, common.ErrRecordNotFound
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
		return common.ErrRecordNotFound
	}
	return nil
}

func (i *Index[K]) minNode() *node[K] {
	if i.tree.len() == 0 {
		return nil
	}
	if i.tree.len() == 1 {
		return i.tree.root
	}

	minNode := i.tree.min()

	return minNode
}

func (i *Index[K]) maxNode() *structs.PostingList {
	if i.tree.len() == 0 {
		return nil
	}
	if i.tree.len() == 1 {
		return i.tree.root.
			records.
			toPosting()
	}

	maxNode := i.tree.max()

	return maxNode.
		records.
		toPosting()
}

func (i *Index[K]) Len() int {
	return i.tree.elemsCount
}

func (i *Index[K]) Clear() {
	i.tree.clear()
}

func (i *Index[K]) Min(
	posting *structs.PostingList,
) (domain.Value, error) {
	return i.tree.extremum(
		posting,
		ascending,
	)
}

func (i *Index[K]) Max(
	posting *structs.PostingList,
) (domain.Value, error) {
	return i.tree.extremum(
		posting,
		descending,
	)
}

func (i *Index[K]) Avg(
	posting *structs.PostingList,
) (domain.Value, error) {
	sum, count, err := i.sumAndCount(posting)
	if err != nil {
		return nil, err
	}

	return sum / domain.FloatValue(count), nil
}

func (i *Index[K]) Sum(
	posting *structs.PostingList,
) (domain.Value, error) {
	sum, _, err := i.sumAndCount(posting)
	if err != nil {
		return nil, err
	}
	return sum, nil
}

func (i *Index[K]) sumAndCount(
	posting *structs.PostingList,
) (domain.FloatValue, int, error) {
	var (
		count int
		sum   domain.FloatValue
		err   error
	)

	i.tree.forEach(posting, func(
		value domain.Value,
		length int,
	) bool {

		switch v := value.(type) {
		case domain.FloatValue:
			count += length
			sum += domain.FloatValue(length) * v
		case domain.IntValue:
			count += length
			sum += domain.FloatValue(length) * domain.FloatValue(v)
		default:
			err = fmt.Errorf(
				"aggregate value %T: %w",
				v,
				common.ErrUnsupportedType,
			)
			return false
		}

		return count < posting.Len()
	})

	if err != nil {
		return 0, 0, err
	}

	if count == 0 {
		return 0, 0, common.ErrRecordNotFound
	}
	return sum, count, nil
}
