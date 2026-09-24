package query

import "fishyAHP/LogParser.git/internal/features/query/lexer"

type Querizer struct {
	lex lexer.Lexer
}

func (q *Querizer) Query(input string) {
	lexemes := q.lex.Parse(input)
	var b BinaryExpr

	for i := 0; i < len(lexemes); i++ {

	}
}
