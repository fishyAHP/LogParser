package hash

import (
	"sync"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/common"
	"fishyAHP/LogParser.git/internal/features/index/structs"
)

type Index[K common.Key] struct {
	count int
	idx   map[K]*structs.PostingList
	mtx   sync.RWMutex
}

func New[K common.Key]() *Index[K] {
	return &Index[K]{
		idx: make(map[K]*structs.PostingList),
		mtx: sync.RWMutex{},
	}
}

func (i *Index[K]) Exact(
	value domain.Value,
) (*structs.PostingList, error) {
	i.mtx.RLock()
	defer i.mtx.RUnlock()

	newVal, ok := value.(K)
	if !ok {
		return nil, common.ErrInvalidType
	}
	s, ok := i.idx[newVal]
	if !ok {
		return nil, common.ErrRecordNotFound
	}

	return s, nil
}

func (i *Index[K]) Add(
	value domain.Value,
	record domain.RecordID,
) error {
	i.mtx.Lock()
	defer i.mtx.Unlock()

	newVal, ok := value.(K)
	if !ok {
		return common.ErrInvalidType
	}
	if _, ok := i.idx[newVal]; !ok {
		i.idx[newVal] = structs.NewPostingLists(1)
	}
	if i.idx[newVal].Add(record) {
		i.count++
	}
	return nil
}

func (i *Index[K]) Remove(
	value domain.Value,
	data domain.RecordID,
) error {
	i.mtx.Lock()
	defer i.mtx.Unlock()

	key, ok := value.(K)
	if !ok {
		return common.ErrInvalidType
	}

	s, ok := i.idx[key]
	if !ok {
		return common.ErrInvalidType
	}
	if s.Remove(data) {
		i.count--
		if s.Len() == 0 {
			delete(i.idx, key)
		}
		return nil
	}
	return common.ErrRecordNotFound
}

func (i *Index[K]) Len() int {
	i.mtx.RLock()
	defer i.mtx.RUnlock()

	return i.count
}

func (i *Index[K]) Clear() {
	i.mtx.Lock()
	defer i.mtx.Unlock()

	i.idx = make(map[K]*structs.PostingList)
	i.count = 0
}

func (i *Index[K]) Group(
	posting *structs.PostingList,
) (common.Groups, error) {
	groups := make(common.Groups, 0, len(i.idx))

	for k, v := range i.idx {
		res := structs.IntersectionLists(posting, v)

		if res.Len() > 0 {
			groups = append(groups, common.Group{
				Value:   k,
				Records: res,
			})
		}
	}

	if len(groups) == 0 {
		return nil, common.ErrRecordNotFound
	}
	return groups, nil
}
