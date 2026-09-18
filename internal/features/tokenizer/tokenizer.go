package tokenizer

import "strings"

type Tokenizer struct {
	separator rune
}

func NewTokenizer(sep rune) Tokenizer {
	return Tokenizer{
		sep,
	}
}

func (t *Tokenizer) Tokenize(raw string) []Token {
	if strings.TrimSpace(raw) == "" {
		return []Token{newToken("", TokenEOF)}
	}

	scanner := NewScanner(raw)
	result := make([]Token, 0, 6)

	for {
		if r, ok := scanner.peek(); ok && r == t.separator {
			s := scanner.getValue()
			result = append(result, newToken(
				s,
				TokenString,
			))
			result = append(result, newToken(
				string(t.separator),
				TokenSeparator,
			))
		}

		if scanner.advance() != nil {
			s := scanner.getValue()
			result = append(result, newToken(
				s,
				TokenString,
			))

			break
		}
	}

	result = append(result, newToken(
		"",
		TokenEOF,
	))
	return result
}
