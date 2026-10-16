package ranged

import (
	"iter"

	"fishyAHP/LogParser.git/internal/core/domain"
)

type heapItem struct {
	id      domain.RecordID
	records *records
	pos     int
}

type mergeHeap []heapItem

func (m *mergeHeap) Swap(i, j int) {
	(*m)[i], (*m)[j] = (*m)[j], (*m)[i]
}

func (m *mergeHeap) push(item heapItem) {
	*m = append(*m, item)

	child := len(*m) - 1

	for child > 0 {
		parent := (child - 1) / 2

		if (*m)[parent].id <= (*m)[child].id {
			break
		}
		m.Swap(parent, child)
		child = parent
	}

}

func (m *mergeHeap) pop() heapItem {
	old := *m
	lowest := old[0]

	if len(old) == 1 {
		old[0] = heapItem{}
		*m = old[:0]
		return lowest
	}
	last := len(old) - 1
	old[0] = old[last]
	old[last] = heapItem{}
	old = old[:last]

	pos := 0

	for {
		left := 2*pos + 1
		right := 2*pos + 2

		if left >= len(old) {
			break
		}

		smallest := left

		if right < len(old) &&
			old[right].id < old[left].id {
			smallest = right
		}

		if old[pos].id <= old[smallest].id {
			break
		}

		old.Swap(pos, smallest)
		pos = smallest
	}

	*m = old
	return lowest
}

func (m *mergeHeap) len() int {
	return len(*m)
}

func merge(
	sources iter.Seq[*records],
) iter.Seq[domain.RecordID] {
	return func(yield func(domain.RecordID) bool) {
		h := make(mergeHeap, 0)

		for source := range sources {
			if source.count() == 0 {
				continue
			}

			h.push(heapItem{
				id:      source.at(0),
				records: source,
			})
		}

		var (
			last    domain.RecordID
			hasLast bool
		)

		for h.len() > 0 {
			item := h.pop()

			if !hasLast ||
				item.id != last {
				last = item.id
				hasLast = true

				if !yield(item.id) {
					return
				}
			}

			item.pos++

			if item.pos < item.records.count() {
				item.id = item.records.at(item.pos)
				h.push(item)
			}
		}
	}
}
