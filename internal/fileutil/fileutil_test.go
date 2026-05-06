package fileutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWriteFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fileutil-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	path := filepath.Join(tmpDir, "testfile")
	data := []byte("hello world")
	perm := os.FileMode(0644)

	if err := AtomicWriteFile(path, data, perm); err != nil {
		t.Fatalf("AtomicWriteFile() error = %v", err)
	}

	// Verify content
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != string(data) {
		t.Errorf("content mismatch: got %q, want %q", string(content), string(data))
	}

	// Verify permissions
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	// On some systems/filesystems, the actual mode might not perfectly match the requested mode
	// (e.g., due to umask), but we check the lower 9 bits.
	if info.Mode().Perm() != perm {
		t.Errorf("permission mismatch: got %v, want %v", info.Mode().Perm(), perm)
	}

	// Test overwriting
	newData := []byte("new data")
	if err := AtomicWriteFile(path, newData, perm); err != nil {
		t.Fatalf("AtomicWriteFile() error during overwrite = %v", err)
	}

	content, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != string(newData) {
		t.Errorf("content mismatch after overwrite: got %q, want %q", string(content), string(newData))
	}
}
