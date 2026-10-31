package structs

import (
	"encoding/binary"
	"errors"
	"fmt"
	"iter"
	"slices"

	"fishyAHP/LogParser.git/internal/core/binaryio"
	"fishyAHP/LogParser.git/internal/core/domain"
)

type PostingList []domain.RecordID

func NewPostingLists(capacity int) *PostingList {
	list := make(PostingList, 0, capacity)
	return &list
}

func NewPostingFromSorted(sl []domain.RecordID) *PostingList {
	list := PostingList(sl)
	return &list
}

func (p *PostingList) Add(id domain.RecordID) bool {
	if id == 0 {
		return false
	}
	if len(*p) > 0 &&
		(*p)[len(*p)-1] >= id {
		return false
	}

	*p = append(*p, id)
	return true
}

func (p *PostingList) Contains(id domain.RecordID) bool {
	return p.Index(id) != -1
}

func (p *PostingList) Index(id domain.RecordID) int {
	left, right := 0, len(*p)-1

	for left <= right {
		mid := left + (right-left)/2

		if (*p)[mid] == id {
			return mid
		} else if (*p)[mid] > id {
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
		*p = slices.Delete(*p, idx, idx+1)
		return true
	}
	return false
}

func (p *PostingList) Clear() {
	*p = make([]domain.RecordID, 0)
}

func (p *PostingList) Len() int {
	return len(*p)
}

func (p *PostingList) Slice() []domain.RecordID {
	return slices.Clone(*p)
}

func (p *PostingList) Iterator() iter.Seq[domain.RecordID] {
	return func(yield func(domain.RecordID) bool) {
		for _, id := range *p {
			if !yield(id) {
				return
			}
		}
	}
}

func (p *PostingList) At(idx int) domain.RecordID {
	if idx > len(*p) ||
		idx < 0 {
		return 0
	}
	return (*p)[idx]
}

func (p *PostingList) MarshalBinary() ([]byte, error) {
	encoder, err := binaryio.NewEncoder(p.Len() * 8)
	if err != nil {
		return nil, err
	}

	if err = p.EncodeTo(encoder); err != nil {
		return nil, err
	}

	return encoder.Result(), nil
}

func (p *PostingList) UnmarshalBinary(data []byte) error {
	if len(data)%8 != 0 {
		return errors.New(
			"size of elements must divide by 8",
		)
	}

	posts := make(PostingList, len(data)/8)
	for i := range posts {
		id := domain.RecordID(
			binary.BigEndian.Uint64(
				data[i*8 : (i+1)*8],
			),
		)

		if id <= 0 {
			return fmt.Errorf(
				"invalid record id: %d",
				id,
			)
		}

		if i > 0 && id <= posts[i-1] {
			return fmt.Errorf(
				"posting list must be sorted: %d",
				id,
			)
		}

		posts[i] = id
	}

	*p = posts
	return nil
}

func (p *PostingList) iterator() iter.Seq[domain.RecordID] {
	return func(yield func(domain.RecordID) bool) {
		for _, id := range *p {
			if !yield(id) {
				return
			}
		}
	}
}

func (p *PostingList) EncodeTo(
	encoder *binaryio.Encoder,
) error {
	for id := range p.iterator() {
		encoder.Uint64(uint64(id))
	}

	return nil
}
