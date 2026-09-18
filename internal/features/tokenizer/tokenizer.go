package tokenizer

type Tokenizer struct {
	separator rune
}

func (t *Tokenizer) Tokenize(raw string) []Token {
	scanner := Scanner{input: []rune(raw)}
	result := make([]Token, 0, 6)

	for scanner.advance() == nil {
		if scanner.peek() == t.separator {
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
	}

	result = append(result, newToken(
		"\n",
		TokenEOF,
	))

	return result
}
