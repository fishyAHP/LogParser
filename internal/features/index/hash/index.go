package hash

import (
	"errors"
	"math"
	"sync"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/set"
)

type Key interface {
	domain.Value
	comparable
}

type Index[K Key] struct {
	count int
	idx   map[K]*set.Set[domain.RecordData]
	mtx   sync.RWMutex
}

func New[K Key]() *Index[K] {
	return &Index[K]{
		idx: make(map[K]*set.Set[domain.RecordData]),
		mtx: sync.RWMutex{},
	}
}

var ErrInvalidType = errors.New("invalid type type")

func (i *Index[K]) Exact(value domain.Value) (*set.Set[domain.RecordData], error) {
	i.mtx.RLock()
	defer i.mtx.RUnlock()

	newVal, ok := value.(K)
	if !ok {
		return nil, ErrInvalidType
	}
	s, ok := i.idx[newVal]
	if !ok {
		return nil, nil
	}

	return s, nil
}

func (i *Index[K]) Add(value domain.Value, record domain.RecordData) error {
	i.mtx.Lock()
	defer i.mtx.Unlock()

	newVal, ok := value.(K)
	if !ok {
		return ErrInvalidType
	}
	if _, ok := i.idx[newVal]; !ok {
		i.idx[newVal] = set.New[domain.RecordData](newSet(i.count))
	}
	if i.idx[newVal].Add(record) {
		i.count++
	}
	return nil
}

var newSet = func(length int) int {
	return int(math.Pow(float64(length), 0.5))
}

func (i *Index[K]) Len() int {
	i.mtx.RLock()
	defer i.mtx.RUnlock()

	return i.count
}

func (i *Index[K]) Remove(key K) bool {
	i.mtx.Lock()
	defer i.mtx.Unlock()

	if _, ok := i.idx[key]; !ok {
		return false
	}

	i.count -= i.idx[key].Len()
	delete(i.idx, key)
	return true
}

func (i *Index[K]) Clear() {
	i.mtx.Lock()
	defer i.mtx.Unlock()

	i.idx = make(map[K]*set.Set[domain.RecordData])
	i.count = 0
}

func (i *Index[K]) Delete(key K, value domain.RecordData) bool {
	i.mtx.Lock()
	defer i.mtx.Unlock()

	s, ok := i.idx[key]
	if !ok {
		return false
	}

	if s.Remove(value) {
		i.count--
		if s.Len() == 0 {
			delete(i.idx, key)
		}
		return true
	}
	return false
}
