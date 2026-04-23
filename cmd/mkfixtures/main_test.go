package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunMainPrintsRuntimeErrors(t *testing.T) {
	parent := t.TempDir()
	filePath := filepath.Join(parent, "not-a-dir")
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	var exit int
	stdout, stderr := captureOutput(t, func() {
		exit = runMain([]string{"--out-dir", filepath.Join(filePath, "child")})
	})

	if exit != 1 {
		t.Fatalf("expected exit code 1, got %d", exit)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "error: create output dir") {
		t.Fatalf("expected readable runtime error on stderr, got %q", stderr)
	}
}

func TestRunMainPrintsReadableSuccessStatus(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "fixtures")

	var exit int
	stdout, stderr := captureOutput(t, func() {
		exit = runMain([]string{"--out-dir", outDir})
	})

	if exit != 0 {
		t.Fatalf("expected exit code 0, got %d", exit)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "wrote offline fixtures to "+outDir) {
		t.Fatalf("expected readable success status on stderr, got %q", stderr)
	}
}

func captureOutput(t *testing.T, fn func()) (string, string) {
	t.Helper()

	originalStdout := os.Stdout
	originalStderr := os.Stderr

	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}

	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter

	fn()

	if err := stdoutWriter.Close(); err != nil {
		t.Fatalf("close stdout writer: %v", err)
	}
	if err := stderrWriter.Close(); err != nil {
		t.Fatalf("close stderr writer: %v", err)
	}

	os.Stdout = originalStdout
	os.Stderr = originalStderr

	stdout, err := io.ReadAll(stdoutReader)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	stderr, err := io.ReadAll(stderrReader)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}

	return string(stdout), string(stderr)
}
