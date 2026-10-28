package manifest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"testing"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/storage/segments"
)

func testManifest() Manifest {
	hash := sha256.Sum256([]byte("snapshot data"))

	return Manifest{
		Version:             Version1,
		Generation:          3,
		LastIndexedRecordID: domain.RecordID(1000),
		CreatedAt:           time.Unix(1700000000, 123456789).UTC(),
		Indexes: []indexManifest{
			{
				name:     "level",
				filename: "index-0001.bin",
				size:     segments.FileSize(1024),
				sha256:   hex.EncodeToString(hash[:]),
			},
			{
				name:     "timestamp",
				filename: "index-0002.bin",
				size:     segments.FileSize(8192),
				sha256:   hex.EncodeToString(hash[:]),
			},
		},
	}
}

func TestManifestRoundTrip(t *testing.T) {
	original := testManifest()

	data, err := original.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	var restored Manifest
	if err := restored.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(original, restored) {
		t.Fatalf(
			"manifest mismatch:\nwant: %+v\ngot: %+v",
			original, restored,
		)
	}
}

func TestManifestDeterministic(t *testing.T) {
	m := testManifest()

	first, err := m.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	second, err := m.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(first, second) {
		t.Fatal("binary representation is not deterministic")
	}
}

func TestManifestTruncated(t *testing.T) {
	m := testManifest()

	data, err := m.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	for n := 0; n < len(data); n++ {
		var restored Manifest

		if err := restored.UnmarshalBinary(data[:n]); err == nil {
			t.Fatalf(
				"accepted truncated manifest: length %d of %d",
				n, len(data),
			)
		}
	}
}

func TestManifestTrailingData(t *testing.T) {
	m := testManifest()

	data, err := m.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	data = append(data, 0xFF, 0xAA)

	var restored Manifest
	if err := restored.UnmarshalBinary(data); err == nil {
		t.Fatal("accepted trailing data")
	}
}

func TestManifestPreserveOnError(t *testing.T) {
	original := testManifest()

	before := original

	before.Indexes = append([]indexManifest(nil), original.Indexes...)

	data := []byte{0x01, 0x02, 0x03}

	if err := original.UnmarshalBinary(data); err == nil {
		t.Fatal("expected unmarshal error")
	}

	if !reflect.DeepEqual(original, before) {
		t.Fatal("manifest changed after failed unmarshal")
	}
}
