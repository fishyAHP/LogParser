package storage

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
)

func testStorageDir(t *testing.T) string {
	t.Helper()

	return filepath.Join(
		t.TempDir(),
		"storage",
		"logs",
		time.Now().Format("2006-01-02"),
		time.Now().Format("15"),
	)
}

func TestStorage_New(t *testing.T) {
	dir := testStorageDir(t)

	s, err := New(dir)
	if err != nil {
		t.Fatalf("new storage: %v", err)
	}

	defer s.Close()

	t.Run("create", func(t *testing.T) {
		if s.writeSegment == nil {
			t.Fatal("write segment is nil")
		}

		if !s.writeSegment.IsOpen() {
			t.Error("write segment should be open")
		}

		if s.writeSegment.ID != 1 {
			t.Errorf(
				"expected write segment ID 1, got %d",
				s.writeSegment.ID,
			)
		}
	})
}

func TestStorage_Write(t *testing.T) {
	dir := testStorageDir(t)

	s, err := New(dir)
	if err != nil {
		t.Fatalf("new storage: %v", err)
	}
	defer s.Close()

	data := []byte("hello world")

	rd, err := s.Write(data)
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	t.Run("storage write", func(t *testing.T) {
		if rd == nil {
			t.Fatal("record data is nil")
		}

		if rd.Pointer.Length != uint32(len(data)) {
			t.Errorf(
				"expected length %d, got %d",
				len(data),
				rd.Pointer.Length,
			)
		}

		if rd.Pointer.SegmentID != s.writeSegment.ID {
			t.Errorf(
				"record belongs to segment %d, writer is %d",
				rd.Pointer.SegmentID,
				s.writeSegment.ID,
			)
		}
	})
}

func TestStorage_WriteRead(t *testing.T) {
	dir := testStorageDir(t)

	s, err := New(dir)
	if err != nil {
		t.Fatalf("new storage: %v", err)
	}
	defer s.Close()

	expected := []byte("hello from storage")

	rd, err := s.Write(expected)
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	actual, err := s.Read(rd)
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
}

func TestStorage_WriteReadMultiple(t *testing.T) {
	dir := testStorageDir(t)

	s, err := New(dir)
	if err != nil {
		t.Fatalf("new storage: %v", err)
	}
	defer s.Close()

	data := [][]byte{
		[]byte("first log"),
		[]byte("second log"),
		[]byte("third log"),
		[]byte("fourth log"),
	}

	records := make([]*domain.RecordData, 0, len(data))

	for _, d := range data {
		rd, err := s.Write(d)
		if err != nil {
			t.Fatalf("write: %v", err)
		}

		records = append(records, rd)
	}

	for i, rd := range records {
		actual, err := s.Read(rd)
		if err != nil {
			t.Fatalf("read %d: %v", i, err)
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

func TestStorage_RotationSegment(t *testing.T) {
	dir := testStorageDir(t)

	s, err := New(dir)
	if err != nil {
		t.Fatalf("new storage: %v", err)
	}
	defer s.Close()

	oldID := s.writeSegment.ID

	if err := s.rotationSegment(); err != nil {
		t.Fatalf("rotation: %v", err)
	}

	if s.writeSegment.ID != oldID+1 {
		t.Errorf(
			"expected segment ID %d, got %d",
			oldID+1,
			s.writeSegment.ID,
		)
	}

	if !s.writeSegment.IsOpen() {
		t.Error("new write segment should be open")
	}
}

func TestStorage_RotationClosesOldSegment(t *testing.T) {
	dir := testStorageDir(t)

	s, err := New(dir)
	if err != nil {
		t.Fatalf("new storage: %v", err)
	}
	defer s.Close()

	oldSegment := s.writeSegment

	if err := s.rotationSegment(); err != nil {
		t.Fatalf("rotation: %v", err)
	}

	if oldSegment.IsOpen() {
		t.Error("old segment should be closed")
	}

	if !s.writeSegment.IsOpen() {
		t.Error("new segment should be open")
	}
}

func TestStorage_Close(t *testing.T) {
	dir := testStorageDir(t)

	s, err := New(dir)
	if err != nil {
		t.Fatalf("new storage: %v", err)
	}

	if err := s.Close(); err != nil {
		t.Fatalf("close storage: %v", err)
	}

	if s.writeSegment.IsOpen() {
		t.Error("write segment should be closed")
	}

	if s.readSegment.IsOpen() {
		t.Error("read segment should be closed")
	}
}

func TestStorage_ReadOldRecordAfterRotation(t *testing.T) {
	dir := testStorageDir(t)

	s, err := New(dir)
	if err != nil {
		t.Fatalf("new storage: %v", err)
	}
	defer s.Close()

	expected := []byte("old log")

	rd, err := s.Write(expected)
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	oldSegmentID := rd.Pointer.SegmentID

	if err := s.rotationSegment(); err != nil {
		t.Fatalf("rotation: %v", err)
	}

	if s.writeSegment.ID == oldSegmentID {
		t.Fatal("rotation did not create new segment")
	}

	actual, err := s.Read(rd)
	if err != nil {
		t.Fatalf("read old record: %v", err)
	}

	if !bytes.Equal(actual, expected) {
		t.Errorf(
			"expected %q, got %q",
			expected,
			actual,
		)
	}
}
