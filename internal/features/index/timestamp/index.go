package timestamp

import (
	"fmt"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
)

type Index struct {
	tree *rbTree
}

func New(accuracy time.Duration) *Index {
	return &Index{
		tree: newRBTree(accuracy),
	}
}

func (i *Index) Add(key time.Time, value domain.RecordData) error {
	if err := i.tree.Add(key, value); err != nil {
		return fmt.Errorf("time index add: %w", err)
	}
	return nil
}

func (i *Index) Clear() {
	i.tree.Clear()
}
