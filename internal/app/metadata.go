package app

import (
	"errors"
	"fmt"
	"iter"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index"
	"fishyAHP/LogParser.git/internal/features/metadata"
	"fishyAHP/LogParser.git/internal/features/snapshot"
)

type Metadata struct {
	catalog *metadata.FieldsCatalog
	store   *snapshot.Store
	manager *index.Manager
}

func NewMetadata(
	catalog *metadata.FieldsCatalog,
	store *snapshot.Store,
	manager *index.Manager,
) *Metadata {
	return &Metadata{
		catalog: catalog,
		store:   store,
		manager: manager,
	}
}

func (m *Metadata) Save(
	lastID domain.RecordID,
) error {
	snapshots := func(
		yield func(index.Snapshot, error) bool) {
		for s, err := range m.manager.MakeSnapshots() {
			if err != nil {
				yield(index.Snapshot{}, err)
				return
			}

			if !yield(s, nil) {
				return
			}
		}

		data, err := m.catalog.MarshalBinary()
		if err != nil {
			yield(index.Snapshot{}, err)
			return
		}

		yield(
			index.Snapshot{
				Name: "catalog",
				Data: data,
			}, nil)
	}

	if err := m.store.Save(
		snapshots,
		lastID,
	); err != nil {
		return fmt.Errorf(
			"save snapshots: %w",
			err,
		)
	}
	return nil
}

func (m *Metadata) Load() error {
	if err := m.store.Load(m); err != nil {
		return fmt.Errorf(
			"load metadata: %w",
			err,
		)
	}

	return nil
}

func (m *Metadata) FromSnapshots(
	snapshots iter.Seq2[index.Snapshot, error],
) error {
	var catalogData []byte
	catalogFound := false

	indexes := func(
		yield func(index.Snapshot, error) bool) {
		for s, err := range snapshots {
			if err != nil {
				yield(index.Snapshot{}, err)
				return
			}

			if s.Name == "catalog" {
				if catalogFound {
					yield(index.Snapshot{},
						errors.New(
							"duplicate catalog snapshot",
						),
					)
				}

				catalogFound = true
				catalogData = s.Data
				continue
			}

			if !yield(s, nil) {
				return
			}
		}
	}

	if err := m.manager.FromSnapshots(
		indexes,
	); err != nil {
		return fmt.Errorf(
			"restore indexes: %w",
			err,
		)
	}

	if !catalogFound {
		return snapshot.ErrNoSnapshots
	}

	if err := m.catalog.UnmarshalBinary(
		catalogData,
	); err != nil {
		return fmt.Errorf(
			"restore catalog: %w",
			err,
		)
	}

	return nil
}

func (m *Metadata) Checkpoint() domain.RecordID {
	lastID, _ := m.store.Checkpoint()
	return lastID
}
