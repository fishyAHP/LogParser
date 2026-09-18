package tokenizer

import "errors"

type Scanner struct {
	input        []rune
	lastPosition int
	curPosition  int
}

func NewScanner(input string) Scanner {
	return Scanner{
		input:        []rune(input),
		lastPosition: -1,
	}
}

func (s *Scanner) peek() (rune, bool) {
	if s.curPosition >= 0 &&
		s.curPosition < len(s.input) {
		return s.input[s.curPosition], true

	}

	return 0, false
}

var EOF = errors.New("end of file")

func (s *Scanner) advance() error {
	if s.curPosition >= len(s.input) {
		return EOF
	}

	s.curPosition++
	return nil
}

func (s *Scanner) getValue() string {
	value := string(s.input[s.lastPosition+1 : s.curPosition])
	s.lastPosition = s.curPosition

	return value
}
