package words

import (
	"strings"
	"unicode"

	"fishyAHP/LogParser.git/internal/features/index/set"
)

type Tokenizer struct {
	minTokenLength uint8
	ignoringWords  *set.Set[Token]
}

type Token string

func NewDefault() *Tokenizer {
	ignore := []Token{"the", "a", "an", "of", "be", "is", "to", "are", "was", "were", "did"}
	s := set.New[Token](len(ignore))
	s.AddMany(ignore...)

	return &Tokenizer{
		minTokenLength: 1,
		ignoringWords:  s,
	}
}

func New(minToken uint8, ignoringWords ...Token) *Tokenizer {
	s := set.New[Token](len(ignoringWords))
	s.AddMany(ignoringWords...)

	return &Tokenizer{
		minTokenLength: minToken,
		ignoringWords:  s,
	}
}

func (t *Tokenizer) Tokenize(input string) []Token {
	if strings.TrimSpace(input) == "" {
		return nil
	}

	return t.splitString(strings.ToLower(input))
}

func (t *Tokenizer) splitString(s string) []Token {
	var start int

	res := make([]Token, 0, len(s)/6)
	runes := []rune(s)

	for i, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			if start < i {
				token := Token(runes[start:i])

				if t.isValid(token) {
					res = append(res, token)
				}
			}

			start = i + 1
		}
	}

	if start < len(runes) {
		token := Token(runes[start:])
		if t.isValid(token) {
			res = append(res, token)
		}
	}

	return res
}

func (t *Tokenizer) isValid(token Token) bool {
	return len([]rune(token)) >= int(t.minTokenLength) &&
		!t.ignoringWords.Contains(token)
}
