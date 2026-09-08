package component

import (
	"fishyAHP/LogParser.git/internal/core/domain"
)

type LogComponent = string

type Index struct {
	index map[LogComponent][]domain.RecordData
}

func New() *Index {
	return &Index{
		index: make(map[LogComponent][]domain.RecordData),
	}
}

func (i *Index) Len() int {
	return len(i.index)
}

func (i *Index) Add(comp LogComponent, data domain.RecordData) {
	i.index[comp] = append(i.index[comp], data)
}
