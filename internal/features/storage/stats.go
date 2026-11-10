package storage

import (
	"fmt"
	"os"
	"strings"
)

type Stats struct {
	Records         uint64
	Segments        int
	DiskBytes       int64
	PointerCount    int
	PointerCapacity int
}

func (s *Storage) Stats() (Stats, error) {
	s.writeMtx.Lock()
	defer s.writeMtx.Unlock()

	if s.closed {
		return Stats{}, ErrClosedStorage
	}

	stats := Stats{
		Records:         s.recordsCount,
		PointerCount:    len(s.records),
		PointerCapacity: cap(s.records),
	}

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return Stats{}, fmt.Errorf(
			"read storage directory: %w",
			err,
		)
	}

	for _, entry := range entries {
		if !strings.HasPrefix(
			entry.Name(),
			"segment-",
		) {
			continue
		}

		if !entry.Type().IsRegular() {
			return Stats{}, fmt.Errorf(
				"invalid segment: %q",
				entry.Name(),
			)
		}

		info, err := entry.Info()
		if err != nil {
			return Stats{}, fmt.Errorf(
				"stat segment %q: %w",
				entry.Name(),
				err,
			)
		}

		stats.Segments++
		stats.DiskBytes += info.Size()
	}

	return stats, nil
}
