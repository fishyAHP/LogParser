package timestamp

import (
	"errors"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
)

type rbTree struct {
	root  *node
	count int
}

func newRBTree() *rbTree {
	return &rbTree{}
}

func (t *rbTree) Add(key time.Time, value domain.RecordData) error {
	if time.Since(key) < 0 {
		return errors.New("key in future")
	}

	if t.count < 1 {
		t.root = newNode(key, value)
		t.count = 1
		return nil
	}

	current := t.root
	keyMinute := key.Minute()
	for current.left != nil && current.right != nil {
		curMinute := current.key.Minute()

		switch {
		case keyMinute < curMinute:
			current = current.left
		case keyMinute > curMinute:
			current = current.right
		default:
			current.records = append(current.records, value)
			return nil
		}
	}

	if current.parent == nil {

	}
}
