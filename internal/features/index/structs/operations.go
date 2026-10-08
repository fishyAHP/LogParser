package structs

import (
	"slices"

	"fishyAHP/LogParser.git/internal/core/domain"
)

func IntersectionSets[K comparable](
	s1, s2 *Set[K],
) *Set[K] {
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

func UnionSets[K comparable](
	s1, s2 *Set[K],
) *Set[K] {
	s := NewSet[K](s1.Len() + s2.Len())
	for k := range s1.set {
		s.Add(k)
	}

	for k := range s2.set {
		s.Add(k)
	}

	return s
}

func DifferenceSets[K comparable](
	s1, s2 *Set[K],
) *Set[K] {
	s := NewSet[K](s1.Len())

	for k := range s1.set {
		if !s2.Contains(k) {
			s.Add(k)
		}
	}

	return s
}

func minSet[K comparable](
	s1, s2 *Set[K],
) *Set[K] {
	if s1.Len() <= s2.Len() {
		return s1
	}
	return s2
}

func otherSet[K comparable](
	cur, s1, s2 *Set[K],
) *Set[K] {
	if cur == s1 {
		return s2
	}
	return s1
}

func forEachIntersectLists(
	s1, s2 *PostingList,
	visit func(
	id domain.RecordID,
) bool) {
	i, j := 0, 0

	for i < s1.Len() && j < s2.Len() {
		switch {
		case s1.posting[i] < s2.posting[j]:
			i++
		case s1.posting[i] > s2.posting[j]:
			j++
		default:
			if !visit(s1.posting[i]) {
				return
			}
			i++
			j++
		}
	}
}

func IntersectionLists(s1, s2 *PostingList) *PostingList {
	capacity := min(s1.Len(), s2.Len())
	res := NewPostingLists(capacity)

	forEachIntersectLists(
		s1, s2,
		func(id domain.RecordID) bool {
			res.Add(id)
			return true
		})

	return res
}

func IsListIntersects(s1, s2 *PostingList) bool {
	var found bool

	forEachIntersectLists(
		s1, s2,
		func(id domain.RecordID) bool {
			found = true
			return false
		})

	return found
}

func CountListIntersections(s1, s2 *PostingList) int {
	var count int

	forEachIntersectLists(
		s1, s2,
		func(id domain.RecordID) bool {
			count++
			return true
		})

	return count
}

func UnionLists(s1, s2 *PostingList) *PostingList {
	res := NewPostingLists(s1.Len() + s2.Len())
	i, j := 0, 0

	for i < s1.Len() && j < s2.Len() {
		switch {
		case s1.posting[i] < s2.posting[j]:
			res.Add(s1.posting[i])
			i++
		case s1.posting[i] > s2.posting[j]:
			res.Add(s2.posting[j])
			j++
		default:
			res.Add(s1.posting[i])
			i++
			j++
		}
	}

	for ; i < s1.Len(); i++ {
		res.Add(s1.posting[i])
	}

	for ; j < s2.Len(); j++ {
		res.Add(s2.posting[j])
	}

	return res
}

func DifferenceLists(s1, s2 *PostingList) *PostingList {
	res := NewPostingLists(s1.Len())

	i, j := 0, 0

	for i < s1.Len() && j < s2.Len() {
		switch {
		case s1.posting[i] < s2.posting[j]:
			res.Add(s1.posting[i])
			i++
		case s1.posting[i] > s2.posting[j]:
			j++
		default:
			i++
			j++

		}
	}

	for ; i < s1.Len(); i++ {
		res.Add(s1.posting[i])
	}

	return res
}

func SetToPosting(set *Set[domain.RecordID]) *PostingList {
	arr := make([]domain.RecordID, 0, set.Len())
	for k := range set.set {
		arr = append(arr, k)
	}
	slices.Sort(arr)

	posting := NewPostingFromSorted(arr)

	return posting
}
