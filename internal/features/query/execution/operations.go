package execution

import (
	"iter"

	"fishyAHP/LogParser.git/internal/core/domain"
)

func intersect(
	i1, i2 iter.Seq[domain.RecordID],
) iter.Seq[domain.RecordID] {
	return func(yield func(domain.RecordID) bool) {
		next1, stop1 := iter.Pull(i1)
		defer stop1()

		next2, stop2 := iter.Pull(i2)
		defer stop2()

		a, okA := next1()
		b, okB := next2()

		for okA && okB {
			switch {
			case a < b:
				a, okA = next1()
			case a > b:
				b, okB = next2()
			default:
				if !yield(a) {
					return
				}

				a, okA = next1()
				b, okB = next2()
			}
		}
	}
}

func union(
	i1, i2 iter.Seq[domain.RecordID],
) iter.Seq[domain.RecordID] {
	return func(yield func(domain.RecordID) bool) {
		next1, stop1 := iter.Pull(i1)
		defer stop1()

		next2, stop2 := iter.Pull(i2)
		defer stop2()

		a, okA := next1()
		b, okB := next2()

		for okA && okB {
			switch {
			case a < b:
				if !yield(a) {
					return
				}

				a, okA = next1()
			case a > b:
				if !yield(b) {
					return
				}

				b, okB = next2()
			default:
				if !yield(a) {
					return
				}

				a, okA = next1()
				b, okB = next2()
			}
		}

		for okA {
			if !yield(a) {
				return
			}

			a, okA = next1()
		}

		for okB {
			if !yield(b) {
				return
			}

			b, okB = next2()
		}
	}
}
