package set

func Intersection[K comparable](s1, s2 *Set[K]) *Set[K] {
	smaller := minSet[K](s1, s2)
	other := otherSet[K](smaller, s1, s2)
	res := New[K](smaller.Len())

	for k := range smaller.set {
		if other.Contains(k) {
			res.Add(k)
		}
	}

	return res
}

func Union[K comparable](s1, s2 *Set[K]) *Set[K] {
	s := New[K](s1.Len() + s2.Len())
	for k := range s1.set {
		s.Add(k)
	}

	for k := range s2.set {
		s.Add(k)
	}

	return s
}

func Difference[K comparable](s1, s2 *Set[K]) *Set[K] {
	s := New[K](s1.Len())

	for k := range s1.set {
		if !s2.Contains(k) {
			s.Add(k)
		}
	}

	return s
}

func minSet[K comparable](s1, s2 *Set[K]) *Set[K] {
	if s1.Len() <= s2.Len() {
		return s1
	}
	return s2
}

func otherSet[K comparable](cur, s1, s2 *Set[K]) *Set[K] {
	if cur == s1 {
		return s2
	}
	return s1
}
