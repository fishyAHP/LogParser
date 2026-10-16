package ranged

import (
	"testing"

	"fishyAHP/LogParser.git/internal/core/domain"
)

func TestMergeHeap(t *testing.T) {
	h := &mergeHeap{}

	values := []domain.RecordID{8, 3, 10, 1, 5, 2}

	for _, id := range values {
		h.push(heapItem{id: id})
	}

	expected := []domain.RecordID{1, 2, 3, 5, 8, 10}

	for _, want := range expected {
		got := h.pop()

		if got.id != want {
			t.Fatalf("got %d, want %d", got.id, want)
		}
	}

	if h.len() != 0 {
		t.Fatalf("heap should be empty")
	}
}
