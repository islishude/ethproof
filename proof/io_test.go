package proof

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSaveJSONWritesPrivateFileAtomically(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "proof.json")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatalf("seed destination: %v", err)
	}
	if err := SaveJSON(path, map[string]int{"value": 1}); err != nil {
		t.Fatalf("SaveJSON: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat output: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("unexpected permissions: %o", info.Mode().Perm())
	}
	var decoded map[string]int
	if err := LoadJSON(path, &decoded); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	if decoded["value"] != 1 {
		t.Fatalf("unexpected json: %v", decoded)
	}
	assertNoJSONTemps(t, dir)
}

func TestSaveJSONFailurePreservesDestinationAndCleansTemp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "proof.json")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatalf("seed destination: %v", err)
	}
	if err := SaveJSON(path, make(chan int)); err == nil {
		t.Fatal("expected marshal failure")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}
	if string(got) != "old" {
		t.Fatalf("destination changed after failure: %q", got)
	}
	assertNoJSONTemps(t, dir)

	directoryTarget := filepath.Join(dir, "directory-target")
	if err := os.Mkdir(directoryTarget, 0o700); err != nil {
		t.Fatalf("create directory target: %v", err)
	}
	if err := SaveJSON(directoryTarget, map[string]int{"value": 1}); err == nil {
		t.Fatal("expected rename failure for directory target")
	}
	assertNoJSONTemps(t, dir)
}

func assertNoJSONTemps(t *testing.T, dir string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, ".*.tmp-*"))
	if err != nil {
		t.Fatalf("glob temp files: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files were not cleaned up: %v", matches)
	}
}
