package storage

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
)

func TestNewSegment_Create(t *testing.T) {
	tmpDir := filepath.Join(
		t.TempDir(),
		time.Now().Format("2006-01-02/15"),
	)

	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatalf("segment mkdir: %s", err)
	}

	seg, err := newSegment(tmpDir, 1)
	if err != nil {
		t.Fatalf("new segment create: %s", err)
	}
	defer seg.Close()

	t.Run("create", func(*testing.T) {
		if seg.ID != 1 {
			t.Fatalf("segment have no wanted id: %d", seg.ID)
		}
		if seg.file.Name() != filepath.Join(tmpDir, fmt.Sprintf("segment-%04d", 1)) {
			t.Fatalf("segment have no wanted path: %s", seg.file.Name())
		}
		if seg.size != 0 {
			t.Fatalf("segment have too big size: %s", seg.size)
		}
		if !seg.IsOpen() {
			t.Fatalf("segment dont open")
		}
	})

	defer os.RemoveAll(tmpDir)
}

func TestNewSegment_OpenExisting(t *testing.T) {
	dir := t.TempDir()

	seg, err := newSegment(dir, 1)
	if err != nil {
		t.Fatalf("create segment: %v", err)
	}

	data := []byte("hello world")

	if _, err := seg.Write(data); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := seg.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	seg, err = newSegment(dir, 1)
	if err != nil {
		t.Fatalf("open existing segment: %v", err)
	}
	defer seg.Close()

	t.Run("segment open", func(t *testing.T) {
		if seg.size != FileSize(len(data)) {
			t.Errorf(
				"expected size %d, got %d",
				len(data),
				seg.size,
			)
		}

		if !seg.IsOpen() {
			t.Error("segment should be open")
		}
	})
}

func TestSegment_Write(t *testing.T) {
	dir := t.TempDir()

	seg, err := newSegment(dir, 1)
	if err != nil {
		t.Fatalf("new segment: %v", err)
	}
	defer seg.Close()

	data := []byte("hello world")

	rd, err := seg.Write(data)
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	t.Run("segment write", func(t *testing.T) {
		if rd == nil {
			t.Fatal("record data is nil")
		}

		if rd.Pointer.SegmentID != 1 {
			t.Errorf(
				"expected segment ID 1, got %d",
				rd.Pointer.SegmentID,
			)
		}

		if rd.Pointer.Offset != 0 {
			t.Errorf(
				"expected offset 0, got %d",
				rd.Pointer.Offset,
			)
		}

		if rd.Pointer.Length != uint32(len(data)) {
			t.Errorf(
				"expected length %d, got %d",
				len(data),
				rd.Pointer.Length,
			)
		}

		if seg.size != FileSize(len(data)) {
			t.Errorf(
				"expected size %d, got %d",
				len(data),
				seg.size,
			)
		}
	})
}

func TestSegment_Read(t *testing.T) {
	dir := t.TempDir()

	seg, err := newSegment(dir, 1)
	if err != nil {
		t.Fatalf("new segment: %v", err)
	}
	defer seg.Close()

	expected := []byte("hello world")

	rd, err := seg.Write(expected)
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	t.Run("segment read", func(t *testing.T) {
		actual, err := seg.Read(rd)
		if err != nil {
			t.Fatalf("read: %v", err)
		}

		if !bytes.Equal(actual, expected) {
			t.Errorf(
				"expected %q, got %q",
				expected,
				actual,
			)
		}
	})
}

func TestSegment_ReadMultiple(t *testing.T) {
	dir := t.TempDir()

	seg, err := newSegment(dir, 1)
	if err != nil {
		t.Fatalf("new segment: %v", err)
	}
	defer seg.Close()

	data := [][]byte{
		[]byte("first"),
		[]byte("second"),
		[]byte("third"),
	}

	records := make([]*domain.RecordData, 0, len(data))

	for _, d := range data {
		rd, err := seg.Write(d)
		if err != nil {
			t.Fatalf("write: %v", err)
		}

		records = append(records, rd)
	}

	for i, rd := range records {
		actual, err := seg.Read(rd)
		if err != nil {
			t.Fatalf("read record %d: %v", i, err)
		}

		if !bytes.Equal(actual, data[i]) {
			t.Errorf(
				"record %d: expected %q, got %q",
				i,
				data[i],
				actual,
			)
		}
	}
}

func TestSegment_ReadWrongSegment(t *testing.T) {
	dir := t.TempDir()

	seg, err := newSegment(dir, 1)
	if err != nil {
		t.Fatalf("new segment: %v", err)
	}
	defer seg.Close()

	rd := &domain.RecordData{
		Pointer: domain.RecordPointer{
			SegmentID: 999,
			Offset:    0,
			Length:    10,
		},
	}

	_, err = seg.Read(rd)

	if err == nil {
		t.Fatal("expected error when reading record from another segment")
	}
}

func TestSegment_Close(t *testing.T) {
	dir := t.TempDir()

	seg, err := newSegment(dir, 1)
	if err != nil {
		t.Fatalf("new segment: %v", err)
	}

	t.Run("close", func(t *testing.T) {
		if !seg.IsOpen() {
			t.Fatal("segment should be open")
		}

		if err := seg.Close(); err != nil {
			t.Fatalf("close: %v", err)
		}

		if seg.IsOpen() {
			t.Error("segment should be closed")
		}
	})
}

func TestSegment_CloseTwice(t *testing.T) {
	dir := t.TempDir()

	seg, err := newSegment(dir, 1)
	if err != nil {
		t.Fatalf("new segment: %v", err)
	}

	t.Run("double close", func(t *testing.T) {
		if err := seg.Close(); err != nil {
			t.Fatalf("first close: %v", err)
		}

		if err := seg.Close(); err == nil {
			t.Error("expected error on second close")
		}
	})
}

func TestSegment_IsOverloaded(t *testing.T) {
	cases := []struct {
		name string
		size FileSize
		data FileSize
		want bool
	}{
		{
			name: "far from limit",
			size: 10 * MByte,
			data: MByte,
			want: false,
		},
		{
			name: "reaches load factor",
			size: FileSize(float64(MaxSegmentSize) * loadFactor),
			data: 0,
			want: true,
		},
		{
			name: "exceeds load factor",
			size: 90 * MByte,
			data: MByte,
			want: true,
		},
	}

	for _, suit := range cases {
		t.Run(suit.name, func(t *testing.T) {
			seg := &segment{
				size: suit.size,
			}

			if got := seg.isOverloaded(suit.data); got != suit.want {
				t.Errorf(
					"isOverloaded() = %v, want %v",
					got,
					suit.want,
				)
			}
		})
	}
}
