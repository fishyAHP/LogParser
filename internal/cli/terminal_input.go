package cli

import (
	"errors"
	"io"

	"github.com/chzyer/readline"
)

type TerminalInput struct {
	rl *readline.Instance
}

func NewTerminalInput(
	in io.ReadCloser,
	out io.Writer,
	errOut io.Writer,
) (*TerminalInput, error) {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:       "logparser> ",
		HistoryLimit: 500,
		Stdin:        in,
		Stdout:       out,
		Stderr:       errOut,
	})
	if err != nil {
		return nil, err
	}

	return &TerminalInput{rl: rl}, nil
}

func (t *TerminalInput) ReadLine() (string, error) {
	line, err := t.rl.Readline()

	if errors.Is(err, readline.ErrInterrupt) {
		return "", io.EOF
	}

	return line, err
}

func (t *TerminalInput) Close() error {
	return t.rl.Close()
}
