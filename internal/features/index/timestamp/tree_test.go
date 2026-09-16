package timestamp

import (
	"testing"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
)

func TestAddFirstNode(t *testing.T) {
	tree := &rbTree{}

	key := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	value := domain.RecordData{}

	if err := tree.Add(key, value); err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	if tree.root == nil {
		t.Fatal("root is nil")
	}

	if tree.root.color != Black {
		t.Fatalf("root color = %v, want Black", tree.root.color)
	}

	if tree.count != 1 {
		t.Fatalf("count = %d, want 1", tree.count)
	}
}

func TestAddBST(t *testing.T) {
	tree := &rbTree{}

	keys := []time.Time{
		time.Date(2026, 1, 2, 12, 30, 0, 0, time.UTC),
		time.Date(2026, 1, 2, 12, 29, 0, 0, time.UTC),
		time.Date(2026, 1, 2, 12, 31, 0, 0, time.UTC),
	}

	for _, key := range keys {
		if err := tree.Add(key, domain.RecordData{}); err != nil {
			t.Fatalf("Index() error = %v", err)
		}
	}

	assertKeyEqual(t, tree.root.key, keys[0])
	assertKeyEqual(t, tree.root.left.key, keys[1])
	assertKeyEqual(t, tree.root.right.key, keys[2])
}

func TestAddDuplicateKey(t *testing.T) {
	tree := &rbTree{}

	// Все три значения находятся в одной минуте.
	keys := []time.Time{
		time.Date(2026, 1, 1, 12, 30, 1, 0, time.UTC),
		time.Date(2026, 1, 1, 12, 30, 25, 0, time.UTC),
		time.Date(2026, 1, 1, 12, 30, 59, 0, time.UTC),
	}

	for _, key := range keys {
		if err := tree.Add(key, domain.RecordData{}); err != nil {
			t.Fatal(err)
		}
	}

	if tree.count != 3 {
		t.Fatalf("count = %d, want 3", tree.count)
	}

	if len(tree.root.records) != 3 {
		t.Fatalf("records = %d, want 3", len(tree.root.records))
	}

	if tree.root.left != nil || tree.root.right != nil {
		t.Fatal("timestamps from the same minute created another node")
	}
}

func TestDifferentSecondsSameMinute(t *testing.T) {
	tree := &rbTree{}

	key1 := time.Date(2026, 1, 1, 12, 30, 1, 0, time.UTC)
	key2 := time.Date(2026, 1, 1, 12, 30, 59, 0, time.UTC)

	if err := tree.Add(key1, domain.RecordData{}); err != nil {
		t.Fatal(err)
	}

	if err := tree.Add(key2, domain.RecordData{}); err != nil {
		t.Fatal(err)
	}

	if tree.root == nil {
		t.Fatal("root is nil")
	}

	if len(tree.root.records) != 2 {
		t.Fatalf("records = %d, want 2", len(tree.root.records))
	}

	if tree.root.left != nil || tree.root.right != nil {
		t.Fatal("same-minute timestamps created multiple nodes")
	}
}

func TestFixInsertLL(t *testing.T) {
	tree := &rbTree{}

	keys := []time.Time{
		time.Date(2026, 1, 1, 12, 32, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 12, 31, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 12, 30, 0, 0, time.UTC),
	}

	for _, key := range keys {
		if err := tree.Add(key, domain.RecordData{}); err != nil {
			t.Fatal(err)
		}
	}

	assertTreeShape(t, tree,
		time.Date(2026, 1, 1, 12, 31, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 12, 30, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 12, 32, 0, 0, time.UTC),
	)
}

func TestFixInsertRR(t *testing.T) {
	tree := &rbTree{}

	keys := []time.Time{
		time.Date(2026, 1, 1, 12, 30, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 12, 31, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 12, 32, 0, 0, time.UTC),
	}

	for _, key := range keys {
		if err := tree.Add(key, domain.RecordData{}); err != nil {
			t.Fatal(err)
		}
	}

	assertTreeShape(t, tree,
		time.Date(2026, 1, 1, 12, 31, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 12, 30, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 12, 32, 0, 0, time.UTC),
	)
}

func TestFixInsertLR(t *testing.T) {
	tree := &rbTree{}

	keys := []time.Time{
		time.Date(2026, 1, 1, 12, 32, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 12, 30, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 12, 31, 0, 0, time.UTC),
	}

	for _, key := range keys {
		if err := tree.Add(key, domain.RecordData{}); err != nil {
			t.Fatal(err)
		}
	}

	assertTreeShape(t, tree,
		time.Date(2026, 1, 1, 12, 31, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 12, 30, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 12, 32, 0, 0, time.UTC),
	)
}

func TestFixInsertRL(t *testing.T) {
	tree := &rbTree{}

	keys := []time.Time{
		time.Date(2026, 1, 1, 12, 30, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 12, 32, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 12, 31, 0, 0, time.UTC),
	}

	for _, key := range keys {
		if err := tree.Add(key, domain.RecordData{}); err != nil {
			t.Fatal(err)
		}
	}

	assertTreeShape(t, tree,
		time.Date(2026, 1, 1, 12, 31, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 12, 30, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 12, 32, 0, 0, time.UTC),
	)
}

func assertTreeShape(
	t *testing.T,
	tree *rbTree,
	root, left, right time.Time,
) {
	t.Helper()

	if tree.root == nil {
		t.Fatal("root is nil")
	}

	assertKeyEqual(t, tree.root.key, root)

	if tree.root.left == nil {
		t.Fatal("left child is nil")
	}

	assertKeyEqual(t, tree.root.left.key, left)

	if tree.root.right == nil {
		t.Fatal("right child is nil")
	}

	assertKeyEqual(t, tree.root.right.key, right)

	if tree.root.color != Black {
		t.Fatal("root is not black")
	}
}

func assertKeyEqual(t *testing.T, actual, expected time.Time) {
	t.Helper()

	actual = actual.Truncate(time.Minute)
	expected = expected.Truncate(time.Minute)

	if !actual.Equal(expected) {
		t.Fatalf(
			"key = %v, want %v",
			actual,
			expected,
		)
	}
}

func TestRandomInsertions(t *testing.T) {
	tree := &rbTree{}

	start := time.Date(
		2026,
		1,
		1,
		0,
		0,
		0,
		0,
		time.UTC,
	)

	for i := range 1000 {
		key := start.Add(time.Duration(i) * time.Minute)

		if err := tree.Add(key, domain.RecordData{}); err != nil {
			t.Fatal(err)
		}

		t.Logf("after insert %d: %v", i, key)

		assertRBTreeValid(t, tree)
	}
}

func assertRBTreeValid(t *testing.T, tree *rbTree) {
	t.Helper()

	if tree.root == nil {
		if tree.count != 0 {
			t.Fatalf("tree is empty, but count = %d", tree.count)
		}

		return
	}

	if tree.root.color != Black {
		t.Fatalf("root is %v, want Black", tree.root.color)
	}

	if tree.root.parent != nil {
		t.Fatal("root has a parent")
	}

	visited := make(map[*node]bool)

	validateNode(
		t,
		tree.root,
		nil,
		nil,
		visited,
	)
}

func validateNode(
	t *testing.T,
	n *node,
	min *time.Time,
	max *time.Time,
	visited map[*node]bool,
) (blackHeight int, nodes int) {
	t.Helper()

	if n == nil {
		return 1, 0
	}

	if visited[n] {
		t.Fatalf("cycle detected at node %v", n.key)
	}

	visited[n] = true

	// Нормализуем границы до минут,
	// потому что так работает comparator.
	key := n.key.Truncate(time.Minute)

	if min != nil {
		minKey := min.Truncate(time.Minute)

		if !key.After(minKey) {
			t.Fatalf(
				"BST violation: key %v <= min %v",
				n.key,
				*min,
			)
		}
	}

	if max != nil {
		maxKey := max.Truncate(time.Minute)

		if !key.Before(maxKey) {
			t.Fatalf(
				"BST violation: key %v >= max %v",
				n.key,
				*max,
			)
		}
	}

	if n.left != nil && n.left.parent != n {
		t.Fatalf(
			"wrong parent for left child %v of %v",
			n.left.key,
			n.key,
		)
	}

	if n.right != nil && n.right.parent != n {
		t.Fatalf(
			"wrong parent for right child %v of %v",
			n.right.key,
			n.key,
		)
	}

	if n.color == Red {
		if n.left != nil && n.left.color == Red {
			t.Fatalf(
				"red node %v has red left child %v",
				n.key,
				n.left.key,
			)
		}

		if n.right != nil && n.right.color == Red {
			t.Fatalf(
				"red node %v has red right child %v",
				n.key,
				n.right.key,
			)
		}
	}

	leftBlackHeight, leftNodes := validateNode(
		t,
		n.left,
		min,
		&key,
		visited,
	)

	rightBlackHeight, rightNodes := validateNode(
		t,
		n.right,
		&key,
		max,
		visited,
	)

	if leftBlackHeight != rightBlackHeight {
		t.Fatalf(
			"black-height violation at %v: left = %d, right = %d",
			n.key,
			leftBlackHeight,
			rightBlackHeight,
		)
	}

	blackHeight = leftBlackHeight

	if n.color == Black {
		blackHeight++
	}

	nodes = 1 + leftNodes + rightNodes

	return blackHeight, nodes
}
