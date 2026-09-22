package set

func Intersection(s1, s2 *Set) *Set {
	small := minSet(s1, s2)
	other := otherSet(small, s1, s2)
	res := NewSet(small.Len())

	for k := range small.set {
		if other.Contains(k) {
			res.Add(k)
		}
	}

	return res
}

func Union(s1, s2 *Set) *Set {
	s := NewSet(s1.Len() + s2.Len())
	for k := range s1.set {
		s.Add(k)
	}

	for k := range s2.set {
		s.Add(k)
	}

	return s
}

func minSet(s1, s2 *Set) *Set {
	if s1.Len() < s2.Len() {
		return s1
	}
	return s2
}

func otherSet(cur, s1, s2 *Set) *Set {
	if cur == s1 {
		return s2
	}
	return s1
}
