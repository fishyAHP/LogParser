package set

type Set[K comparable] struct {
	set map[K]struct{}
}

func New[K comparable](length int) *Set[K] {
	return &Set[K]{
		make(map[K]struct{}, length),
	}
}

func (s *Set[K]) Add(record K) bool {
	if _, ok := s.set[record]; !ok {
		s.set[record] = struct{}{}
		return true
	}

	return false
}

func (s *Set[K]) AddMany(records ...K) {
	for _, record := range records {
		s.Add(record)
	}
}

// Remove delete record from Set. If it found and deleted
// return true, else false.
func (s *Set[K]) Remove(record K) bool {
	if _, ok := s.set[record]; ok {
		delete(s.set, record)
		return true
	}

	return false
}

func (s *Set[K]) Slice() []K {
	res := make([]K, 0, len(s.set))

	for k := range s.set {
		res = append(res, k)
	}

	return res
}

func (s *Set[K]) Len() int {
	return len(s.set)
}

func (s *Set[K]) Contains(value K) bool {
	if _, ok := s.set[value]; ok {
		return true
	}
	return false
}
