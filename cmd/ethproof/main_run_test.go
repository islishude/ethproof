package main

import "testing"

func TestRunMainHelpPrintsUsageToStdout(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "root short help", args: []string{"-h"}},
		{name: "root long help", args: []string{"--help"}},
		{name: "generate help", args: []string{"generate", "-h"}},
		{name: "generate subcommand help", args: []string{"generate", "state", "--help"}},
		{name: "resolve help", args: []string{"resolve", "-h"}},
		{name: "resolve subcommand help", args: []string{"resolve", "slot", "--help"}},
		{name: "verify help", args: []string{"verify", "-h"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var exit int
			stdout, stderr := captureCommandOutput(t, func() {
				exit = runMain(tt.args)
			})

			if exit != 0 || stdout != usageText || stderr != "" {
				t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
			}
		})
	}
}

func TestRunMainUsageErrorsRenderToStderr(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "missing command", args: nil},
		{name: "unknown command", args: []string{"unknown"}},
		{name: "missing generate subcommand", args: []string{"generate"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var exit int
			stdout, stderr := captureCommandOutput(t, func() {
				exit = runMain(tt.args)
			})
			if exit != 2 || stdout != "" || stderr == "" {
				t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
			}
		})
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
