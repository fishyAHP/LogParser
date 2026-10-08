package storage

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

		if !s.writeSegment.isOpen() {
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

	if !s.writeSegment.isOpen() {
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

	if oldSegment.isOpen() {
		t.Error("old segment should be closed")
	}

	if !s.writeSegment.isOpen() {
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

	if s.writeSegment.isOpen() {
		t.Error("write segment should be closed")
	}

	if s.readSegment.isOpen() {
		t.Error("read segment should be closed")
	}
}

func TestStorageRecovery(t *testing.T) {
	dir := t.TempDir()

	storage, err := New(dir)
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}

	const initialRecords = 1000
	const additionalRecords = 100

	expected := make([][]byte, 0, initialRecords+additionalRecords)

	for i := 0; i < initialRecords; i++ {
		data := []byte(fmt.Sprintf("record-%d", i+1))

		id, err := storage.Write(data)
		if err != nil {
			t.Fatalf("write record %d: %v", i+1, err)
		}

		if id != domain.RecordID(i+1) {
			t.Fatalf("expected ID %d, got %d", i+1, id)
		}

		expected = append(expected, data)
	}

	if err := storage.Close(); err != nil {
		t.Fatalf("close storage: %v", err)
	}

	storage, err = New(dir)
	if err != nil {
		t.Fatalf("reopen storage: %v", err)
	}

	for i, want := range expected {
		got, err := storage.ReadByID(domain.RecordID(i + 1))
		if err != nil {
			t.Fatalf("read record %d: %v", i+1, err)
		}

		if !bytes.Equal(got, want) {
			t.Fatalf(
				"record %d: expected %q, got %q",
				i+1,
				want,
				got,
			)
		}
	}

	for i := 0; i < additionalRecords; i++ {
		id := initialRecords + i + 1
		data := []byte(fmt.Sprintf("record-%d", id))

		gotID, err := storage.Write(data)
		if err != nil {
			t.Fatalf("write record %d: %v", id, err)
		}

		if gotID != domain.RecordID(id) {
			t.Fatalf("expected ID %d, got %d", id, gotID)
		}

		expected = append(expected, data)
	}

	if err := storage.Close(); err != nil {
		t.Fatalf("close storage: %v", err)
	}

	storage, err = New(dir)
	if err != nil {
		t.Fatalf("second reopen: %v", err)
	}
	defer storage.Close()

	for i, want := range expected {
		got, err := storage.ReadByID(domain.RecordID(i + 1))
		if err != nil {
			t.Fatalf("read record %d: %v", i+1, err)
		}

		if !bytes.Equal(got, want) {
			t.Fatalf(
				"record %d: expected %q, got %q",
				i+1,
				want,
				got,
			)
		}
	}
}

func TestStorageCorruptedMagic(t *testing.T) {
	dir := t.TempDir()

	storage, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := storage.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}

	if err := storage.Close(); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "segment-0001")

	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := file.WriteAt([]byte("BROKEN"), 0); err != nil {
		file.Close()
		t.Fatal(err)
	}

	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	recovered, err := New(dir)
	if err == nil {
		recovered.Close()
		t.Fatal("expected recovery error for corrupted magic")
	}
}

func TestStorageCorruptedRecordID(t *testing.T) {
	dir := t.TempDir()

	storage, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := storage.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}

	if err := storage.Close(); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "segment-0001")

	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}

	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], 100)

	offset := int64(len(Magic) + 4)

	if _, err := file.WriteAt(buf[:], offset); err != nil {
		file.Close()
		t.Fatal(err)
	}

	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	recovered, err := New(dir)
	if err == nil {
		recovered.Close()
		t.Fatal("expected error for invalid RecordID sequence")
	}
}

func TestStorageTruncatedRecord(t *testing.T) {
	dir := t.TempDir()

	storage, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := storage.Write([]byte("hello world")); err != nil {
		t.Fatal(err)
	}

	if err := storage.Close(); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "segment-0001")

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Truncate(path, info.Size()-3); err != nil {
		t.Fatal(err)
	}

	recovered, err := New(dir)
	if err == nil {
		recovered.Close()
		t.Fatal("expected recovery error for truncated record")
	}
}

func TestStorageMissingSegment(t *testing.T) {
	dir := t.TempDir()

	for _, id := range []uint64{1, 3} {
		seg, err := newSegment(dir, id, writeFlag)
		if err != nil {
			t.Fatal(err)
		}

		if err := seg.close(); err != nil {
			t.Fatal(err)
		}
	}

	storage, err := New(dir)
	if err == nil {
		storage.Close()
		t.Fatal("expected error for missing segment-0002")
	}
}

func TestStorageRotationRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("large integration test")
	}

	dir := t.TempDir()

	storage, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	const count = 200
	data := bytes.Repeat([]byte("A"), 1024*1024)

	for i := 0; i < count; i++ {
		id, err := storage.Write(data)
		if err != nil {
			t.Fatalf("write %d: %v", i+1, err)
		}

		if id != domain.RecordID(i+1) {
			t.Fatalf("expected ID %d, got %d", i+1, id)
		}
	}

	if err := storage.Close(); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	segmentCount := 0
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "segment-") {
			segmentCount++
		}
	}

	if segmentCount < 2 {
		t.Fatalf("expected multiple segments, got %d", segmentCount)
	}

	storage, err = New(dir)
	if err != nil {
		t.Fatalf("reopen storage: %v", err)
	}
	defer storage.Close()

	for i := 1; i <= count; i++ {
		got, err := storage.ReadByID(domain.RecordID(i))
		if err != nil {
			t.Fatalf("read record %d: %v", i, err)
		}

		if !bytes.Equal(got, data) {
			t.Fatalf("record %d corrupted", i)
		}
	}
}

func TestStorageWriteAfterRecovery(t *testing.T) {
	dir := t.TempDir()

	type testRecord struct {
		id   domain.RecordID
		data []byte
	}

	const recordsCount = 200

	records := make([]testRecord, 0, recordsCount)
	usedIDs := make(map[domain.RecordID]struct{})

	openStorage := func() *Storage {
		t.Helper()

		s, err := New(dir)
		if err != nil {
			t.Fatalf("open storage: %v", err)
		}

		return s
	}

	writeRecords := func(s *Storage, from, to int) {
		t.Helper()

		for i := from; i < to; i++ {
			data := []byte(fmt.Sprintf("test record %d", i))

			id, err := s.Write(data)
			if err != nil {
				t.Fatalf("write record %d: %v", i, err)
			}

			if _, exists := usedIDs[id]; exists {
				t.Fatalf("duplicate RecordID: %d", id)
			}

			usedIDs[id] = struct{}{}

			records = append(records, testRecord{
				id:   id,
				data: data,
			})
		}
	}

	checkRecords := func(s *Storage) {
		t.Helper()

		for _, expected := range records {
			actual, err := s.ReadByID(expected.id)
			if err != nil {
				t.Fatalf(
					"read record %d: %v",
					expected.id,
					err,
				)
			}

			if !bytes.Equal(actual, expected.data) {
				t.Fatalf(
					"record %d: expected %q, got %q",
					expected.id,
					expected.data,
					actual,
				)
			}
		}
	}

	closeStorage := func(s *Storage) {
		t.Helper()

		if err := s.Close(); err != nil {
			t.Fatalf("close storage: %v", err)
		}
	}

	// Первый запуск: записываем 100 записей.
	s := openStorage()
	writeRecords(s, 0, 100)
	closeStorage(s)

	// Первый перезапуск: проверяем старые записи
	// и добавляем ещё 100.
	s = openStorage()
	checkRecords(s)
	writeRecords(s, 100, 200)
	checkRecords(s)
	closeStorage(s)

	// Второй перезапуск: проверяем все 200 записей.
	s = openStorage()
	defer closeStorage(s)

	checkRecords(s)

	if len(usedIDs) != recordsCount {
		t.Fatalf(
			"expected %d unique RecordIDs, got %d",
			recordsCount,
			len(usedIDs),
		)
	}
}
