package timestamp

type Index struct {
	tree *rbTree
}

func New() Index {
	return Index{}
}
