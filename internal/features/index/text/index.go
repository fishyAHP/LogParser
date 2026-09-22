package text

import (
	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/set"
	"fishyAHP/LogParser.git/internal/features/index/text/tokens"
)

type Index struct {
	invert    map[tokens.Token]*set.Set[domain.RecordData]
	tokenizer *tokens.Tokenizer
}

func (i Index) Add(k tokens.Token, data domain.RecordData) {
	
}

func (i Index) Get(k tokens.Token) (*set.Set[domain.RecordData], bool) {
	//TODO implement me
	panic("implement me")
}

func (i Index) Remove(k tokens.Token) bool {
	//TODO implement me
	panic("implement me")
}

func (i Index) Delete(k tokens.Token, data domain.RecordData) bool {
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
