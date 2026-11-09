package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

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
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf(
			"get user config directory: %w",
			err,
		)
	}

	dir := filepath.Join(
		configDir,
		"logparser",
	)
	if err = os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf(
			"create CLI config directory: %w",
			err,
		)
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:       "logparser> ",
		HistoryFile:  filepath.Join(dir, "history"),
		HistoryLimit: 500,
		Stdin:        in,
		Stdout:       out,
		Stderr:       errOut,

		AutoComplete: readline.NewPrefixCompleter(
			readline.PcItem("query"),
			readline.PcItem("ingest"),
			readline.PcItem("ingest-file"),
			readline.PcItem("ingest-tcp"),
			readline.PcItem("help"),
			readline.PcItem("exit"),
		),
	})
	if err != nil {
		return nil, fmt.Errorf(
			"initialize terminal: %w",
			err,
		)
	}

	return &TerminalInput{rl: rl}, nil
}

func (t *TerminalInput) ReadLine(
	prompt string,
) (string, error) {
	t.rl.SetPrompt(prompt)

	return t.rl.Readline()
}

func (t *TerminalInput) Close() error {
	return t.rl.Close()
}
