package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"

	"fishyAHP/LogParser.git/internal/app"
	"fishyAHP/LogParser.git/internal/core/domain"
)

func main() {
	c, err := New(os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err := c.service.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	if err := writeHeapProfile("heap.pprof"); err != nil {
		log.Fatal(err)
	}

	if err := c.Run(); err != nil {
		log.Fatal(err)
	}
}

func writeHeapProfile(path string) error {
	runtime.GC()

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return pprof.WriteHeapProfile(f)
}

type Cli struct {
	service *app.Service

	in     io.Reader
	out    io.Writer
	errOut io.Writer
}

var ErrUnknownCommand = errors.New("unknown command")

func New(
	in io.Reader,
	out io.Writer,
	errOut io.Writer,
) (*Cli, error) {
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
		},
	},
		domain.JSON,
		".storage/logs",
	)

	if err != nil {
		return nil, err
	}

	return &Cli{
		service: service,
		in:      in,
		out:     out,
		errOut:  errOut,
	}, nil
}

func (c *Cli) Run() error {
	for {
		fmt.Fprint(c.out, "Enter a command: ")
		data := make([]byte, 1024)
		n, err := c.in.Read(data)

		if err != nil {
			return err
		}
		line := strings.TrimSpace(string(data[:n]))

		exit, err := c.execute(line)
		if err != nil {
			if errors.Is(err, ErrUnknownCommand) {
				fmt.Fprint(c.errOut, err.Error()+"\n")
				continue
			}
			return err
		}
		if exit {
			break
		}
	}
	return nil
}

func (c *Cli) execute(line string) (bool, error) {
	command, argument, _ := strings.Cut(line, " ")

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

func (c *Cli) ingest(raw string) error {
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

func (c *Cli) ingestFile(path string) error {
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

func (c *Cli) ingestTCP(address string) error {
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

func (c *Cli) query(query string) error {
	if strings.TrimSpace(query) == "" {
		return errors.New(
			"expect not empty argument",
		)
	}

	logs, err := c.service.Query(query)
	if err != nil {
		return fmt.Errorf(
			"query %q: %w",
			query,
			err,
		)
	}

	for i, logg := range logs {
		fmt.Fprintf(c.out, "%d) %s\n", i+1, string(logg))
	}
	fmt.Fprintf(c.out, "%d results\n", len(logs))
	return nil
}
