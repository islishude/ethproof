package main

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/islishude/ethproof/internal/logutil"
)

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

func TestParseGenerateStateArgsUsesLoggingConfigAndFlagOverrides(t *testing.T) {
	configPath := writeTestConfig(t, `{
  "logging": {
    "level": "warn",
    "format": "json"
  },
  "generate": {
    "state": {
      "rpcs": ["http://127.0.0.1:9545"],
      "minRpcs": 1,
      "block": 99,
      "account": "0x1111111111111111111111111111111111111111",
      "slot": "0x04"
    }
  }
}`)

	cfg, err := parseGenerateStateArgs([]string{
		"--config", configPath,
		"--log-level", "debug",
	})
	if err != nil {
		t.Fatalf("parseGenerateStateArgs: %v", err)
	}
	if cfg.Logging.Level != "debug" {
		t.Fatalf("expected log level override debug, got %s", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "json" {
		t.Fatalf("expected config log format json, got %s", cfg.Logging.Format)
	}
}

func TestParseGenerateStateArgsRejectsInvalidLogLevel(t *testing.T) {
	_, err := parseGenerateStateArgs([]string{
		"--rpc", "http://127.0.0.1:8545",
		"--min-rpcs", "1",
		"--block", "12",
		"--account", "0x1111111111111111111111111111111111111111",
		"--slot", "0x01",
		"--log-level", "verbose",
	})
	if err == nil {
		t.Fatal("expected invalid log level to fail")
	}
	if !strings.Contains(err.Error(), "unsupported log level") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseGenerateStateArgsDefaultsLogging(t *testing.T) {
	cfg, err := parseGenerateStateArgs([]string{
		"--rpc", "http://127.0.0.1:8545",
		"--min-rpcs", "1",
		"--block", "12",
		"--account", "0x1111111111111111111111111111111111111111",
		"--slot", "0x01",
	})
	if err != nil {
		t.Fatalf("parseGenerateStateArgs: %v", err)
	}
	if cfg.Logging != logutil.DefaultConfig() {
		t.Fatalf("unexpected default logging config: %+v", cfg.Logging)
	}
}
