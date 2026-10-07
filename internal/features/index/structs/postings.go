package structs

import (
	"slices"

	"fishyAHP/LogParser.git/internal/core/domain"
)

type PostingList struct {
	postings []domain.RecordID
}

func NewPostingLists(capacity int) *PostingList {
	return &PostingList{
		postings: make([]domain.RecordID, 0, capacity),
	}
}

func (p *PostingList) Add(id domain.RecordID) bool {
	if len(p.postings) > 0 &&
		p.postings[len(p.postings)-1] >= id {
		return false
	}

	p.postings = append(p.postings, id)
	return true
}

func (p *PostingList) Contains(id domain.RecordID) bool {
	return p.Index(id) != -1
}

func (p *PostingList) Index(id domain.RecordID) int {
	left, right := 0, len(p.postings)-1

	for left <= right {
		mid := left + (right-left)/2

		if p.postings[mid] == id {
			return mid
		} else if p.postings[mid] > id {
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
		p.postings = slices.Delete(p.postings, idx, idx+1)
		return true
	}
	return false
}

func (p *PostingList) Clear() {
	p.postings = make([]domain.RecordID, 0)
}

func (p *PostingList) Len() int {
	return len(p.postings)
}

func (p *PostingList) Slice() []domain.RecordID {
	return slices.Clone(p.postings)
}
