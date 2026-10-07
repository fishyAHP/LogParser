package structs

import (
	"slices"

	"fishyAHP/LogParser.git/internal/core/domain"
)

type PostingList struct {
	posting []domain.RecordID
}

func NewPostingLists(capacity int) *PostingList {
	return &PostingList{
		posting: make([]domain.RecordID, 0, capacity),
	}
}

func NewPostingFromSorted(sl []domain.RecordID) *PostingList {
	return &PostingList{
		posting: sl,
	}
}

func (p *PostingList) Add(id domain.RecordID) bool {
	if len(p.posting) > 0 &&
		p.posting[len(p.posting)-1] >= id {
		return false
	}

	p.posting = append(p.posting, id)
	return true
}

func (p *PostingList) Contains(id domain.RecordID) bool {
	return p.Index(id) != -1
}

func (p *PostingList) Index(id domain.RecordID) int {
	left, right := 0, len(p.posting)-1

	for left <= right {
		mid := left + (right-left)/2

		if p.posting[mid] == id {
			return mid
		} else if p.posting[mid] > id {
			right = mid - 1
		} else {
			left = mid + 1
		}
	}

	return -1
}

func (p *PostingList) Remove(id domain.RecordID) bool {
	idx := p.Index(id)
	if idx != -1 {
		p.posting = slices.Delete(p.posting, idx, idx+1)
		return true
	}
	return false
}

func (p *PostingList) Clear() {
	p.posting = make([]domain.RecordID, 0)
}

func (p *PostingList) Len() int {
	return len(p.posting)
}

func (p *PostingList) Slice() []domain.RecordID {
	return slices.Clone(p.posting)
}
