package metadata

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"strconv"
	"testing"
)

func TestFieldsCatalogRoundTrip(t *testing.T) {
	original := &FieldsCatalog{
		catalog: map[string]uint64{
			"level":   100,
			"pid":     50,
			"message": 75,
		},
	}

	data, err := original.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	restored := &FieldsCatalog{}
	if err := restored.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(original.catalog, restored.catalog) {
		t.Fatalf(
			"catalog mismatch: got %v, want %v",
			restored.catalog,
			original.catalog,
		)
	}
}

func TestFieldsCatalogDeterministic(t *testing.T) {
	catalog := &FieldsCatalog{
		catalog: map[string]uint64{
			"message": 10,
			"level":   20,
			"pid":     30,
		},
	}

	first, err := catalog.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	second, err := catalog.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(first, second) {
		t.Fatal("serialization is not deterministic")
	}
}

func TestFieldsCatalogTruncated(t *testing.T) {
	original := &FieldsCatalog{
		catalog: map[string]uint64{
			"level": 100,
			"pid":   50,
		},
	}

	data, err := original.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(data); i++ {
		t.Run(
			"cut_"+strconv.Itoa(i),
			func(t *testing.T) {
				restored := &FieldsCatalog{}

				if err := restored.UnmarshalBinary(data[:i]); err == nil {
					t.Fatalf(
						"expected error for truncated data at %d",
						i,
					)
				}
			},
		)
	}
}

func TestFieldsCatalogPreserveOnError(t *testing.T) {
	catalog := &FieldsCatalog{
		catalog: map[string]uint64{
			"existing": 42,
		},
	}

	// В заголовке указано одно поле, но данных нет.
	invalid := make([]byte, 4)
	binary.BigEndian.PutUint32(invalid, 1)

	if err := catalog.UnmarshalBinary(invalid); err == nil {
		t.Fatal("expected error")
	}

	if !reflect.DeepEqual(
		catalog.catalog,
		map[string]uint64{"existing": 42},
	) {
		t.Fatal("catalog changed after failed unmarshal")
	}
}
