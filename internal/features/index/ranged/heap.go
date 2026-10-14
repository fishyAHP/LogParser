package ranged

import (
	"container/heap"
	"iter"

	"fishyAHP/LogParser.git/internal/core/domain"
)

var _ heap.Interface = (*mergeHeap)(nil)

type mergeHeap []mergeItem

type mergeItem struct {
	id   domain.RecordID
	next func() (domain.RecordID, bool)
}

func (m mergeHeap) Len() int {
	return len(m)
}

func (m mergeHeap) Less(i, j int) bool {
	return m[i].id < m[j].id
}

func (m mergeHeap) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

func (m *mergeHeap) Push(item any) {
	*m = append(*m, item.(mergeItem))
}

func (m *mergeHeap) Pop() any {
	old := *m
	n := len(old)

	item := old[n-1]
	*m = old[:n-1]

	return item
}

func merge(
	sources iter.Seq[iter.Seq[domain.RecordID]],
) iter.Seq[domain.RecordID] {
	return func(yield func(domain.RecordID) bool) {
		h := make(mergeHeap, 0)

		var stops []func()
		defer func() {
			for _, stop := range stops {
				stop()
			}
		}()

		for source := range sources {
			next, stop := iter.Pull(source)
			stops = append(stops, stop)

			id, ok := next()
			if !ok {
				continue
			}
			heap.Push(&h, mergeItem{
				id:   id,
				next: next,
			})
		}

		var (
			last    domain.RecordID
			hasLast bool
		)

		for h.Len() > 0 {
			item := heap.Pop(&h).(mergeItem)

			if !hasLast ||
				item.id != last {
				last = item.id
				hasLast = true

				if !yield(item.id) {
					return
				}
			}

			nextID, ok := item.next()
			if ok {
				item.id = nextID
				heap.Push(&h, item)
			}
		}
	}
}
