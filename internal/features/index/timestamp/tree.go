package timestamp

import (
	"errors"
	"math"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
)

type rbTree struct {
	root       *node
	accuracy   time.Duration
	elemsCount int
	nodesCount int
}

func newRBTree(accuracy time.Duration) *rbTree {
	return &rbTree{
		accuracy: accuracy,
	}
}

// t.compare returns
// -1 if k1 less than k2
// 0 if k1 equal to k2
// 1 if k1 more than k2
// it need to find place for insert in rb-tree
func (t *rbTree) compare(k1, k2 time.Time) int {
	k1 = k1.Truncate(t.accuracy)
	k2 = k2.Truncate(t.accuracy)

	switch {
	case k1.Before(k2):
		return -1
	case k1.After(k2):
		return 1
	default:
		return 0
	}
}

func (t *rbTree) Insert(key time.Time, value domain.RecordData) (err error) {
	if time.Since(key) < 0 {
		return errors.New("key in future")
	}

	if t.elemsCount < 1 {
		t.root = newNode(key, value)
		t.nodesCount++

		t.root.color = Black
		t.elemsCount = 1
		return nil
	}

	current := t.root
	defer func() {
		if err == nil {
			t.elemsCount++
		}
	}()

	for {
		cmp := t.compare(key, current.key)

		switch cmp {
		case -1:
			if current.left != nil {
				current = current.left
			} else {
				nod := newNode(key, value)
				t.nodesCount++

				nod.parent = current
				current.left = nod
				t.fixInsert(nod)
				return
			}
		case 1:
			if current.right != nil {
				current = current.right
			} else {
				nod := newNode(key, value)
				t.nodesCount++

				nod.parent = current
				current.right = nod
				t.fixInsert(nod)
				return
			}
		default:
			if ok := current.add(value); !ok {
				return errors.New("key already exist")
			}
			return
		}
	}
}

func (t *rbTree) fixInsert(n *node) {
	parent := n.parent
	if parent == nil {
		t.root.color = Black
		return
	}
	if parent.color == Black {
		return
	}

	if parent.color == Red {
		uncle := n.uncle()
		if uncle == nil || uncle.color == Black {
			grandparent := parent.parent
			switch parent {
			case grandparent.left:
				if n == parent.left {
					t.rightRotate(parent)
					parent.color = Black
				} else {
					t.leftRotate(n)
					t.rightRotate(n)
					n.color = Black
				}

				grandparent.color = Red
			case grandparent.right:
				if n == parent.right {
					t.leftRotate(parent)
					parent.color = Black
				} else {
					t.rightRotate(n)
					t.leftRotate(n)
					n.color = Black
				}

				grandparent.color = Red
			}
		} else if uncle.color == Red {
			parent.color = Black
			uncle.color = Black
			uncle.parent.color = Red

			t.fixInsert(uncle.parent)
		}
	}
}

func (t *rbTree) leftRotate(child *node) {
	if child.parent == nil ||
		child != child.parent.right {
		return
	}

	parent := child.parent
	if parent == t.root {
		t.root = child
	}

	if child.left != nil {
		child.left.parent = parent
	}
	parent.right = child.left

	grandparent := parent.parent
	parent.parent = child
	if grandparent != nil {
		if parent == grandparent.left {
			grandparent.left = child
		} else {
			grandparent.right = child
		}
	}

	child.parent = grandparent
	child.left = parent
}

func (t *rbTree) rightRotate(n *node) {
	if n.parent == nil ||
		n != n.parent.left {
		return
	}

	parent := n.parent
	if parent == t.root {
		t.root = n
	}

	if n.right != nil {
		n.right.parent = parent
	}
	parent.left = n.right

	grandparent := parent.parent
	parent.parent = n
	if grandparent != nil {
		if parent == grandparent.left {
			grandparent.left = n
		} else {
			grandparent.right = n
		}
	}

	n.parent = grandparent
	n.right = parent
}

// Find return []domain.RecordData,
// because if it will return domain.RecordData, it changes
// from O(log n) to O(n). Also this func return bool which means
// if true, it founded key, another not yet.
func (t *rbTree) Find(key time.Time) ([]domain.RecordData, bool) {
	cur := t.root

	for cur != nil {
		cmp := t.compare(key, cur.key)

		switch cmp {
		case 1:
			cur = cur.right
		case -1:
			cur = cur.left
		default:
			return cur.records.Slice(), true
		}
	}

	return nil, false
}

func (t *rbTree) Range(from, to time.Time) ([]domain.RecordData, bool) {
	if t.compare(from, to) == 1 {
		return nil, false
	}

	result := make([]domain.RecordData, 0)

	result = t.rangeSearch(t.root, from, to, result)

	if len(result) == 0 {
		return nil, false
	}
	return result, true
}

func (t *rbTree) rangeSearch(
	n *node,
	from, to time.Time,
	res []domain.RecordData,
) []domain.RecordData {
	if n == nil {
		return res
	}

	if t.compare(n.key, from) == 1 {
		res = t.rangeSearch(n.left, from, to, res)
	}

	if t.compare(n.key, from) > -1 &&
		t.compare(n.key, to) < 1 {
		res = append(res, n.records.Slice()...)
	}

	if t.compare(n.key, to) == -1 {
		res = t.rangeSearch(n.right, from, to, res)
	}

	return res
}

func (t *rbTree) Len() int {
	return t.elemsCount
}

func (t *rbTree) Height() int {
	return int(
		2 * math.Log2(
			float64(t.nodesCount+1),
		),
	)
}

func (t *rbTree) Min() *node {
	cur := t.root

	for cur.left != nil {
		cur = cur.left
	}

	return cur
}

func (t *rbTree) Max() *node {
	cur := t.root

	for cur.right != nil {
		cur = cur.right
	}

	return cur
}

func (t *rbTree) Clear() {
	t.root = nil
	t.elemsCount = 0
	t.nodesCount = 0
}

func (t *rbTree) Remove(key time.Time) bool {
	return false
}

func (t *rbTree) Delete(key time.Time, value domain.RecordData) bool {
	return false
}
