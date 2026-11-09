package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

type BasicInput struct {
	reader *bufio.Reader
	out    io.Writer
}

func NewBasicInput(
	r io.Reader,
	out io.Writer,
) *BasicInput {
	return &BasicInput{
		reader: bufio.NewReader(r),
		out:    out,
	}
}

func (b *BasicInput) ReadLine(
	prompt string,
) (string, error) {
	if prompt != "" {
		if _, err := fmt.Fprint(
			b.out,
			prompt,
		); err != nil {
			return "", err
		}
	}

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
