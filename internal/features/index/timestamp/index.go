package timestamp

import (
	"fmt"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
)

type Index struct {
	tree *rbTree
}

func New() *Index {
	return &Index{
		tree: newRBTree(),
	}
}

func (i *Index) Add(key time.Time, value domain.RecordData) error {
	if err := i.tree.Add(key, value); err != nil {
		return fmt.Errorf("time index add: %w", err)
	}
	return nil
}
