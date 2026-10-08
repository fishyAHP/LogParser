package aggregation

import (
	"fmt"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index"
	"fishyAHP/LogParser.git/internal/features/index/common"
	"fishyAHP/LogParser.git/internal/features/index/structs"
)

type Service struct {
	indexes *index.Service
}

func (s *Service) Count(posting *structs.PostingList) int {
	return posting.Len()
}

func (s *Service) GroupBy(
	idxName string,
	postings *structs.PostingList,
) (common.Groups, error) {
	grouper, err := s.indexes.Grouper(idxName)
	if err != nil {
		return nil, fmt.Errorf(
			"index grouper: %w",
			err,
		)
	}

	res, err := grouper.Group(postings)
	if err != nil {
		return nil, fmt.Errorf(
			"group postings: %w",
			err,
		)
	}

	return res, nil
}

type CountStat struct {
	Value domain.Value
	Count int
}

type CountStats []CountStat

func (s *Service) CountBy(
	idxName string,
	postings *structs.PostingList,
) (CountStats, error) {
	groups, err := s.GroupBy(idxName, postings)
	if err != nil {
		return nil, fmt.Errorf(
			"group by: %w",
			err,
		)
	}

	stats := make(CountStats, 0, len(groups))
	for _, group := range groups {
		stats = append(stats, CountStat{
			Value: group.Value,
			Count: group.Records.Len(),
		})
	}

	return stats, nil
}

func (s *Service) Min(
	idxName string,
	posting *structs.PostingList,
) (domain.Value, error) {
	aggregator, err := s.indexes.Aggregator(idxName)
	if err != nil {
		return nil, fmt.Errorf(
			"find aggregator: %w",
			err,
		)
	}

	return aggregator.Min(posting)
}

func (s *Service) Max(
	idxName string,
	posting *structs.PostingList,
) (domain.Value, error) {
	aggregator, err := s.indexes.Aggregator(idxName)
	if err != nil {
		return nil, fmt.Errorf(
			"find aggregator: %w",
			err,
		)
	}

	return aggregator.Max(posting)
}
