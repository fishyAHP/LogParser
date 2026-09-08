package timestamp

type rbTree struct {
	head  *Node
	count int
}

func newRBTree() *rbTree {
	return &rbTree{}
}
