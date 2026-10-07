package structs

import "fishyAHP/LogParser.git/internal/core/domain"

type PostingLists struct {
	postings []domain.RecordID
}

func NewPostingLists(capacity int) *PostingLists {
	return &PostingLists{
		postings: make([]domain.RecordID, 0, capacity),
	}
}

func (p *PostingLists) Add(id domain.RecordID) {

}
