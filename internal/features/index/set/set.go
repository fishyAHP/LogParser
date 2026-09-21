package set

import "fishyAHP/LogParser.git/internal/core/domain"

type Set struct {
	set map[domain.RecordData]struct{}
}

func NewSet(length int) *Set {
	return &Set{
		make(map[domain.RecordData]struct{}, length),
	}
}

func (s *Set) Add(record domain.RecordData) {
	if _, ok := s.set[record]; !ok {
		s.set[record] = struct{}{}
	}
}

// Remove delete record from Set. If it found and deleted
// return true, else false.
func (s *Set) Remove(record domain.RecordData) bool {
	if _, ok := s.set[record]; ok {
		delete(s.set, record)
		return true
	}

	return false
}

func (s *Set) Slice() []domain.RecordData {
	res := make([]domain.RecordData, 0, len(s.set))

	for k := range s.set {
		res = append(res, k)
	}

	return res
}

func (s *Set) Len() int {
	return len(s.set)
}
