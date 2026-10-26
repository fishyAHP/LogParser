package ranged

import (
	"fmt"
	"iter"

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
	from, to *domain.Bound,
) (iter.Seq[domain.RecordID], error) {
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

	return i.tree.innerRange(fromBound, toBound), nil
}

func (i *Index[K]) Exact(
	value domain.Value,
) (iter.Seq[domain.RecordID], error) {
	key, ok := value.(K)
	if !ok {
		return nil, common.ErrInvalidType
	}

	iterator, ok := i.tree.find(key)
	if !ok {
		return nil, common.ErrRecordNotFound
	}
	return iterator, nil
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

func (i *Index[K]) minNode() iter.Seq[domain.RecordID] {
	if i.tree.root == nil {
		return nil
	}

	return i.tree.min().
		records.
		iterator()
}

func (i *Index[K]) maxNode() iter.Seq[domain.RecordID] {
	if i.tree.root == nil {
		return nil
	}

	return i.tree.max().
		records.
		iterator()
}

func (i *Index[K]) Len() int {
	return i.tree.elemsCount
}

func (i *Index[K]) Clear() {
	i.tree.clear()
}

func (i *Index[K]) Min(
	source iter.Seq[domain.RecordID],
) (domain.Value, error) {
	return i.tree.extremum(
		toPosting(source),
		ascending,
	)
}

func (i *Index[K]) Max(
	source iter.Seq[domain.RecordID],
) (domain.Value, error) {
	return i.tree.extremum(
		toPosting(source),
		descending,
	)
}

func (i *Index[K]) Avg(
	source iter.Seq[domain.RecordID],
) (domain.Value, error) {
	sum, count, err := i.sumAndCount(source)
	if err != nil {
		return nil, err
	}

	return sum / domain.FloatValue(count), nil
}

func (i *Index[K]) Sum(
	source iter.Seq[domain.RecordID],
) (domain.Value, error) {
	sum, _, err := i.sumAndCount(source)
	if err != nil {
		return nil, err
	}
	return sum, nil
}

func (i *Index[K]) sumAndCount(
	source iter.Seq[domain.RecordID],
) (domain.FloatValue, int, error) {
	var (
		count int
		sum   domain.FloatValue
		err   error
	)

	i.tree.forEach(
		toPosting(source),
		func(
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

			return true
		})

	if err != nil {
		return 0, 0, err
	}

	if count == 0 {
		return 0, 0, common.ErrRecordNotFound
	}
	return sum, count, nil
}

func toPosting(
	source iter.Seq[domain.RecordID],
) *structs.PostingList {
	post := structs.NewPostingLists(0)

	for id := range source {
		post.Add(id)
	}

	return post
}
