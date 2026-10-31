package text

import (
	"errors"
	"fmt"
	"math"
	"slices"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/binary"
	"fishyAHP/LogParser.git/internal/features/index/structs"
	"fishyAHP/LogParser.git/internal/features/index/text/words"
)

func (i *Index) MarshalBinary() ([]byte, error) {
	if i.count < 0 {
		return nil, fmt.Errorf(
			"negative records count: %d",
			i.count,
		)
	}

	if uint64(len(i.invert)) > math.MaxUint32 {
		return nil, errors.New(
			"entries count overflow",
		)
	}

	tokens := make([]words.Token, 0, len(i.invert))
	for token := range i.invert {
		tokens = append(tokens, token)
	}

	slices.Sort(tokens)

	return indexbin.MarshalIndex(
		uint64(i.count),
		uint32(len(i.invert)),
		func(yield func(
			words.Token,
			*structs.PostingList,
		) bool) {
			for _, token := range tokens {
				if !yield(token, i.invert[token]) {
					return
				}
			}
		},
		func(token words.Token) ([]byte, error) {
			return []byte(token), nil
		},
	)
}

func (i *Index) UnmarshalBinary(data []byte) error {
	tmp := Index{
		invert:    make(invertIndex),
		Tokenizer: i.Tokenizer,
	}

	header, err := indexbin.UnmarshalIndex(
		data,
		func(data []byte) (words.Token, error) {
			key, err := indexbin.DecodeKey[domain.StringValue](data)
			if err != nil {
				return "", fmt.Errorf(
					"decode token: %w",
					err,
				)
			}
			return words.Token(key), nil
		},
		func(
			token words.Token,
			posting *structs.PostingList,
		) error {
			if _, exists := tmp.invert[token]; exists {
				return fmt.Errorf(
					"duplicate token %q",
					token,
				)
			}

			tmp.invert[token] = posting
			return nil
		},
	)

	if err != nil {
		return fmt.Errorf(
			"text index unmarshal: %w",
			err,
		)
	}

	if header.RecordsCount > uint64(math.MaxInt) {
		return fmt.Errorf(
			"records count exceeds int: %d",
			header.RecordsCount,
		)
	}

	tmp.count = int(header.RecordsCount)
	*i = tmp

	return nil
}
