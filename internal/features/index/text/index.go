package text

import (
	"slices"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/set"
	"fishyAHP/LogParser.git/internal/features/index/text/words"
)

type Index struct {
	count     int
	invert    invertIndex
	Tokenizer *words.Tokenizer
}

type invertIndex = map[words.Token]*set.Set[domain.RecordData]

func New() *Index {
	return &Index{
		invert:    make(invertIndex),
		Tokenizer: words.NewDefault(),
	}
}

func (i *Index) Add(s string, data domain.RecordData) {
	tokens := i.Tokenizer.Tokenize(s)

	for _, token := range tokens {
		if _, ok := i.invert[token]; !ok {
			i.invert[token] = set.New[domain.RecordData](1)
		}

		if i.invert[token].Add(data) {
			i.count++
		}
	}
}

func (i *Index) Get(s string) *set.Set[domain.RecordData] {
	tokens := i.Tokenizer.Tokenize(s)
	sets := make([]*set.Set[domain.RecordData], 0, len(tokens))

	for _, token := range tokens {
		if sett, ok := i.invert[token]; ok {
			sets = append(sets, sett)
		} else {
			return nil
		}
	}

	slices.SortFunc(sets, func(a, b *set.Set[domain.RecordData]) int {
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
	res := set.New[domain.RecordData](sets[0].Len())
	res.AddMany(sets[0].Slice()...)

	for j := start; j < len(sets); j++ {
		res = set.Intersection(res, sets[j])

		if res.Len() == 0 {
			return nil
		}
	}

	return res
}

func (i *Index) RemoveTokens(s string) bool {
	tokens := i.Tokenizer.Tokenize(s)

	var isChanged bool
	for _, token := range tokens {
		if _, ok := i.invert[token]; ok {
			if !isChanged {
				isChanged = true
			}
			i.count -= i.invert[token].Len()
			delete(i.invert, token)
		}
	}

	return isChanged
}

func (i *Index) RemoveRecord(s string, data domain.RecordData) bool {
	tokens := i.Tokenizer.Tokenize(s)

	var isDeleted bool
	for _, token := range tokens {
		if sett, ok := i.invert[token]; ok {
			if sett.Remove(data) {
				i.count--

				if !isDeleted {
					isDeleted = true
				}
			}

			if sett.Len() == 0 {
				delete(i.invert, token)
			}
		}
	}

	return isDeleted
}

func (i *Index) Len() int {
	return i.count
}

func (i *Index) Clear() {
	i.invert = make(map[words.Token]*set.Set[domain.RecordData])
	i.count = 0
}
