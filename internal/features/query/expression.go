package query

import (
	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/set"
)

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

func (c *Condition) IsExpr() {

}

func (b *BinaryExpr) IsExpr() {

}

func (c *Condition) Execute() (*set.Set[domain.RecordData], error) {
	return nil, nil
}

func (b *BinaryExpr) Execute() {

}
