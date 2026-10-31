package indexbin

import (
	"encoding/binary"
	"testing"
)

func FuzzHeaderUnmarshalBinary(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, HeaderSize))
	f.Add(make([]byte, HeaderSize-1))

	valid := make([]byte, HeaderSize+MinEntrySize)

	binary.BigEndian.PutUint64(
		valid[:RecordsCountSize],
		1,
	)
	binary.BigEndian.PutUint32(
		valid[RecordsCountSize:HeaderSize],
		1,
	)
	f.Add(valid)

	f.Fuzz(func(t *testing.T, data []byte) {
		var header Header

		err := header.UnmarshalBinary(data)
		if err != nil {
			return
		}

		if len(data) < HeaderSize {
			t.Fatal("accepted truncated header")
		}

		if uint64(header.EntriesCount) >
			uint64(len(data)-HeaderSize)/MinEntrySize {
			t.Fatal("accepted impossible entries count")
		}

		encoded, err := header.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}

		if len(encoded) != HeaderSize {
			t.Fatal("invalid encoded header size")
		}

		for j := range encoded {
			if encoded[j] != data[j] {
				t.Fatalf("header round-trip differs at byte %d", j)
			}
		}
	})
}
