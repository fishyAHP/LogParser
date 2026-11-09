package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"unicode"

	"fishyAHP/LogParser.git/internal/app"
	"fishyAHP/LogParser.git/internal/core/domain"
	"github.com/chzyer/readline"
)

var ErrUnknownCommand = errors.New("unknown command")

type Input interface {
	ReadLine() (string, error)
	io.Closer
}

type CLI struct {
	service *app.Service

	input Input

	out    io.Writer
	errOut io.Writer

	logs <-chan string
}

func New(
	in io.ReadCloser,
	out io.Writer,
	errOut io.Writer,
) (*CLI, error) {
	service, err := app.NewService(&domain.Scheme{
		Separator: ',',
		Parameters: []domain.Field{
			{
				Name:      "level",
				FieldType: domain.StringType,
				IndexType: domain.HashIndex,
			},
			{
				Name:      "pid",
				FieldType: domain.IntType,
				IndexType: domain.RangeIndex,
			},
			{
				Name:      "message",
				FieldType: domain.StringType,
				IndexType: domain.TextIndex,
			},
			{
				Name:      "timestamp",
				FieldType: domain.TimeType,
				IndexType: domain.RangeIndex,
			},
		},
	},
		domain.JSON,
		".storage",
	)

	if err != nil {
		return nil, err
	}

	var input Input = NewBasicInput(in)

	if file, ok := in.(*os.File); ok &&
		readline.IsTerminal(int(file.Fd())) {

		terminal, err := NewTerminalInput(in, out, errOut)
		if err != nil {
			return nil, errors.Join(
				err,
				service.Close(),
			)
		}

		input = terminal
	}

	return &CLI{
		service: service,
		input:   input,
		out:     out,
		errOut:  errOut,
	}, nil
}

type InputResult struct {
	line string
	err  error
}

func (c *CLI) ReadInput(
	ctx context.Context,
	ch chan<- InputResult,
) {
	defer close(ch)

	for {
		line, err := c.input.ReadLine()

		if errors.Is(err, io.EOF) {
			return
		}

		result := InputResult{
			line: line,
			err:  err,
		}

		select {
		case <-ctx.Done():
			return

		case ch <- result:
		}

		if err != nil {
			return
		}
	}
}

func (c *CLI) Run(
	ctx context.Context,
) error {
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}

		line, err := c.input.ReadLine()

		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		exit, err := c.execute(line)
		if err != nil {
			fmt.Fprintln(c.errOut, "Error: ", err)
			continue
		}

		if exit {
			return nil
		}
	}
}

func (c *CLI) execute(line string) (bool, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return false, nil
	}

	idx := strings.IndexFunc(line, unicode.IsSpace)

	command := line
	argument := ""

	if idx >= 0 {
		command = line[:idx]
		argument = strings.TrimSpace(line[idx:])
	}

	switch strings.ToLower(command) {
	case "ingest":
		return false, c.ingest(argument)

	case "ingest-file":
		return false, c.ingestFile(argument)

	case "ingest-tcp":
		return false, c.ingestTCP(argument)

	case "query":
		return false, c.query(argument)

	case "exit":
		return true, nil

	default:
		return false, fmt.Errorf(
			"%w: %q",
			ErrUnknownCommand,
			command,
		)
	}
}

func (c *CLI) ingest(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return errors.New(
			"expect not empty argument",
		)
	}

	bytes := []byte(raw)
	err := c.service.Ingest(bytes)
	if err != nil {
		return fmt.Errorf(
			"ingest: %w",
			err,
		)
	}
	return nil
}

func (c *CLI) ingestFile(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New(
			"expected path file",
		)
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf(
			"open file %s: %w",
			path,
			err,
		)
	}
	defer func() {
		if err = file.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	if err = c.service.IngestReader(file); err != nil {
		return fmt.Errorf(
			"ingest file %s: %w",
			path,
			err,
		)
	}
	return nil
}

func (c *CLI) ingestTCP(address string) error {
	if strings.TrimSpace(address) == "" {
		return errors.New(
			"expected tcp address",
		)
	}

	conn, err := net.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf(
			"connect to %s: %w",
			address,
			err,
		)
	}

	defer func() {
		if err = conn.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	if err = c.service.IngestReader(conn); err != nil {
		return fmt.Errorf(
			"ingest TCP %q: %w",
			address,
			err,
		)
	}
	return nil
}

func (c *CLI) query(query string) error {
	if strings.TrimSpace(query) == "" {
		return errors.New(
			"expect not empty argument",
		)
	}

	rows, err := c.service.Query(query)
	if err != nil {
		return fmt.Errorf(
			"query %q: %w",
			query,
			err,
		)
	}

	for data, err := range rows {
		if err != nil {
			return err
		}
		fmt.Fprintln(c.out, string(data))
	}
	return nil
}

func (c *CLI) Close() error {
	return errors.Join(
		c.input.Close(),
		c.service.Close(),
	)
}
