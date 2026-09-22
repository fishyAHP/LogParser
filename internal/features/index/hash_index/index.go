package hash_index

import (
	"math"
	"sync"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/set"
)

type Index[K comparable] struct {
	count int
	idx   map[K]*set.Set
	mtx   sync.RWMutex
}

var newSet = func(length int) int {
	return int(math.Pow(float64(length), 0.5))
}

func NewIndex[K comparable]() *Index[K] {
	return &Index[K]{
		idx: make(map[K]*set.Set),
		mtx: sync.RWMutex{},
	}
}

func (i *Index[K]) Len() int {
	i.mtx.RLock()
	defer i.mtx.RUnlock()

	return i.count
}

func (i *Index[K]) Add(key K, value domain.RecordData) {
	i.mtx.Lock()
	defer i.mtx.Unlock()

	if _, ok := i.idx[key]; !ok {
		i.idx[key] = set.NewSet(newSet(i.count))
	}
	i.idx[key].Add(value)
	i.count++
}

func (i *Index[K]) Get(key K) (*set.Set, bool) {
	i.mtx.RLock()
	defer i.mtx.RUnlock()

	s, ok := i.idx[key]
	if !ok {
		return nil, false
	}

	return s, true
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

	i.idx = make(map[K]*set.Set)
	i.count = 0
}

func (i *Index[K]) Delete(key K, value domain.RecordData) bool {
	i.mtx.Lock()
	defer i.mtx.Unlock()

	s, ok := i.idx[key]
	if !ok {
		return false
	}

	return s.Remove(value)
}
