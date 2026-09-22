package query

import (
	"errors"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index"
	"fishyAHP/LogParser.git/internal/features/index/set"
)

type expr interface {
	IsExpr()
}

type BinaryExpr struct {
	Left     expr
	Operator LogicalOperator
	Right    expr
}

type Condition struct {
	Field    Field
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

type Field string

const (
	Level     Field = "level"
	PID       Field = "pid"
	Component Field = "component"
	IP        Field = "ip"
	Timestamp Field = "timestamp"
	Message   Field = "message"
)

func (c *Condition) IsExpr() {

}

func (b *BinaryExpr) IsExpr() {

}

func (c *Condition) Execute(i *index.IndexService) (*set.Set[domain.RecordData], error) {
	switch c.Field {
	case Level:
		if c.Operator != Equal {
			return nil, errors.New("unknown operation for level index")
		}

		return
	}
}
