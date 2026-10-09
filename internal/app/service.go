package app

import (
	"bufio"
	"bytes"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index"
	"fishyAHP/LogParser.git/internal/features/parse"
	"fishyAHP/LogParser.git/internal/features/query"
	"fishyAHP/LogParser.git/internal/features/storage"
)

type Service struct {
	index    *index.Manager
	executor *query.Executor
	parser   *parse.LogParser
	store    *storage.Storage

	catalog *domain.FieldsCatalog
}

func NewService(
	scheme *domain.Scheme,
	format domain.Format,
	storagePath string,
) (*Service, error) {
	service, err := newService(
		scheme,
		format,
		storagePath,
	)
	if err != nil {
		return nil, err
	}

	return service, nil
}

var ErrIndexRebuild = errors.New("rebuild indexes")

func newService(
	scheme *domain.Scheme,
	format domain.Format,
	storagePath string,
) (*Service, error) {
	catalog := domain.NewCatalog()

	store, err := storage.New(storagePath)
	if err != nil {
		return nil, fmt.Errorf(
			"new storage: %w",
			err,
		)
	}

	idx, err := index.New(scheme)
	if err != nil {
		_ = store.Close()
		return nil, fmt.Errorf(
			"new index: %w",
			err,
		)
	}

	executor := query.NewExecutor(idx, catalog)
	parser := parse.NewParser(scheme, format)

	service := &Service{
		parser:   parser,
		store:    store,
		index:    idx,
		executor: executor,
		catalog:  catalog,
	}

	if err = service.rebuildMetadata(); err != nil {
		_ = store.Close()
		return nil, fmt.Errorf(
			"%w: %w",
			ErrIndexRebuild,
			err,
		)
	}

	return service, nil
}

func (s *Service) RebuildIndexes() error {
	s.index.Clear()
	s.catalog.Clear()

	if err := s.rebuildMetadata(); err != nil {
		return fmt.Errorf(
			"%w: %w",
			ErrIndexRebuild,
			err,
		)
	}
	return nil
}

func (s *Service) rebuildMetadata() error {
	iter := s.store.Iterator()
	for iter.Next() {
		data := iter.Data()

		parsed, err := s.parser.Parse(data)
		if err != nil {
			return fmt.Errorf(
				"parse iter data: %w",
				err,
			)
		}

		if err := s.index.Index(
			iter.RecordID(),
			parsed.Entry,
		); err != nil {
			return fmt.Errorf(
				"indexing record: %w",
				err,
			)
		}

		s.catalog.Add(parsed.Fields...)
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("iterator err: %w", err)
	}

	return nil
}

func (s *Service) Ingest(raw []byte) error {
	parsed, err := s.parser.Parse(raw)
	if err != nil {
		return fmt.Errorf("parse raw byte: %w", err)
	}

	// TODO: storage write may succeed, but index failed, need rollback/WAL
	record, err := s.store.Write(raw)
	if err != nil {
		return fmt.Errorf("storage write: %w", err)
	}

	if err = s.index.Index(record, parsed.Entry); err != nil {
		return fmt.Errorf("indexing log: %w", err)
	}

	s.catalog.Add(parsed.Fields...)

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
	result, err := s.executor.Execute(q)
	if err != nil {
		return nil, fmt.Errorf("executing raw: %w", err)
	}

	if result.FullScan {
		iterator := s.store.Iterator()
		res := make([][]byte, 0)

		for iterator.Next() {
			data, err := s.project(iterator.Data(), result.Fields)
			if err != nil {
				return nil, fmt.Errorf(
					"full scan projection: %w",
					err,
				)
			}

			res = append(res, bytes.Clone(data))
		}

		if err = iterator.Err(); err != nil {
			return nil, fmt.Errorf(
				"storage iterator: %w",
				err,
			)
		}

		return res, nil
	}

	sl := result.Posting.Slice()
	res := make([][]byte, 0, len(sl))
	for _, record := range sl {
		data, err := s.store.ReadByID(record)
		if err != nil {
			if errors.Is(err, storage.ErrRecordNotFound) {
				continue
			}
			return nil, fmt.Errorf(
				"read record %v: %w",
				record,
				err,
			)
		}

		data, err = s.project(data, result.Fields)
		if err != nil {
			return nil, fmt.Errorf(
				"with index projection: %w",
				err,
			)
		}
		res = append(res, data)
	}

	return res, nil
}

func (s *Service) project(
	data []byte,
	fields []string,
) ([]byte, error) {
	if len(fields) == 0 ||
		(len(fields) == 1 && fields[0] == "*") {
		return data, nil
	}

	projections, err := s.parser.Project(data, fields)
	if err != nil {
		return nil, fmt.Errorf(
			"parser project: %w",
			err,
		)
	}

	res, err := json.Marshal(projections)
	if err != nil {
		return nil, fmt.Errorf(
			"json marshal: %w",
			err,
		)
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
