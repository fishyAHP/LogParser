package domain

import "sync"

type FieldsCatalog struct {
	catalog map[string]uint64
	mtx     sync.RWMutex
}

func NewCatalog() *FieldsCatalog {
	return &FieldsCatalog{
		catalog: make(map[string]uint64, 5),
		mtx:     sync.RWMutex{},
	}
}

func (f *FieldsCatalog) Add(fields ...string) {
	f.mtx.Lock()
	defer f.mtx.Unlock()

	seen := make(map[string]struct{}, len(fields))

	for _, field := range fields {
		if _, exists := seen[field]; exists {
			continue
		}

		seen[field] = struct{}{}
		f.catalog[field]++
	}
}

func (f *FieldsCatalog) Remove(fields ...string) {
	f.mtx.Lock()
	defer f.mtx.Unlock()

	seen := make(map[string]struct{}, len(fields))

	for _, field := range fields {
		if _, exists := seen[field]; exists {
			continue
		}

		seen[field] = struct{}{}
		if _, ok := f.catalog[field]; ok {
			f.catalog[field]--
			if f.catalog[field] == 0 {
				delete(f.catalog, field)
			}
		}
	}
}

func (f *FieldsCatalog) Has(field string) bool {
	f.mtx.RLock()
	defer f.mtx.RUnlock()

	_, ok := f.catalog[field]
	return ok
}

func (f *FieldsCatalog) Count(field string) uint64 {
	f.mtx.RLock()
	defer f.mtx.RUnlock()

	return f.catalog[field]
}

func (f *FieldsCatalog) Clear() {
	f.mtx.Lock()
	defer f.mtx.Unlock()

	clear(f.catalog)
}
