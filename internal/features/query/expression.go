package query

type Expr interface {
	IsExpr()
}

type BinaryExpr struct {
	Left     Expr
	Operator LogicalOperator
	Right    Expr
}

type Condition struct {
	Field    string
	Operator CompareOperator
	Value    string
}

type LogicalOperator uint8

const (
	And LogicalOperator = iota
	Or
)

type CompareOperator uint8

const (
	Equal CompareOperator = iota
	Bigger
	BiggerOrEqual
	Less
	LessOrEqual
)

func (c CompareOperator) String() string {
	switch c {
	case Equal:
		return "="
	case Bigger:
		return ">"
	case BiggerOrEqual:
		return ">="
	case Less:
		return "<"
	case LessOrEqual:
		return "<="
	default:
		return "unknown"
	}
}

func (lo LogicalOperator) String() string {
	switch lo {
	case And:
		return "AND"
	case Or:
		return "OR"
	default:
		return "unknown"
	}
}

func (c *Condition) IsExpr() {

}

func (b *BinaryExpr) IsExpr() {

}
