package app

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index"
	"fishyAHP/LogParser.git/internal/features/metadata"
	"fishyAHP/LogParser.git/internal/features/parse"
	"fishyAHP/LogParser.git/internal/features/query"
	"fishyAHP/LogParser.git/internal/features/query/iterators"
	"fishyAHP/LogParser.git/internal/features/snapshot"
	"fishyAHP/LogParser.git/internal/features/storage"
	"fishyAHP/LogParser.git/internal/features/storage/segments"
)

type Service struct {
	index   *index.Manager
	engine  *query.Engine
	parser  *parse.LogParser
	storage *storage.Storage
	catalog *metadata.FieldsCatalog

	metadata *Metadata

	lastIndexedID domain.RecordID
}

func NewService(
	scheme *domain.Scheme,
	format domain.Format,
	path string,
) (*Service, error) {
	service, err := newService(
		scheme,
		format,
		path,
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
	path string,
) (*Service, error) {
	if err := scheme.Validate(); err != nil {
		return nil, fmt.Errorf(
			"invalid scheme: %w",
			err,
		)
	}

	catalog := metadata.NewCatalog()

	start := time.Now()
	store, err := storage.New(
		filepath.Join(
			path,
			"logs",
		))
	fmt.Println("Storage recovery:", time.Since(start))
	if err != nil {
		return nil, fmt.Errorf(
			"new storage: %w",
			err,
		)
	}

	snapshotStore, err := snapshot.NewStore(
		filepath.Join(
			path,
			"snapshots",
		))
	if err != nil {
		return nil, fmt.Errorf(
			"new snapshot store: %w",
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

	meta := NewMetadata(
		catalog,
		snapshotStore,
		idx,
	)

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
		parser:   parser,
		storage:  store,
		index:    idx,
		engine:   engine,
		catalog:  catalog,
		metadata: meta,
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
	start := time.Now()

	if err := s.metadata.Load(); err != nil {
		if !errors.Is(err, snapshot.ErrNoSnapshots) &&
			!errors.Is(err, snapshot.ErrNoValidSnapshots){
			return fmt.Errorf(
				"load indexes: %w",
				err,
			)
		}

		fmt.Println("No valid snapshots, rebuilding metadata from storage")

		s.index.Clear()
		s.catalog.Clear()
	}

	fmt.Println("Snapshot load:", time.Since(start))
	start = time.Now()

	lastIndexedID := s.metadata.Checkpoint()
	s.lastIndexedID = lastIndexedID

	fmt.Printf("Loaded checkpoint: %d\n", lastIndexedID)

	iter, err := s.storage.IteratorFromID(lastIndexedID+1)
	if err != nil {
		return fmt.Errorf(
			"create iterator from id: %w",
			err,
		)
	}

	var reindexed int
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

		reindexed++
		s.catalog.Add(parsed.Fields...)
		s.lastIndexedID = iter.RecordID()
	}

	fmt.Printf("indexed after snapshot: %d\n", reindexed)
	fmt.Println("Metadata recovery:", time.Since(start))

	if err := iter.Err(); err != nil {
		return fmt.Errorf(
			"iterator err: %w",
			err,
		)
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
	record, err := s.storage.Write(raw)
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

	s.lastIndexedID = record
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
	if err := s.Save(); err != nil {
		return fmt.Errorf(
			"service save: %w",
			err,
		)
	}

	if err := s.storage.Close(); err != nil {
		return fmt.Errorf(
			"storage close: %w",
			err,
		)
	}

	return nil
}

func (s *Service) Save() error {
	if err := s.metadata.Save(
		s.lastIndexedID,
	); err != nil {
		return fmt.Errorf(
			"saving snapshot: %w",
			err,
		)
	}

	return nil
}
