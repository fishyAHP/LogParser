package text

import (
	"bytes"
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
	if uint64(len(i.invert)) > math.MaxUint32 {
		return nil, fmt.Errorf(
			"entries count overflowed: %d",
			len(i.invert),
		)
	}

	header := indexbin.Header{
		RecordsCount: uint64(i.count),
		EntriesCount: uint32(len(i.invert)),
	}

	headerData, err := header.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf(
			"text index header binary marshal: %w",
			err,
		)
	}

	var buf bytes.Buffer
	if _, err = buf.Write(
		headerData,
	); err != nil {
		return nil, fmt.Errorf(
			"buffer write header data: %w",
			err,
		)
	}

	tokens := make([]words.Token, 0, len(i.invert))
	for token := range i.invert {
		tokens = append(tokens, token)
	}

	slices.Sort(tokens)

	var actual int
	for _, token := range tokens {
		tokenData := []byte(token)

		posting := i.invert[token]
		if posting == nil ||
			posting.Len() == 0 {
			return nil, fmt.Errorf(
				"empty posting for token %q",
				token,
			)
		}

		postData, err := posting.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf(
				"posting marshal binary: %w",
				err,
			)
		}

		actual += posting.Len()

		entry := indexbin.BinaryEntry{
			KeyLength:    uint32(len(tokenData)),
			Key:          tokenData,
			RecordsCount: uint64(posting.Len()),
			Records:      postData,
		}

		data, err := entry.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf(
				"entry binary marshal: %w",
				err,
			)
		}

		if _, err = buf.Write(
			data,
		); err != nil {
			return nil, fmt.Errorf(
				"buffer write entry data: %w",
				err,
			)
		}
	}

	if actual != i.count {
		return nil, fmt.Errorf(
			"records count not consistently: %d and %d",
			actual,
			i.count,
		)
	}

	return buf.Bytes(), nil
}

func (i *Index) UnmarshalBinary(
	data []byte,
) error {
	if len(data) < indexbin.HeaderSize {
		return fmt.Errorf(
			"data dont accommodate header: got %d, want %d",
			len(data),
			indexbin.HeaderSize,
		)
	}

	var header indexbin.Header
	if err := header.UnmarshalBinary(
		data,
	); err != nil {
		return fmt.Errorf(
			"text index header binary unmarshal: %w",
			err,
		)
	}

	tmp := Index{
		invert:    make(invertIndex),
		Tokenizer: i.Tokenizer,
	}

	idx := indexbin.HeaderSize
	var actualCount uint64
	for range header.EntriesCount {
		var entry indexbin.BinaryEntry
		if err := entry.UnmarshalBinary(
			data[idx:],
		); err != nil {
			return fmt.Errorf(
				"entry binary unmarshal: %w",
				err,
			)
		}

		key, err := indexbin.DecodeKey[domain.StringValue](
			entry.Key,
		)
		if err != nil {
			return fmt.Errorf(
				"decode string key: %w",
				err,
			)
		}

		if entry.RecordsCount == 0 {
			return fmt.Errorf(
				"empty posting list for token %q",
				key,
			)
		}

		posting := structs.NewPostingLists(
			int(entry.RecordsCount),
		)

		if err = posting.UnmarshalBinary(
			entry.Records,
		); err != nil {
			return fmt.Errorf(
				"posting with key='%s' unmarshal: %w",
				key,
				err,
			)
		}

		token := words.Token(key)
		if _, ok := tmp.invert[token]; ok {
			return fmt.Errorf(
				"key '%s' already exist",
				token,
			)
		}

		tmp.invert[token] = posting

		size := indexbin.KeyLengthSize +
			len(entry.Key) +
			indexbin.RecordsCountSize +
			len(entry.Records)

		if entry.RecordsCount > header.RecordsCount-actualCount {
			return errors.New(
				"records count exceeds header",
			)
		}
		actualCount += entry.RecordsCount

		actualCount += entry.RecordsCount
		idx += size
	}

	if idx != len(data) {
		return fmt.Errorf(
			"unexpected trailing data: %d bytes",
			len(data)-idx,
		)
	}

	if actualCount != header.RecordsCount {
		return fmt.Errorf(
			"records count mismatch: expected %d, got %d",
			header.RecordsCount,
			actualCount,
		)
	}

	*i = tmp

	return nil
}
