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
		{name: "subcommand help", args: []string{"generate", "state", "--help"}},
		{name: "resolve help", args: []string{"resolve", "-h"}},
		{name: "resolve subcommand help", args: []string{"resolve", "slot", "--help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var exit int
			stdout, stderr := captureCommandOutput(t, func() {
				exit = runMain(tt.args)
			})

			if exit != 0 {
				t.Fatalf("expected exit code 0, got %d", exit)
			}
			if stdout != usageText {
				t.Fatalf("unexpected stdout:\n%s", stdout)
			}
			if stderr != "" {
				t.Fatalf("expected empty stderr, got %q", stderr)
			}
		})
	}
}
