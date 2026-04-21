package main

import (
	"io"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestRunMainHelpPrintsUsageToStdout(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "root short help", args: []string{"-h"}},
		{name: "root long help", args: []string{"--help"}},
		{name: "generate help", args: []string{"generate", "-h"}},
		{name: "subcommand help", args: []string{"generate", "state", "--help"}},
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

func TestParseGenerateStateArgsPassesMinRPCs(t *testing.T) {
	cfg, err := parseGenerateStateArgs([]string{
		"--rpc", "http://127.0.0.1:8545",
		"--min-rpcs", "1",
		"--block", "12",
		"--account", "0x1111111111111111111111111111111111111111",
		"--slot", "0x01",
		"--out", "state.json",
	})
	if err != nil {
		t.Fatalf("parseGenerateStateArgs: %v", err)
	}
	if cfg.Request.MinRPCSources != 1 {
		t.Fatalf("expected MinRPCSources=1, got %d", cfg.Request.MinRPCSources)
	}
	if cfg.Request.BlockNumber != 12 {
		t.Fatalf("expected block number 12, got %d", cfg.Request.BlockNumber)
	}
	if cfg.Request.Account != common.HexToAddress("0x1111111111111111111111111111111111111111") {
		t.Fatalf("unexpected account: %s", cfg.Request.Account)
	}
	if cfg.Request.Slot != common.HexToHash("0x01") {
		t.Fatalf("unexpected slot: %s", cfg.Request.Slot)
	}
	if cfg.Out != "state.json" {
		t.Fatalf("unexpected output path: %s", cfg.Out)
	}
}

func TestParseGenerateReceiptArgsDefaultsMinRPCs(t *testing.T) {
	cfg, err := parseGenerateReceiptArgs([]string{
		"--rpc", "http://127.0.0.1:8545",
		"--rpc", "http://127.0.0.1:8546",
		"--rpc", "http://127.0.0.1:8547",
		"--tx", "0x02",
		"--log-index", "3",
	})
	if err != nil {
		t.Fatalf("parseGenerateReceiptArgs: %v", err)
	}
	if cfg.Request.MinRPCSources != proofMinRPCsDefault() {
		t.Fatalf("expected default MinRPCSources=%d, got %d", proofMinRPCsDefault(), cfg.Request.MinRPCSources)
	}
	if cfg.Request.LogIndex != 3 {
		t.Fatalf("expected log index 3, got %d", cfg.Request.LogIndex)
	}
}

func TestParseGenerateTransactionArgsPassesMinRPCs(t *testing.T) {
	cfg, err := parseGenerateTransactionArgs([]string{
		"--rpc", "http://127.0.0.1:8545",
		"--min-rpcs", "1",
		"--tx", "0x03",
		"--out", "tx.json",
	})
	if err != nil {
		t.Fatalf("parseGenerateTransactionArgs: %v", err)
	}
	if cfg.Request.MinRPCSources != 1 {
		t.Fatalf("expected MinRPCSources=1, got %d", cfg.Request.MinRPCSources)
	}
	if cfg.Request.TxHash != common.HexToHash("0x03") {
		t.Fatalf("unexpected tx hash: %s", cfg.Request.TxHash)
	}
	if cfg.Out != "tx.json" {
		t.Fatalf("unexpected output path: %s", cfg.Out)
	}
}

func captureCommandOutput(t *testing.T, fn func()) (string, string) {
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
