package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
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

func TestParseGenerateStateArgsUsesConfig(t *testing.T) {
	configPath := writeTestConfig(t, `{
  "generate": {
    "state": {
      "rpcs": ["http://127.0.0.1:9545", "http://127.0.0.1:9546", "http://127.0.0.1:9547"],
      "minRpcs": 2,
      "block": 99,
      "account": "0x1111111111111111111111111111111111111111",
      "slot": "0x04",
      "out": "from-config.json"
    }
  }
}`)

	cfg, err := parseGenerateStateArgs([]string{"--config", configPath})
	if err != nil {
		t.Fatalf("parseGenerateStateArgs: %v", err)
	}
	if cfg.Request.BlockNumber != 99 {
		t.Fatalf("expected block number 99, got %d", cfg.Request.BlockNumber)
	}
	if cfg.Request.MinRPCSources != 2 {
		t.Fatalf("expected MinRPCSources=2, got %d", cfg.Request.MinRPCSources)
	}
	if got := strings.Join(cfg.Request.RPCURLs, ","); got != "http://127.0.0.1:9545,http://127.0.0.1:9546,http://127.0.0.1:9547" {
		t.Fatalf("unexpected rpc urls: %s", got)
	}
	if cfg.Out != "from-config.json" {
		t.Fatalf("unexpected output path: %s", cfg.Out)
	}
}

func TestParseGenerateStateArgsFlagsOverrideConfig(t *testing.T) {
	configPath := writeTestConfig(t, `{
  "generate": {
    "state": {
      "rpcs": ["http://127.0.0.1:9545", "http://127.0.0.1:9546", "http://127.0.0.1:9547"],
      "minRpcs": 3,
      "block": 99,
      "account": "0x1111111111111111111111111111111111111111",
      "slot": "0x04",
      "out": "from-config.json"
    }
  }
}`)

	cfg, err := parseGenerateStateArgs([]string{
		"--config", configPath,
		"--rpc", "http://127.0.0.1:8545",
		"--min-rpcs", "1",
		"--out", "override.json",
	})
	if err != nil {
		t.Fatalf("parseGenerateStateArgs: %v", err)
	}
	if got := strings.Join(cfg.Request.RPCURLs, ","); got != "http://127.0.0.1:8545" {
		t.Fatalf("expected rpc override, got %s", got)
	}
	if cfg.Request.MinRPCSources != 1 {
		t.Fatalf("expected MinRPCSources=1, got %d", cfg.Request.MinRPCSources)
	}
	if cfg.Out != "override.json" {
		t.Fatalf("unexpected output path: %s", cfg.Out)
	}
	if cfg.Request.BlockNumber != 99 {
		t.Fatalf("expected config block number 99, got %d", cfg.Request.BlockNumber)
	}
}

func TestParseVerifyReceiptArgsUsesConfigAndFlagOverrides(t *testing.T) {
	configPath := writeTestConfig(t, `{
  "verify": {
    "receipt": {
      "rpcs": ["http://127.0.0.1:9545", "http://127.0.0.1:9546", "http://127.0.0.1:9547"],
      "minRpcs": 2,
      "proof": "receipt-from-config.json",
      "expectEmitter": "0x2222222222222222222222222222222222222222",
      "expectTopics": ["0x01", "0x02"],
      "expectData": "0xaa"
    }
  }
}`)

	cfg, err := parseVerifyReceiptArgs([]string{
		"--config", configPath,
		"--expect-data", "0xbb",
	})
	if err != nil {
		t.Fatalf("parseVerifyReceiptArgs: %v", err)
	}
	if cfg.ProofPath != "receipt-from-config.json" {
		t.Fatalf("unexpected proof path: %s", cfg.ProofPath)
	}
	if cfg.VerifyRequest.MinRPCSources != 2 {
		t.Fatalf("expected MinRPCSources=2, got %d", cfg.VerifyRequest.MinRPCSources)
	}
	if got := strings.Join(cfg.VerifyRequest.RPCURLs, ","); got != "http://127.0.0.1:9545,http://127.0.0.1:9546,http://127.0.0.1:9547" {
		t.Fatalf("unexpected rpc urls: %s", got)
	}
	if cfg.Expectations == nil {
		t.Fatal("expected receipt expectations")
	}
	if cfg.Expectations.Emitter == nil || *cfg.Expectations.Emitter != common.HexToAddress("0x2222222222222222222222222222222222222222") {
		t.Fatalf("unexpected emitter: %+v", cfg.Expectations.Emitter)
	}
	if len(cfg.Expectations.Topics) != 2 {
		t.Fatalf("expected 2 topics, got %d", len(cfg.Expectations.Topics))
	}
	if got := common.Bytes2Hex(cfg.Expectations.Data); got != "bb" {
		t.Fatalf("expected expect-data override, got %s", got)
	}
}

func TestParseVerifyStateArgsRequiresIndependentRPCs(t *testing.T) {
	configPath := writeTestConfig(t, `{
  "generate": {
    "state": {
      "rpcs": ["http://127.0.0.1:9545", "http://127.0.0.1:9546", "http://127.0.0.1:9547"],
      "minRpcs": 3,
      "account": "0x1111111111111111111111111111111111111111",
      "slot": "0x04"
    }
  },
  "verify": {
    "state": {
      "proof": "state.json"
    }
  }
}`)

	_, err := parseVerifyStateArgs([]string{"--config", configPath})
	if err == nil {
		t.Fatal("expected verify state parsing to require independent rpc config")
	}
	if !strings.Contains(err.Error(), "verify state requires independent RPCs") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseVerifyTransactionArgsFlagRPCOverridesConfig(t *testing.T) {
	configPath := writeTestConfig(t, `{
  "verify": {
    "tx": {
      "rpcs": ["http://127.0.0.1:9545", "http://127.0.0.1:9546", "http://127.0.0.1:9547"],
      "minRpcs": 3,
      "proof": "from-config.json"
    }
  }
}`)

	cfg, err := parseVerifyTransactionArgs([]string{
		"--config", configPath,
		"--rpc", "http://127.0.0.1:8545",
		"--min-rpcs", "1",
		"--proof", "override.json",
	})
	if err != nil {
		t.Fatalf("parseVerifyTransactionArgs: %v", err)
	}
	if cfg.ProofPath != "override.json" {
		t.Fatalf("unexpected proof path: %s", cfg.ProofPath)
	}
	if got := strings.Join(cfg.VerifyRequest.RPCURLs, ","); got != "http://127.0.0.1:8545" {
		t.Fatalf("expected rpc override, got %s", got)
	}
	if cfg.VerifyRequest.MinRPCSources != 1 {
		t.Fatalf("expected MinRPCSources=1, got %d", cfg.VerifyRequest.MinRPCSources)
	}
}

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
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
