package text

import (
	"slices"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/common"
	"fishyAHP/LogParser.git/internal/features/index/structs"
	"fishyAHP/LogParser.git/internal/features/index/text/words"
)

type Index struct {
	count     int
	invert    invertIndex
	Tokenizer *words.Tokenizer
}
type invertIndex = map[words.Token]*structs.PostingList

func New() *Index {
	return &Index{
		invert:    make(invertIndex),
		Tokenizer: words.NewDefault(),
	}
}

func (i *Index) Search(
	value domain.Value,
) (*structs.PostingList, error) {
	strVal, ok := value.(domain.StringValue)
	if !ok {
		return nil, common.ErrInvalidType
	}

	res := i.get(string(strVal))
	if res == nil {
		return nil, common.ErrRecordNotFound
	}
	return res, nil
}

func (i *Index) Add(
	value domain.Value,
	record domain.RecordID,
) error {
	strVal, ok := value.(domain.StringValue)
	if !ok {
		return common.ErrInvalidType
	}

	tokens := i.Tokenizer.Tokenize(string(strVal))
	set := structs.NewSet[words.Token](len(tokens))
	set.AddMany(tokens...)

	set.ForEach(func(token words.Token) bool {
		if _, ok := i.invert[token]; !ok {
			i.invert[token] = structs.NewPostingLists(1)
		}

		if i.invert[token].Add(record) {
			i.count++
		}

		return false
	})

	return nil
}

func (i *Index) get(
	s string,
) *structs.PostingList {
	tokens := i.Tokenizer.Tokenize(s)
	lists := make([]*structs.PostingList, 0, len(tokens))

	for _, token := range tokens {
		if list, ok := i.invert[token]; ok {
			lists = append(lists, list)
		} else {
			return nil
		}
	}

	slices.SortFunc(lists, func(a, b *structs.PostingList) int {
		if a.Len() > b.Len() {
			return 1
		}
		if a.Len() < b.Len() {
			return -1
		}
		return 0
	})

	if len(lists) < 1 {
		return nil
	}
	start := 1
	res := lists[0]

	for j := start; j < len(lists); j++ {
		res = structs.IntersectionLists(res, lists[j])

		if res.Len() == 0 {
			return nil
		}
	}

	return res
}

func (i *Index) Remove(
	s domain.Value,
	data domain.RecordID,
) error {
	keys, ok := s.(domain.StringValue)
	if !ok {
		return common.ErrInvalidType
	}
	tokens := i.Tokenizer.Tokenize(string(keys))

	var isChanged bool
	for _, token := range tokens {
		if posting, ok := i.invert[token]; ok {
			if posting.Remove(data) {
				isChanged = true
				i.count--
			}
			if posting.Len() == 0 {
				delete(i.invert, token)
			}
		}
	}

	if !isChanged {
		return common.ErrRecordNotFound
	}
	return nil
}

func (i *Index) Len() int {
	return i.count
}

func (i *Index) Clear() {
	i.invert = make(invertIndex)
	i.count = 0
}
