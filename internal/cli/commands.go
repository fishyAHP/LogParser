package cli

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"unicode"
)

var ErrUnknownCommand = errors.New("unknown command")

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

	case "exit", "quit":
		return true, nil

	case "query":
		return false, errors.New(
			"SQL queries must be handled by the REPL",
		)

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
  exit                 Exit LogParser

Terminal:
  Up / Down            Command history
  Left / Right         Move cursor
  Tab                  Complete command
  Ctrl+C               Cancel current input
  Ctrl+D               Exit

Example:
  query SELECT COUNT(*) WHERE level="ERROR";

Multiline query:
  query SELECT timestamp, message
  WHERE level="ERROR"
  LIMIT 10;`)

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

func (c *CLI) query(sql string) error {
	if strings.TrimSpace(sql) == "" {
		return errors.New("expected SQL query")
	}

	rows, err := c.service.Query(sql)
	if err != nil {
		return fmt.Errorf("execute query: %w", err)
	}

	count := 0

	for data, err := range rows {
		if err != nil {
			return fmt.Errorf(
				"read query result: %w",
				err,
			)
		}

		if _, err := fmt.Fprintln(
			c.out,
			string(data),
		); err != nil {
			return fmt.Errorf(
				"write query result: %w",
				err,
			)
		}

		count++
	}

	_, err = fmt.Fprintf(
		c.out,
		"(%d rows affected)\n",
		count,
	)

	return err
}

var ErrMultipleStatements = errors.New(
	"only one SQL statement is supported",
)

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
				// SQL-style doubled quotes.
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
				return "", false, ErrMultipleStatements
			}

			return strings.TrimSpace(source[:i]), true, nil
		}
	}

	return "", false, nil
}
