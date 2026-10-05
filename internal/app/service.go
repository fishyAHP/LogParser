package app

import (
	"bufio"
	"bytes"
	"fmt"
	"io"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index"
	"fishyAHP/LogParser.git/internal/features/parse"
	"fishyAHP/LogParser.git/internal/features/query"
	"fishyAHP/LogParser.git/internal/features/storage"
)

type Service struct {
	index    *index.Service
	executor *query.Executor
	parser   *parse.LogParser
	store    *storage.Storage
}

func NewService(
	scheme *domain.Scheme,
	format domain.Format,
	storagePath string,
) (*Service, error) {
	idx, err := index.New(scheme)
	if err != nil {
		return nil, fmt.Errorf("new index: %w", err)
	}

	store, err := storage.New(storagePath)
	if err != nil {
		return nil, fmt.Errorf("new storage: %w", err)
	}

	executor := query.NewExecutor(idx)
	parser := parse.NewParser(scheme, format)

	return &Service{
		index:    idx,
		executor: executor,
		parser:   parser,
		store:    store,
	}, nil
}

func (s *Service) Ingest(raw []byte) error {
	logentry, err := s.parser.Parse(raw)
	if err != nil {
		return fmt.Errorf("parse raw byte: %w", err)
	}

	// TODO: storage write may succeed, but index failed, need rollback/WAL
	record, err := s.store.Write(raw)
	if err != nil {
		return fmt.Errorf("storage write: %w", err)
	}

	if err = s.index.Index(*record, logentry); err != nil {
		return fmt.Errorf("indexing log: %w", err)
	}

	return nil
}

func (s *Service) IngestReader(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(
		make([]byte, 4*storage.KByte),
		int(storage.MByte),
	)

	for scanner.Scan() {
		raw := bytes.Clone(scanner.Bytes())

		if len(bytes.TrimSpace(raw)) == 0 {
			continue
		}

		if err := s.Ingest(raw); err != nil {
			return fmt.Errorf("ingest record: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}
	return nil
}

func (s *Service) Query(q string) ([][]byte, error) {
	records, err := s.executor.Execute(q)
	if err != nil {
		return nil, fmt.Errorf("executing raw: %w", err)
	}

	sl := records.Slice()
	res := make([][]byte, 0, len(sl))
	for _, record := range sl {
		data, err := s.store.Read(&record)
		if err != nil {
			return nil, fmt.Errorf(
				"read record %v: %w",
				record.ID,
				err,
			)
		}

		res = append(res, data)
	}

	return res, nil
}

func (s *Service) Close() error {
	err := s.store.Close()
	if err != nil {
		return fmt.Errorf("storage close: %w", err)
	}

	return nil
}
