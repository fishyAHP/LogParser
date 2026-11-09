package cli

import (
	"bufio"
	"errors"
	"io"
	"strings"
)

type BasicInput struct {
	reader *bufio.Reader
}

func NewBasicInput(r io.Reader) *BasicInput {
	return &BasicInput{
		reader: bufio.NewReader(r),
	}
}

func (b *BasicInput) ReadLine() (string, error) {
	line, err := b.reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	if errors.Is(err, io.EOF) && len(line) == 0 {
		return "", io.EOF
	}

	return strings.TrimRight(line, "\r\n"), nil
}

func (b *BasicInput) Close() error {
	return nil
}
