package system

import (
	"os"
	"testing"
)

func TestParseMeminfo(t *testing.T) {
	infoMap := ParseMeminfo()
	if size, exists := infoMap[MemMainTotalKey]; !exists || size <= 0 {
		t.Fatal(size, MemMainTotalKey)
	}
	if _, exists := infoMap[MemSwapTotalKey]; !exists {
		t.Fatal(MemSwapTotalKey)
	}
}

func TestGetMemSize(t *testing.T) {
	if size := GetMainMemSizeMB(); size < 1 {
		t.Fatal(size)
	}
	if size := GetTotalMemSizeMB(); size < 1 {
		t.Fatal(size)
	}
}

func TestGetTotalMemSizePages(t *testing.T) {
	if pages := GetTotalMemSizePages(); pages != GetTotalMemSizeMB()*1024/uint64(os.Getpagesize()) {
		t.Fatal(pages)
	}
}

func TestGetSemaphoreLimits(t *testing.T) {
	msl, mns, opm, mni := GetSemaphoreLimits()
	if msl < 3 || mns < 3 || opm < 3 || mni < 3 {
		t.Fatal(msl, mns, opm, mni)
	}
}

func TestGetMeminfoEntry(t *testing.T) {
	// Test reading an existing key from real /proc/meminfo
	memTotal, err := GetMeminfoEntry(MemMainTotalKey)
	if err != nil || memTotal == 0 {
		t.Fatalf("unexpected error reading %s: %v, val: %d", MemMainTotalKey, err, memTotal)
	}

	// Test non-existent key
	_, err = GetMeminfoEntry("NonExistentKey123")
	if err == nil {
		t.Fatalf("expected error for non-existent key, got nil")
	}

	// Test with mock file
	origProc := ProcMeminfo
	defer func() { ProcMeminfo = origProc }()

	tmpFile, err := os.CreateTemp("", "meminfo_test")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	content := `MemTotal:       2147483648 kB
MemFree:         1048576 kB
HugePages_Total:       4
InvalidLineWithoutColon
InvalidNumber: abc kB
`
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	ProcMeminfo = tmpFile.Name()

	val, err := GetMeminfoEntry("MemTotal")
	if err != nil || val != 2147483648 {
		t.Fatalf("expected 2147483648 for MemTotal, got %d, err: %v", val, err)
	}

	val, err = GetMeminfoEntry("HugePages_Total")
	if err != nil || val != 4 {
		t.Fatalf("expected 4 for HugePages_Total, got %d, err: %v", val, err)
	}

	_, err = GetMeminfoEntry("InvalidNumber")
	if err == nil {
		t.Fatalf("expected error for InvalidNumber, got nil")
	}

	// Test file does not exist
	ProcMeminfo = "/path/to/nonexistent/meminfo"
	_, err = GetMeminfoEntry("MemTotal")
	if err == nil {
		t.Fatalf("expected error for non-existent file, got nil")
	}
}
