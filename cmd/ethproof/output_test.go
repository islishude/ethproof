package main

import "testing"

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

type errString string

func (e errString) Error() string {
	return string(e)
}
