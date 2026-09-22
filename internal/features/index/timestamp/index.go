package timestamp

import (
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/set"
)

type Index struct {
	tree *rbTree
}

func (i *Index) Get(key time.Time) (*set.Set, bool) {
	return i.tree.Find(key)
}

func (i *Index) Range(from, to time.Time) ([]domain.RecordData, bool) {
	return i.tree.Range(from, to)
}

func (i *Index) Remove(key time.Time) bool {
	return i.tree.Remove(key)
}

func (i *Index) Min() *set.Set {
	if i.tree.Len() == 0 {
		return nil
	}
	if i.tree.Len() == 1 {
		return i.tree.root.records
	}

	minNode := i.tree.Min()

	return minNode.records
}

func (i *Index) Max() *set.Set {
	if i.tree.Len() == 0 {
		return nil
	}
	if i.tree.Len() == 1 {
		return i.tree.root.records
	}

	maxNode := i.tree.Max()

	return maxNode.records
}

func (i *Index) Len() int {
	return i.tree.elemsCount
}

func New(accuracy time.Duration) *Index {
	return &Index{
		tree: newRBTree(accuracy),
	}
}

func (i *Index) Add(key time.Time, value domain.RecordData) {
	if err := i.tree.Insert(key, value); err != nil {
		// залогируем fmt.Errorf("time index add: %w", err))
	}
}

func (i *Index) Delete(key time.Time, value domain.RecordData) bool {
	return i.tree.Delete(key, value)
}

func (i *Index) Clear() {
	i.tree.Clear()
}
