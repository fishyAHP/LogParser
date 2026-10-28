package app

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index"
	"fishyAHP/LogParser.git/internal/features/parse"
	"fishyAHP/LogParser.git/internal/features/query"
	"fishyAHP/LogParser.git/internal/features/query/iterators"
	"fishyAHP/LogParser.git/internal/features/storage"
	"fishyAHP/LogParser.git/internal/features/storage/segments"
)

type Service struct {
	index  *index.Manager
	engine *query.Engine
	parser *parse.LogParser
	store  *storage.Storage

	catalog *domain.FieldsCatalog
}

type IndexManager interface {
	Index(
		record domain.RecordID,
		entry domain.LogEntry,
	) error
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

	parser := parse.NewParser(scheme, format)

	engine, err := query.NewEngine(
		store,
		parser,
		idx,
		catalog,
	)
	if err != nil {
		_ = store.Close()
		return nil, fmt.Errorf(
			"new query engine: %w",
			err,
		)
	}

	service := &Service{
		parser:  parser,
		store:   store,
		index:   idx,
		engine:  engine,
		catalog: catalog,
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
		return fmt.Errorf(
			"parse raw byte: %w",
			err,
		)
	}

	// TODO: storage write may succeed, but index failed, need rollback/WAL
	record, err := s.store.Write(raw)
	if err != nil {
		return fmt.Errorf(
			"storage write: %w",
			err,
		)
	}

	if err = s.index.Index(record, parsed.Entry); err != nil {
		return fmt.Errorf(
			"indexing log: %w",
			err,
		)
	}

	s.catalog.Add(parsed.Fields...)

	return nil
}

func (s *Service) IngestReader(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(
		make([]byte, 4*segments.KByte),
		int(segments.MByte),
	)

	for scanner.Scan() {
		raw := bytes.Clone(scanner.Bytes())

		if len(bytes.TrimSpace(raw)) == 0 {
			continue
		}

		if err := s.Ingest(raw); err != nil {
			return fmt.Errorf(
				"ingest record: %w",
				err,
			)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf(
			"scanner error: %w",
			err,
		)
	}
	return nil
}

func (s *Service) Query(q string) (iterators.Rows, error) {
	rows, err := s.engine.Query(q)
	if err != nil {
		return nil, fmt.Errorf(
			"executing raw: %w",
			err,
		)
	}

	return rows, nil
}

func (s *Service) Close() error {
	err := s.store.Close()
	if err != nil {
		return fmt.Errorf(
			"storage close: %w",
			err,
		)
	}

	return nil
}
