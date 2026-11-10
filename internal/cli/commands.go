package cli

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
	"unicode"
)

var (
	ErrUnknownCommand = errors.New("unknown command")

	ErrMultipleStatements = errors.New("only one SQL statement is supported")
)

func splitCommand(line string) (string, string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", ""
	}

	idx := strings.IndexFunc(line, unicode.IsSpace)
	if idx < 0 {
		return line, ""
	}

	return line[:idx], strings.TrimSpace(line[idx:])
}

func (c *CLI) execute(line string) (bool, error) {
	command, argument := splitCommand(line)

	switch strings.ToLower(command) {
	case "":
		return false, nil

	case "ingest":
		return false, c.ingest(argument)

	case "ingest-file":
		return false, c.ingestFile(argument)

	case "ingest-tcp":
		return false, c.ingestTCP(argument)

	case "help":
		return false, c.help()

	case "exit", "quit", "q":
		return true, nil

	case "query":
		return false, errors.New(
			"SQL queries must be handled by the REPL",
		)

	case "mode":
		return false, c.setOutputMode(argument)

	case "timer":
		return false, c.setTimer(argument)

	default:
		return false, fmt.Errorf(
			"%w: %q",
			ErrUnknownCommand,
			command,
		)
	}
}

func (c *CLI) help() error {
	_, err := fmt.Fprintln(c.out, `LogParser commands:

  query <SQL>;         Execute SQL query
  ingest <JSON>        Ingest a single JSON log
  ingest-file <PATH>   Ingest logs from file
  ingest-tcp <ADDR>    Ingest logs from TCP connection
  help                 Show available commands
  exit, quit, q        Exit LogParser

Terminal:
  Up / Down            Command history
  Left / Right         Move cursor
  Tab                  Complete command
  Ctrl+C               Cancel current input
  Ctrl+D               Exit

Output:
  mode json          Print results as JSON
  mode table         Print results as a table
  timer on           Show query execution time
  timer off          Hide query execution time
`)

	return err
}

func (c *CLI) ingest(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return errors.New("expected JSON log")
	}

	if err := c.service.Ingest([]byte(raw)); err != nil {
		return fmt.Errorf("ingest: %w", err)
	}

	return nil
}

func (c *CLI) ingestFile(path string) (retErr error) {
	if path == "" {
		return errors.New("expected file path")
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf(
			"open file %q: %w",
			path,
			err,
		)
	}

	defer func() {
		if err := file.Close(); err != nil {
			retErr = errors.Join(
				retErr,
				fmt.Errorf("close file: %w", err),
			)
		}
	}()

	if err := c.service.IngestReader(file); err != nil {
		return fmt.Errorf(
			"ingest file %q: %w",
			path,
			err,
		)
	}

	return nil
}

func (c *CLI) ingestTCP(
	address string,
) (retErr error) {
	if address == "" {
		return errors.New("expected TCP address")
	}

	conn, err := net.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf(
			"connect to %q: %w",
			address,
			err,
		)
	}

	defer func() {
		if err := conn.Close(); err != nil {
			retErr = errors.Join(
				retErr,
				fmt.Errorf("close TCP connection: %w", err),
			)
		}
	}()

	if err := c.service.IngestReader(conn); err != nil {
		return fmt.Errorf(
			"ingest TCP %q: %w",
			address,
			err,
		)
	}

	return nil
}

func (c *CLI) query(sql string) (retErr error) {
	if strings.TrimSpace(sql) == "" {
		return errors.New("expected SQL query")
	}

	start := time.Now()

	if c.timer {
		defer func() {
			_, err := fmt.Fprintf(
				c.out,
				"Execution time: %s\n",
				time.Since(start).Round(time.Microsecond),
			)
			retErr = errors.Join(retErr, err)
		}()
	}

	rows, columns, err := c.service.QueryWithColumns(sql)
	if err != nil {
		return fmt.Errorf("execute query: %w", err)
	}

	count := 0
	truncated := false

	switch c.mode {
	case OutputJSON:
		for data, err := range rows {
			if err != nil {
				return fmt.Errorf(
					"read query result: %w", err,
				)
			}

			if _, err := fmt.Fprintln(
				c.out,
				string(data),
			); err != nil {
				return fmt.Errorf(
					"write result: %w", err,
				)
			}

			count++
		}

	case OutputTable:
		table := NewTablePrinter(columns...)

		for data, err := range rows {
			if err != nil {
				return fmt.Errorf(
					"read query result: %w", err,
				)
			}

			if count == maxTableRows {
				truncated = true
				break
			}

			if err := table.AddRow(data); err != nil {
				return err
			}

			count++
		}

		if err := table.Print(c.out); err != nil {
			return fmt.Errorf(
				"print table: %w", err,
			)
		}

	default:
		return fmt.Errorf(
			"unknown output mode: %s",
			c.mode,
		)
	}

	if _, err := fmt.Fprintf(
		c.out,
		"\n(%d rows displayed)\n",
		count,
	); err != nil {
		return err
	}

	if truncated {
		_, err := fmt.Fprintf(
			c.out,
			"Output truncated after %d rows. "+
				"Use LIMIT in SQL to restrict results.\n",
			maxTableRows,
		)
		return err
	}

	return nil
}

func completeLSQL(
	source string,
) (sql string, complete bool, err error) {
	var quote byte

	for i := 0; i < len(source); i++ {
		ch := source[i]

		if quote != 0 {
			if ch == '\\' && i+1 < len(source) {
				i++
				continue
			}

			if ch == quote {
				if i+1 < len(source) &&
					source[i+1] == quote {
					i++
					continue
				}
				quote = 0
			}

			continue
		}

		switch ch {
		case '\'', '"', '`':
			quote = ch

		case ';':
			if strings.TrimSpace(source[i+1:]) != "" {
				return "",
					false,
					ErrMultipleStatements
			}

			return strings.TrimSpace(source[:i]),
				true,
				nil
		}
	}

	return "", false, nil
}

func (c *CLI) setOutputMode(argument string) error {
	switch strings.ToLower(strings.TrimSpace(argument)) {
	case "json":
		c.mode = OutputJSON

	case "table":
		c.mode = OutputTable

	default:
		return fmt.Errorf(
			"unknown output mode %q: expected json or table",
			argument,
		)
	}

	_, err := fmt.Fprintf(
		c.out,
		"Output mode: %s\n",
		c.mode,
	)
	return err
}

func (c *CLI) setTimer(argument string) error {
	switch strings.ToLower(strings.TrimSpace(argument)) {
	case "on":
		c.timer = true

	case "off":
		c.timer = false

	default:
		return fmt.Errorf(
			"invalid timer value %q: expected on or off",
			argument,
		)
	}

	status := "disabled"
	if c.timer {
		status = "enabled"
	}

	_, err := fmt.Fprintf(
		c.out,
		"Timer %s\n",
		status,
	)
	return err
}
