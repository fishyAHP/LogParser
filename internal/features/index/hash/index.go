package hash

import (
	"sync"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/common"
	"fishyAHP/LogParser.git/internal/features/index/set"
)

type Index[K common.Key] struct {
	count int
	idx   map[K]*set.Set[domain.RecordData]
	mtx   sync.RWMutex
}

func New[K common.Key]() *Index[K] {
	return &Index[K]{
		idx: make(map[K]*set.Set[domain.RecordData]),
		mtx: sync.RWMutex{},
	}
}

func (i *Index[K]) Exact(
	value domain.Value,
) (*set.Set[domain.RecordData], error) {
	i.mtx.RLock()
	defer i.mtx.RUnlock()

	newVal, ok := value.(K)
	if !ok {
		return nil, common.ErrInvalidType
	}
	s, ok := i.idx[newVal]
	if !ok {
		return nil, common.ErrNotFoundRecord
	}

	return s, nil
}

func (i *Index[K]) Add(
	value domain.Value,
	record domain.RecordData,
) error {
	i.mtx.Lock()
	defer i.mtx.Unlock()

	newVal, ok := value.(K)
	if !ok {
		return common.ErrInvalidType
	}
	if _, ok := i.idx[newVal]; !ok {
		i.idx[newVal] = set.New[domain.RecordData](1)
	}
	if i.idx[newVal].Add(record) {
		i.count++
	}
	return nil
}

func (i *Index[K]) Remove(
	value domain.Value,
	data domain.RecordData,
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
	return common.ErrNotFoundRecord
}

func (i *Index[K]) Len() int {
	i.mtx.RLock()
	defer i.mtx.RUnlock()

	return i.count
}

func (i *Index[K]) Clear() {
	i.mtx.Lock()
	defer i.mtx.Unlock()

	i.idx = make(map[K]*set.Set[domain.RecordData])
	i.count = 0
}
