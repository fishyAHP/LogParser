package tokenizer

type Token struct {
	Value string
	Type  TokenType
}

type TokenType uint8

const (
	TokenString TokenType = iota
	TokenSeparator
	TokenEOF
)

func newToken(s string, ttype TokenType) Token {
	return Token{
		Value: s,
		Type:  ttype,
	}
}
