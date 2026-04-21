package main

import (
	"strings"
	"testing"
)

func TestRenderErrorUsageWritesToStderr(t *testing.T) {
	stdout, stderr := captureCommandOutput(t, func() {
		renderError(newUsageError("bad args"))
	})
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if stderr == "" {
		t.Fatal("expected usage output on stderr")
	}
}

func TestExitCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "help", err: newHelpError(), want: 0},
		{name: "usage", err: newUsageError("bad"), want: 2},
		{name: "runtime", err: errString("boom"), want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := exitCode(tt.err); got != tt.want {
				t.Fatalf("exitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestRunMainRuntimeErrorWritesErrorLogToStderr(t *testing.T) {
	var exit int
	stdout, stderr := captureCommandOutput(t, func() {
		exit = runMain([]string{
			"verify", "tx",
			"--rpc", "http://127.0.0.1:8545",
			"--min-rpcs", "1",
			"--proof", "missing.json",
		})
	})

	if exit != 1 {
		t.Fatalf("expected exit code 1, got %d", exit)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "level=ERROR") {
		t.Fatalf("expected structured error log on stderr, got %q", stderr)
	}
	if !strings.Contains(stderr, "read transaction proof json") {
		t.Fatalf("expected runtime error message in stderr, got %q", stderr)
	}
}

type errString string

func (e errString) Error() string {
	return string(e)
}
