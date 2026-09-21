package timestamp

import (
	"fmt"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
)

type Index struct {
	tree *rbTree
}

func (i *Index) Find(key time.Time) ([]domain.RecordData, bool) {
	return i.tree.Find(key)
}

func (i *Index) Range(from, to time.Time) ([]domain.RecordData, bool) {
	return i.tree.Range(from, to)
}

func (i *Index) Remove(key time.Time) bool {
	return i.tree.Remove(key)
}

func (i *Index) Min() []domain.RecordData {
	if i.tree.Len() <= 1 {
		return i.tree.root.records.Slice()
	}

	minNode := i.tree.Min()

	return minNode.records.Slice()
}

func (i *Index) Max() []domain.RecordData {
	if i.tree.Len() <= 1 {
		return i.tree.root.records.Slice()
	}

	maxNode := i.tree.Max()

	return maxNode.records.Slice()
}

func (i *Index) Len() int {
	return i.tree.elemsCount
}

func New(accuracy time.Duration) *Index {
	return &Index{
		tree: newRBTree(accuracy),
	}
}

func (i *Index) Add(key time.Time, value domain.RecordData) error {
	if err := i.tree.Insert(key, value); err != nil {
		return fmt.Errorf("time index add: %w", err)
	}
	return nil
}

func (i *Index) Clear() {
	i.tree.Clear()
}
