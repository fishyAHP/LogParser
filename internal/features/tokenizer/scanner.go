package tokenizer

import "errors"

type Scanner struct {
	input        []rune
	lastPosition int
	curPosition  int
}

func (s *Scanner) peek() rune {
	return s.input[s.curPosition]
}

var EOF = errors.New("end of file")

func (s *Scanner) advance() error {
	if s.curPosition+1 >= len(s.input) {
		return EOF
	}

	s.curPosition++
	return nil
}

func (s *Scanner) getValue() string {
	return string(s.input[s.lastPosition:s.curPosition])
}

// msgtime
