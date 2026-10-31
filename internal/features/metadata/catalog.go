package metadata

import (
	"fmt"
	"math"
	"sort"
	"sync"

	"fishyAHP/LogParser.git/internal/core/binaryio"
)

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

func (f *FieldsCatalog) MarshalBinary() ([]byte, error) {
	f.mtx.RLock()
	defer f.mtx.RUnlock()

	if uint64(len(f.catalog)) > math.MaxUint32 {
		return nil, fmt.Errorf(
			"too many objects in catalog: %d",
			len(f.catalog),
		)
	}

	encoder, err := binaryio.NewEncoder(4)
	if err != nil {
		return nil, fmt.Errorf(
			"create catalog encoder: %w",
			err,
		)
	}

	encoder.Uint32(uint32(len(f.catalog)))

	keys := make([]string, 0, len(f.catalog))
	for key := range f.catalog {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		if len(key) > math.MaxUint16 {
			return nil, fmt.Errorf(
				"field name too long: %d bytes",
				len(key),
			)
		}

		encoder.Uint16(uint16(len(key)))
		encoder.String(key)
		encoder.Uint64(f.catalog[key])
	}

	return encoder.Result(), nil
}

func (f *FieldsCatalog) UnmarshalBinary(data []byte) error {
	decoder := binaryio.NewDecoder(data)

	elemsCount, err := decoder.Uint32()
	if err != nil {
		return fmt.Errorf(
			"decode catalog header: %w",
			err,
		)
	}

	const minEntrySize = 2 + 8

	if uint64(elemsCount) >
		uint64(decoder.Remaining()/minEntrySize) {
		return fmt.Errorf(
			"invalid catalog elements count: %d",
			elemsCount,
		)
	}

	catalog := make(map[string]uint64)

	for range elemsCount {
		keyLen, err := decoder.Uint16()
		if err != nil {
			return fmt.Errorf(
				"decode key length: %w",
				err,
			)
		}

		key, err := decoder.String(int(keyLen))
		if err != nil {
			return fmt.Errorf(
				"decode key: %w",
				err,
			)
		}

		if _, exists := catalog[key]; exists {
			return fmt.Errorf(
				"duplicate key: %q",
				key,
			)
		}

		count, err := decoder.Uint64()
		if err != nil {
			return fmt.Errorf(
				"decode field count: %w",
				err,
			)
		}

		catalog[key] = count
	}

	if err = decoder.Finish(); err != nil {
		return fmt.Errorf(
			"finish catalog decoding: %w",
			err,
		)
	}

	f.mtx.Lock()
	f.catalog = catalog
	f.mtx.Unlock()

	return nil
}
