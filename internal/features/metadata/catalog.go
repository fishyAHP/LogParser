package metadata

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
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

	var buf bytes.Buffer

	if len(f.catalog) > math.MaxUint32 {
		return nil, fmt.Errorf(
			"too many objects in catalog: %d",
			len(f.catalog),
		)
	}

	if err := binary.Write(
		&buf,
		binary.BigEndian,
		uint32(len(f.catalog)),
	); err != nil {
		return nil, fmt.Errorf(
			"write fields count: %w",
			err,
		)
	}

	keys := make([]string, 0, len(f.catalog))
	for key := range f.catalog {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		count := f.catalog[key]

		if len(key) > math.MaxUint16 {
			return nil, fmt.Errorf(
				"field name too long: %d bytes",
				len(key),
			)
		}

		if err := binary.Write(
			&buf,
			binary.BigEndian,
			uint16(len(key)),
		); err != nil {
			return nil, fmt.Errorf(
				"write key length: %w",
				err,
			)
		}

		if _, err := buf.Write([]byte(key)); err != nil {
			return nil, fmt.Errorf(
				"write key: %w",
				err,
			)
		}

		if err := binary.Write(
			&buf,
			binary.BigEndian,
			count,
		); err != nil {
			return nil, fmt.Errorf(
				"write count: %w",
				err,
			)
		}
	}

	return buf.Bytes(), nil
}

func (f *FieldsCatalog) UnmarshalBinary(
	data []byte,
) error {

	if len(data) < 4 {
		return errors.New(
			"catalog header truncated",
		)
	}

	elemsCount := binary.BigEndian.Uint32(
		data[:4],
	)

	if uint64(elemsCount) > uint64(len(data)-4)/10 {
		return fmt.Errorf(
			"invalid catalog elements count: %d",
			elemsCount,
		)
	}

	catalog := make(map[string]uint64)

	offset := 4

	for range elemsCount {
		if len(data)-offset < 2 {
			return errors.New(
				"key length truncated",
			)
		}

		keyLen := binary.BigEndian.Uint16(
			data[offset : offset+2],
		)

		offset += 2

		if int(keyLen) > len(data)-offset {
			return errors.New(
				"key data truncated",
			)
		}

		key := string(data[offset : offset+int(keyLen)])

		if _, ok := catalog[key]; ok {
			return fmt.Errorf(
				"duplicate key: %q",
				key,
			)
		}

		offset += int(keyLen)

		if len(data)-offset < 8 {
			return errors.New(
				"field count truncated",
			)
		}

		count := binary.BigEndian.Uint64(
			data[offset : offset+8],
		)
		offset += 8

		catalog[key] = count
	}

	if offset != len(data) {
		return fmt.Errorf(
			"unexpected trailing data: %d bytes",
			len(data)-offset,
		)
	}

	f.mtx.Lock()
	f.catalog = catalog
	f.mtx.Unlock()

	return nil
}
