package ranged

import (
	"errors"
	"math"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/common"
	"fishyAHP/LogParser.git/internal/features/index/structs"
)

type rbTree[K common.Key] struct {
	root *node[K]

	// t.compare returns
	// -1 if k1 less than k2
	// 0 if k1 equal to k2
	// 1 if k1 more than k2
	// it need to find place for insert in rb-tree
	compare func(K, K) int

	elemsCount int
	nodesCount int
}

func newRBTree[K common.Key](comparator func(K, K) int) *rbTree[K] {
	return &rbTree[K]{
		compare: comparator,
	}
}

func (t *rbTree[K]) insert(
	key K,
	value domain.RecordID,
) (err error) {
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

func (t *rbTree[K]) fixInsert(n *node[K]) {
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

func (t *rbTree[K]) leftRotate(child *node[K]) {
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

func (t *rbTree[K]) rightRotate(n *node[K]) {
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

// find return []domain.RecordID,
// because if it will return domain.RecordID, it changes
// from O(log n) to O(n). Also this func return bool which means
// if true, it founded key, another not yet.
func (t *rbTree[K]) find(key K) (*structs.PostingList, bool) {
	cur := t.root

	for cur != nil {
		cmp := t.compare(key, cur.key)

		switch cmp {
		case 1:
			cur = cur.right
		case -1:
			cur = cur.left
		default:
			return cur.
				records.
				toPosting(), true
		}
	}

	return nil, false
}

func (t *rbTree[K]) innerRange(from, to *localBound[K]) ([]domain.RecordID, bool) {
	result := make([]domain.RecordID, 0)
	result = t.rangeSearch(t.root, from, to, result)

	if len(result) == 0 {
		return nil, false
	}
	return result, true
}

func (t *rbTree[K]) rangeSearch(
	n *node[K],
	from, to *localBound[K],
	res []domain.RecordID,
) []domain.RecordID {
	if n == nil {
		return res
	}

	if from == nil || t.compare(n.key, from.value) > 0 {
		res = t.rangeSearch(n.left, from, to, res)
	}

	if t.inBoundPeriod(n, from, to) {
		res = append(res, n.records.slice()...)
	}

	if to == nil || t.compare(n.key, to.value) < 0 {
		res = t.rangeSearch(n.right, from, to, res)
	}

	return res
}

func (t *rbTree[K]) inBoundPeriod(n *node[K], left, right *localBound[K]) bool {
	inLeft := true
	if left != nil {
		cmp := t.compare(n.key, left.value)

		if left.inclusive {
			inLeft = cmp >= 0
		} else {
			inLeft = cmp > 0
		}
	}

	inRight := true
	if right != nil {
		cmp := t.compare(n.key, right.value)

		if right.inclusive {
			inRight = cmp <= 0
		} else {
			inRight = cmp < 0
		}
	}

	return inLeft && inRight
}

func (t *rbTree[K]) len() int {
	return t.elemsCount
}

func (t *rbTree[K]) height() int {
	return int(
		math.Ceil(
			2 * math.Log2(
				float64(t.nodesCount+1),
			),
		),
	)
}

func (t *rbTree[K]) min() *node[K] {
	cur := t.root

	for cur.left != nil {
		cur = cur.left
	}

	return cur
}

func (t *rbTree[K]) max() *node[K] {
	cur := t.root

	for cur.right != nil {
		cur = cur.right
	}

	return cur
}

func (t *rbTree[K]) clear() {
	t.root = nil
	t.elemsCount = 0
	t.nodesCount = 0
}

func (t *rbTree[K]) remove(key K, value domain.RecordID) bool {
	return false
}

type direction uint8

const (
	ascending direction = iota
	descending
)

func (t *rbTree[K]) extremum(
	posting *structs.PostingList,
	direction direction,
) (domain.Value, error) {
	stack := make([]*node[K], 0, t.height())
	current := t.root

	for current != nil ||
		len(stack) != 0 {

		for current != nil {
			stack = append(stack, current)

			switch direction {
			case ascending:
				current = current.left
			case descending:
				current = current.right
			}
		}

		current = stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if structs.IsListIntersects(
			current.records.toPosting(),
			posting,
		) {
			return current.key, nil
		}

		switch direction {
		case ascending:
			current = current.right
		case descending:
			current = current.left
		}
	}

	return nil, common.ErrRecordNotFound
}

func (t *rbTree[K]) forEach(
	posting *structs.PostingList,
	visit func(
		value domain.Value,
		count int,
	) bool) {
	queue := append(
		make([]*node[K], 0, t.height()),
		t.root,
	)

	for len(queue) > 0 {
		front := queue[0]
		queue = queue[1:]

		matches := structs.CountListIntersections(
			front.records.toPosting(),
			posting,
		)

		if !visit(front.key, matches) {
			return
		}

		if front.left != nil {
			queue = append(queue, front.left)
		}
		if front.right != nil {
			queue = append(queue, front.right)
		}
	}
}
