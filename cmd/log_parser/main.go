package main

import (
	"fmt"
	"log"

	"fishyAHP/LogParser.git/internal/features/query/lexer"
)

func main() {
	var l lexer.Lexer

	lexemes, err := l.Parse(`time >= 12:15 and pid = 1234 or text='database timeout broke server'`)
	if err != nil {
		log.Fatal(err)
	}
	for _, lexeme := range lexemes {
		fmt.Printf("%d: %s\n", lexeme.Type, lexeme.Literal)
	}
}
