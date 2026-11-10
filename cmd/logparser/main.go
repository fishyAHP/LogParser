package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"

	"fishyAHP/LogParser.git/internal/cli"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() (runErr error) {
	c, err := cli.New(os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		return err
	}

	defer func() {
		if err = c.Close(); err != nil {
			runErr = errors.Join(runErr, err)
		}
	}()

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer cancel()

	if err = c.Run(ctx); err != nil {
		return err
	}

	if err = writeHeapProfile("heap.pprof"); err != nil {
		return err
	}

	return nil
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
