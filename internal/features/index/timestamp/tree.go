package timestamp

import (
	"cmp"
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

// compare returns
// -1 if k1 less than k2
// 0 if k1 equal to k2
// 1 if k1 more than k2
// it need to find place for insert in rb-tree
func compare(k1, k2 time.Time) int {
	m1 := k1.Minute()
	m2 := k2.Minute()

	return cmp.Compare(m1, m2)
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
		cmpr := compare(key, current.key)

		switch cmpr {
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
				if n == parent.right {
					t.leftRotate(n)
				}
				t.rightRotate(n)

				grandparent.color = Red
				if n == parent.left {
					parent.color = Black
				} else {
					n.color = Black
				}
			case grandparent.right:
				if n == parent.left {
					t.rightRotate(n)
				}
				t.leftRotate(n)

				grandparent.color = Red
				if n == parent.right {
					parent.color = Black
				} else {
					n.color = Black
				}
			}
		} else {
			parent.color = Black
			uncle.color = Black
			uncle.parent.color = Red

			t.fixInsert(uncle.parent)
		}
	}
}

func (t *rbTree) leftRotate(n *node) {
	if n.parent == nil ||
		n != n.parent.right {
		return
	}

	parent := n.parent
	if parent == t.root {
		t.root = n
	}

	if n.left != nil {
		n.left.parent = parent
	}
	parent.right = n.left

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
	n.left = parent
}

func (t *rbTree) rightRotate(n *node) {
	if n != n.parent.left {
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
