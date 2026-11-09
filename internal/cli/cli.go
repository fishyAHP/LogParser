package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"fishyAHP/LogParser.git/internal/app"
	"fishyAHP/LogParser.git/internal/core/domain"
	"github.com/chzyer/readline"
)

type Input interface {
	ReadLine(
		prompt string,
	) (string, error)
	Close() error
}

type CLI struct {
	service *app.Service
	input   Input

	out    io.Writer
	errOut io.Writer

	mode  OutputMode
	timer bool

	closeOnce sync.Once
	closeErr  error
}

func New(
	in io.ReadCloser,
	out io.Writer,
	errOut io.Writer,
) (*CLI, error) {
	scheme := &domain.Scheme{
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
	}

	service, err := app.NewService(
		scheme,
		domain.JSON,
		".storage",
	)
	if err != nil {
		return nil, err
	}

	var input Input = NewBasicInput(in, out)

	if file, ok := in.(*os.File); ok &&
		readline.IsTerminal(int(file.Fd())) {

		input, err = NewTerminalInput(
			in,
			out,
			errOut,
		)
		if err != nil {
			return nil, errors.Join(
				err,
				service.Close(),
			)
		}
	}

	return &CLI{
		service: service,
		input:   input,
		mode:    OutputTable,
		out:     out,
		errOut:  errOut,
	}, nil
}

func (c *CLI) Run(ctx context.Context) error {
	var sqlBuffer strings.Builder
	multiline := false

	for {
		if ctx.Err() != nil {
			return nil
		}

		prompt := "logparser> "
		if multiline {
			prompt = "       ... "
		}

		line, err := c.input.ReadLine(prompt)

		if errors.Is(err, readline.ErrInterrupt) {
			if multiline {
				sqlBuffer.Reset()
				multiline = false

				fmt.Fprintln(c.out, "Query canceled")
				continue
			}

			return nil
		}

		if errors.Is(err, io.EOF) {
			if multiline {
				fmt.Fprintln(c.errOut,
					"Query canceled: unexpected EOF")
			}
			return nil
		}

		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("read input: %w", err)
		}

		if ctx.Err() != nil {
			return nil
		}

		line = strings.TrimSpace(line)

		if multiline {
			continuation := strings.HasSuffix(line, "\\")

			if continuation {
				line = strings.TrimSpace(
					strings.TrimSuffix(
						line, "\\",
					),
				)
			}

			if sqlBuffer.Len() > 0 {
				sqlBuffer.WriteByte('\n')
			}
			sqlBuffer.WriteString(line)

			sql, complete, err := completeLSQL(
				sqlBuffer.String(),
			)
			if err != nil {
				c.printCommandError(err)
				sqlBuffer.Reset()
				multiline = false
				continue
			}

			if !complete {
				continue
			}

			sqlBuffer.Reset()
			multiline = false

			c.printCommandError(c.query(sql))
			continue
		}

		if line == "" {
			continue
		}

		command, argument := splitCommand(line)

		if strings.EqualFold(command, "query") {
			if argument == "" {
				multiline = true
				sqlBuffer.Reset()
				continue
			}

			if strings.HasSuffix(argument, "\\") {
				sqlBuffer.Reset()
				sqlBuffer.WriteString(
					strings.TrimSpace(
						strings.TrimSuffix(
							argument, "\\",
						),
					),
				)
				multiline = true
				continue
			}

			sql, complete, err := completeLSQL(argument)
			if err != nil {
				c.printCommandError(err)
				continue
			}

			if !complete {
				sql = argument
			}

			c.printCommandError(c.query(sql))
			continue
		}

		exit, err := c.execute(line)
		if err != nil {
			c.printCommandError(err)
			continue
		}

		if exit {
			return nil
		}
	}
}

func (c *CLI) printCommandError(err error) {
	if err != nil {
		fmt.Fprintln(c.errOut, "Error:", err)
	}
}

func (c *CLI) Close() error {
	c.closeOnce.Do(func() {
		c.closeErr = errors.Join(
			c.input.Close(),
			c.service.Close(),
		)
	})

	return c.closeErr
}
