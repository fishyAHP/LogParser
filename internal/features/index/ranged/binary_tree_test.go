package ranged

import (
	"bytes"
	"cmp"
	"fmt"
	"slices"
	"testing"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/common"
)

func assertRBTreeValid[K common.Key](
	t *testing.T,
	tree *rbTree[K],
	compare func(K, K) int,
) {
	t.Helper()

	if tree.root == nil {
		if tree.nodesCount != 0 || tree.elemsCount != 0 {
			t.Fatalf(
				"empty tree has counters: nodes=%d, records=%d",
				tree.nodesCount,
				tree.elemsCount,
			)
		}
		return
	}

	if tree.root.parent != nil {
		t.Fatal("root has a parent")
	}

	if tree.root.color != Black {
		t.Fatal("root must be black")
	}

	var nodes uint64
	var elements uint64

	var walk func(
		n *node[K],
		parent *node[K],
		lower, upper *K,
	) int

	walk = func(
		n *node[K],
		parent *node[K],
		lower, upper *K,
	) int {
		if n == nil {
			// NIL leaves are black.
			return 1
		}

		nodes++
		elements += uint64(n.records.count())

		if n.parent != parent {
			t.Fatalf("invalid parent pointer for key=%v", n.key)
		}

		if lower != nil && compare(n.key, *lower) <= 0 {
			t.Fatalf("BST lower bound violated: %v", n.key)
		}

		if upper != nil && compare(n.key, *upper) >= 0 {
			t.Fatalf("BST upper bound violated: %v", n.key)
		}

		if n.color == Red {
			if n.left != nil && n.left.color == Red {
				t.Fatalf("red-red violation at key=%v", n.key)
			}
			if n.right != nil && n.right.color == Red {
				t.Fatalf("red-red violation at key=%v", n.key)
			}
		}

		leftHeight := walk(
			n.left, n, lower, &n.key,
		)
		rightHeight := walk(
			n.right, n, &n.key, upper,
		)

		if leftHeight != rightHeight {
			t.Fatalf(
				"black height mismatch at key=%v: left=%d right=%d",
				n.key, leftHeight, rightHeight,
			)
		}

		if n.color == Black {
			return leftHeight + 1
		}

		return leftHeight
	}

	walk(tree.root, nil, nil, nil)

	if nodes != uint64(tree.nodesCount) {
		t.Fatalf(
			"nodes count mismatch: actual=%d stored=%d",
			nodes, tree.nodesCount,
		)
	}

	if elements != tree.elemsCount {
		t.Fatalf(
			"elements count mismatch: actual=%d stored=%d",
			elements, tree.elemsCount,
		)
	}
}

func TestRangeIndexBinaryTreeInvariants(t *testing.T) {
	for _, size := range []int{0, 1, 2, 3, 10, 100, 1000} {
		t.Run(
			fmt.Sprintf("nodes_%d", size),
			func(t *testing.T) {
				idx := New(cmp.Compare[domain.IntValue])

				// Insert keys in descending order.
				// This forces many RBTree rotations.
				for j := size - 1; j >= 0; j-- {
					if err := idx.Add(
						domain.IntValue(j),
						domain.RecordID(j+1),
					); err != nil {
						t.Fatal(err)
					}
				}

				data, err := idx.MarshalBinary()
				if err != nil {
					t.Fatal(err)
				}

				restored := New(cmp.Compare[domain.IntValue])

				if err := restored.UnmarshalBinary(data); err != nil {
					t.Fatal(err)
				}

				assertRBTreeValid(
					t,
					restored.tree,
					cmp.Compare[domain.IntValue],
				)
			},
		)
	}
}

func TestRangeIndexBinaryDeterministic(t *testing.T) {
	first := New(cmp.Compare[domain.IntValue])
	second := New(cmp.Compare[domain.IntValue])

	keys := []domain.IntValue{50, 10, 90, 30, 70, 20, 80}

	for _, key := range keys {
		if err := first.Add(
			key,
			domain.RecordID(key),
		); err != nil {
			t.Fatal(err)
		}
	}

	for j := len(keys) - 1; j >= 0; j-- {
		key := keys[j]
		if err := second.Add(
			key,
			domain.RecordID(key),
		); err != nil {
			t.Fatal(err)
		}
	}

	a, err := first.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	b, err := second.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(a, b) {
		t.Fatal("equal indexes produced different snapshots")
	}
}

func TestRangeIndexBinarySortedTraversal(t *testing.T) {
	idx := New(cmp.Compare[domain.IntValue])

	for _, key := range []domain.IntValue{
		40, 10, 50, 20, 30,
	} {
		if err := idx.Add(
			key,
			domain.RecordID(key),
		); err != nil {
			t.Fatal(err)
		}
	}

	data, err := idx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	restored := New(cmp.Compare[domain.IntValue])
	if err := restored.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}

	var ascendingKeys []domain.IntValue
	for n := range restored.tree.traverse(ascending) {
		ascendingKeys = append(ascendingKeys, n.key)
	}

	var descendingKeys []domain.IntValue
	for n := range restored.tree.traverse(descending) {
		descendingKeys = append(descendingKeys, n.key)
	}

	if !slices.Equal(
		ascendingKeys,
		[]domain.IntValue{10, 20, 30, 40, 50},
	) {
		t.Fatalf("ascending traversal: %v", ascendingKeys)
	}

	if !slices.Equal(
		descendingKeys,
		[]domain.IntValue{50, 40, 30, 20, 10},
	) {
		t.Fatalf("descending traversal: %v", descendingKeys)
	}
}

func TestRangeIndexBinaryInsertAfterRestore(t *testing.T) {
	idx := New(cmp.Compare[domain.IntValue])

	for j := 0; j < 100; j++ {
		if err := idx.Add(
			domain.IntValue(j),
			domain.RecordID(j+1),
		); err != nil {
			t.Fatal(err)
		}
	}

	data, err := idx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	restored := New(cmp.Compare[domain.IntValue])
	if err := restored.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}

	// Insert new keys.
	for j := 100; j < 200; j++ {
		if err := restored.Add(
			domain.IntValue(j),
			domain.RecordID(j+1),
		); err != nil {
			t.Fatal(err)
		}
	}

	// Insert a new RecordID for an existing key.
	if err := restored.Add(
		domain.IntValue(50),
		domain.RecordID(1000),
	); err != nil {
		t.Fatal(err)
	}

	checkExact(
		t,
		restored,
		domain.IntValue(50),
		[]domain.RecordID{51, 1000},
	)

	assertRBTreeValid(
		t,
		restored.tree,
		cmp.Compare[domain.IntValue],
	)

	if restored.tree.nodesCount != 200 {
		t.Fatalf(
			"nodes: got %d, want 200",
			restored.tree.nodesCount,
		)
	}

	if restored.tree.elemsCount != 201 {
		t.Fatalf(
			"records: got %d, want 201",
			restored.tree.elemsCount,
		)
	}
}

func TestRangeIndexBinarySingleAndMany(t *testing.T) {
	idx := New(cmp.Compare[domain.IntValue])

	if err := idx.Add(domain.IntValue(10), 1); err != nil {
		t.Fatal(err)
	}

	for j := 1; j <= 1000; j++ {
		if err := idx.Add(
			domain.IntValue(20),
			domain.RecordID(j),
		); err != nil {
			t.Fatal(err)
		}
	}

	data, err := idx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	restored := New(cmp.Compare[domain.IntValue])
	if err := restored.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}

	var singleNode, manyNode *node[domain.IntValue]

	for n := range restored.tree.traverse(ascending) {
		switch n.key {
		case 10:
			singleNode = n
		case 20:
			manyNode = n
		}
	}

	if singleNode == nil || manyNode == nil {
		t.Fatal("missing restored nodes")
	}

	if singleNode.records.count() != 1 {
		t.Fatal("single node has invalid record count")
	}

	if manyNode.records.count() != 1000 {
		t.Fatal("many node has invalid record count")
	}

	var expected []domain.RecordID
	for j := 1; j <= 1000; j++ {
		expected = append(expected, domain.RecordID(j))
	}

	checkExact(t, restored, 10, []domain.RecordID{1})
	checkExact(t, restored, 20, expected)

	assertRBTreeValid(
		t,
		restored.tree,
		cmp.Compare[domain.IntValue],
	)
}
