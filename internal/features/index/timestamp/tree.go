package timestamp

import (
	"errors"
	"math"
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

// compare returns
// -1 if k1 less than k2
// 0 if k1 equal to k2
// 1 if k1 more than k2
// it need to find place for insert in rb-tree
func compare(k1, k2 time.Time) int {
	k1 = k1.Truncate(time.Minute)
	k2 = k2.Truncate(time.Minute)

	switch {
	case k1.Before(k2):
		return -1
	case k1.After(k2):
		return 1
	default:
		return 0
	}
}

func (t *rbTree) Add(key time.Time, value domain.RecordData) (err error) {
	if time.Since(key) < 0 {
		return errors.New("key in future")
	}

	if t.count < 1 {
		t.root = newNode(key, value)
		t.root.color = Black
		t.count = 1
		return nil
	}

	current := t.root
	defer func() {
		if err == nil {
			t.count++
		}
	}()

	for {
		cmp := compare(key, current.key)

		switch cmp {
		case -1:
			if current.left != nil {
				current = current.left
			} else {
				nod := newNode(key, value)
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
				nod.parent = current
				current.right = nod
				t.fixInsert(nod)
				return
			}
		default:
			current.add(value)
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

// Find return []domain.RecordData
// because if it will return domain.RecordData it changes
// from O(log n) to O(n). Also this func return bool which mean
// if true, it founded key, another not yet.
func (t *rbTree) Find(key time.Time) ([]domain.RecordData, bool) {
	cur := t.root

	for cur != nil {
		cmp := compare(key, cur.key)

		switch cmp {
		case 1:
			cur = cur.right
		case -1:
			cur = cur.left
		default:
			return cur.records, true
		}
	}

	return nil, false
}

func (t *rbTree) Range(from, to time.Time) ([]domain.RecordData, bool) {
	cur := t.root
	ans := make([]domain.RecordData, 0, int(math.Log2(float64(t.count)+1)))

findLoop:
	for cur != nil {
		cmp := compare(from, cur.key)

		switch cmp {
		case 1:
			cur = cur.right
		case -1:
			if cur == cur.parent.left {
				break findLoop
			}
			cur = cur.left
		default:
			ans = append(ans, cur.records...)
			break findLoop
		}
	}

	cur = cur.parent
	for {
		localCur := cur

		for localCur != nil {
			if compare(from, localCur.key) == -1 &&
				compare(localCur.key, to) == -1 {

			}
		}
	}

	return ans, true
}
