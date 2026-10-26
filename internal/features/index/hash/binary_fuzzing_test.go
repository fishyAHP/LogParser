package hash

import (
	"testing"

	"fishyAHP/LogParser.git/internal/core/domain"
)

func FuzzIndexUnmarshalBinary(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, 12))
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 255, 255})

	f.Fuzz(func(t *testing.T, data []byte) {
		// Не проверяем память на огромных входах в этом тесте.
		if len(data) > 1<<20 {
			t.Skip()
		}

		index := New[domain.StringValue]()

		if err := index.UnmarshalBinary(data); err != nil {
			return
		}

		encoded, err := index.MarshalBinary()
		if err != nil {
			t.Fatalf("marshal after successful unmarshal: %v", err)
		}

		restored := New[domain.StringValue]()
		if err := restored.UnmarshalBinary(encoded); err != nil {
			t.Fatalf("second unmarshal: %v", err)
		}
	})
}
