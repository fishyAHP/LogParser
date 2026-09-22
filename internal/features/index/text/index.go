package text

import (
	"slices"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/set"
	"fishyAHP/LogParser.git/internal/features/index/text/words"
)

type Index struct {
	count     int
	invert    map[words.Token]*set.Set[domain.RecordData]
	tokenizer *words.Tokenizer
}

func (i Index) Add(s string, data domain.RecordData) {
	tokens := i.tokenizer.Tokenize(s)

	for _, token := range tokens {
		sett, ok := i.invert[token]
		if !ok {
			sett.Add(data)
			i.count++
		}
	}
}

func (i Index) Get(s string) (*set.Set[domain.RecordData], bool) {
	tokens := i.tokenizer.Tokenize(s)
	sets := make([]*set.Set[domain.RecordData], len(tokens))

	for _, token := range tokens {
		sets = append(sets, i.invert[token])
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
		return nil, false
	}
	start := 1
	res := set.New[domain.RecordData](sets[0].Len())
	res.AddMany(sets[0].Slice()...)

	for j := start; j < len(sets); j++ {
		res = set.Intersection(res, sets[j])
	}

	return res, true
}

func (i Index) Remove(k string) bool {
	//TODO implement me
	panic("implement me")
}

func (i Index) Delete(k string, data domain.RecordData) bool {
	//TODO implement me
	panic("implement me")
}

func (i Index) Len() int {
	//TODO implement me
	panic("implement me")
}

func (i Index) Clear() {
	//TODO implement me
	panic("implement me")
}
