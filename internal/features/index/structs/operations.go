package structs

func IntersectionSets[K comparable](s1, s2 *Set[K]) *Set[K] {
	smaller := minSet[K](s1, s2)
	other := otherSet[K](smaller, s1, s2)
	res := NewSet[K](smaller.Len())

	for k := range smaller.set {
		if other.Contains(k) {
			res.Add(k)
		}
	}

	return res
}

func UnionSets[K comparable](s1, s2 *Set[K]) *Set[K] {
	s := NewSet[K](s1.Len() + s2.Len())
	for k := range s1.set {
		s.Add(k)
	}

	for k := range s2.set {
		s.Add(k)
	}

	return s
}

func DifferenceSets[K comparable](s1, s2 *Set[K]) *Set[K] {
	s := NewSet[K](s1.Len())

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

func IntersectionLists(s1, s2 *PostingList) *PostingList {
	capacity := min(s1.Len(), s2.Len())
	res := NewPostingLists(capacity)

	i, j := 0, 0

	for i < s1.Len() && j < s2.Len() {
		switch {
		case s1.postings[i] < s2.postings[j]:
			i++
		case s1.postings[i] > s2.postings[j]:
			j++
		default:
			res.Add(s1.postings[i])
			i++
			j++
		}
	}

	return res
}

func UnionLists(s1, s2 *PostingList) *PostingList {
	res := NewPostingLists(s1.Len() + s2.Len())
	i, j := 0, 0

	for i < s1.Len() && j < s2.Len() {
		switch {
		case s1.postings[i] < s2.postings[j]:
			res.Add(s1.postings[i])
			i++
		case s1.postings[i] > s2.postings[j]:
			res.Add(s2.postings[j])
			j++
		default:
			res.Add(s1.postings[i])
			i++
			j++
		}
	}

	for ; i < s1.Len(); i++ {
		res.Add(s1.postings[i])
	}

	for ; j < s2.Len(); j++ {
		res.Add(s2.postings[j])
	}

	return res
}

func DifferenceLists(s1, s2 *PostingList) *PostingList {
	res := NewPostingLists(s1.Len())

	i, j := 0, 0

	for i < s1.Len() && j < s2.Len() {
		switch {
		case s1.postings[i] < s2.postings[j]:
			res.Add(s1.postings[i])
			i++
		case s1.postings[i] > s2.postings[j]:
			j++
		default:
			i++
			j++

		}
	}

	for ; i < s1.Len(); i++ {
		res.Add(s1.postings[i])
	}

	return res
}
