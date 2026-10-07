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
type invertIndex = map[words.Token]*structs.Set[domain.RecordID]

func New() *Index {
	return &Index{
		invert:    make(invertIndex),
		Tokenizer: words.NewDefault(),
	}
}

func (i *Index) Search(value domain.Value) (*structs.Set[domain.RecordID], error) {
	strVal, ok := value.(domain.StringValue)
	if !ok {
		return nil, common.ErrInvalidType
	}

	res := i.get(string(strVal))
	if res == nil {
		return nil, common.ErrNotFoundRecord
	}
	return res.Clone(), nil
}

func (i *Index) Add(value domain.Value, record domain.RecordID) error {
	strVal, ok := value.(domain.StringValue)
	if !ok {
		return common.ErrInvalidType
	}

	tokens := i.Tokenizer.Tokenize(string(strVal))
	sett := structs.New[words.Token](len(tokens))
	sett.AddMany(tokens...)
	tokens = sett.Slice()

	for _, token := range tokens {
		if _, ok := i.invert[token]; !ok {
			i.invert[token] = structs.New[domain.RecordID](1)
		}

		if i.invert[token].Add(record) {
			i.count++
		}
	}

	return nil
}

func (i *Index) get(s string) *structs.Set[domain.RecordID] {
	tokens := i.Tokenizer.Tokenize(s)
	sets := make([]*structs.Set[domain.RecordID], 0, len(tokens))

	for _, token := range tokens {
		if sett, ok := i.invert[token]; ok {
			sets = append(sets, sett)
		} else {
			return nil
		}
	}

	slices.SortFunc(sets, func(a, b *structs.Set[domain.RecordID]) int {
		if a.Len() > b.Len() {
			return 1
		}
		if a.Len() < b.Len() {
			return -1
		}
		return 0
	})

	if len(sets) < 1 {
		return nil
	}
	start := 1
	res := structs.New[domain.RecordID](sets[0].Len())
	res.AddMany(sets[0].Slice()...)

	for j := start; j < len(sets); j++ {
		res = structs.Intersection(res, sets[j])

		if res.Len() == 0 {
			return nil
		}
	}

	return res
}

func (i *Index) Remove(
	s domain.Value,
	data domain.RecordID) error {
	keys, ok := s.(domain.StringValue)
	if !ok {
		return common.ErrInvalidType
	}
	tokens := i.Tokenizer.Tokenize(string(keys))

	var isChanged bool
	for _, token := range tokens {
		if posting, ok := i.invert[token]; ok {
			if !isChanged {
				isChanged = true
			}

			if posting.Remove(data) {
				i.count--
			}
			if posting.Len() == 0 {
				delete(i.invert, token)
			}
		}
	}

	if !isChanged {
		return common.ErrNotFoundRecord
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
