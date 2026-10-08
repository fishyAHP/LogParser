package aggregation

import (
	"fmt"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index"
	"fishyAHP/LogParser.git/internal/features/index/common"
	"fishyAHP/LogParser.git/internal/features/index/structs"
)

type Aggregator struct {
	indexes *index.Manager
}

func (a *Aggregator) Count(posting *structs.PostingList) int {
	return posting.Len()
}

func (a *Aggregator) GroupBy(
	idxName string,
	postings *structs.PostingList,
) (common.Groups, error) {
	grouper, err := a.indexes.Grouper(idxName)
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

func (a *Aggregator) CountBy(
	idxName string,
	postings *structs.PostingList,
) (CountStats, error) {
	groups, err := a.GroupBy(idxName, postings)
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

func (a *Aggregator) Min(
	idxName string,
	posting *structs.PostingList,
) (domain.Value, error) {
	aggregator, err := a.indexes.Aggregator(idxName)
	if err != nil {
		return nil, fmt.Errorf(
			"find aggregator: %w",
			err,
		)
	}

	return aggregator.Min(posting)
}

func (a *Aggregator) Max(
	idxName string,
	posting *structs.PostingList,
) (domain.Value, error) {
	aggregator, err := a.indexes.Aggregator(idxName)
	if err != nil {
		return nil, fmt.Errorf(
			"find aggregator: %w",
			err,
		)
	}

	return aggregator.Max(posting)
}
