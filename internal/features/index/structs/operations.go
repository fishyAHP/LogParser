package structs

import (
	"fishyAHP/LogParser.git/internal/core/domain"
)

func forEachIntersectLists(
	s1, s2 *PostingList,
	visit func(
		id domain.RecordID,
	) bool) {
	i, j := 0, 0

	for i < s1.Len() && j < s2.Len() {
		switch {
		case (*s1)[i] < (*s2)[j]:
			i++
		case (*s1)[i] > (*s2)[j]:
			j++
		default:
			if !visit((*s1)[i]) {
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
		case (*s1)[i] < (*s2)[j]:
			res.Add((*s1)[i])
			i++
		case (*s1)[i] > (*s2)[j]:
			res.Add((*s2)[j])
			j++
		default:
			res.Add((*s1)[i])
			i++
			j++
		}
	}

	for ; i < s1.Len(); i++ {
		res.Add((*s1)[i])
	}

	for ; j < s2.Len(); j++ {
		res.Add((*s2)[j])
	}

	return res
}

func DifferenceLists(s1, s2 *PostingList) *PostingList {
	res := NewPostingLists(s1.Len())

	i, j := 0, 0

	for i < s1.Len() && j < s2.Len() {
		switch {
		case (*s1)[i] < (*s2)[j]:
			res.Add((*s1)[i])
			i++
		case (*s1)[i] > (*s2)[j]:
			j++
		default:
			i++
			j++

		}
	}

	for ; i < s1.Len(); i++ {
		res.Add((*s1)[i])
	}

	return res
}
