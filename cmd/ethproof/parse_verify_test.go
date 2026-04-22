package main

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

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
      "slots": ["0x04"]
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

func TestParseVerifyTransactionArgsUsesLoggingConfigAndFlagOverrides(t *testing.T) {
	configPath := writeTestConfig(t, `{
  "logging": {
    "level": "error",
    "format": "text"
  },
  "verify": {
    "tx": {
      "rpcs": ["http://127.0.0.1:9545"],
      "minRpcs": 1,
      "proof": "from-config.json"
    }
  }
}`)

	cfg, err := parseVerifyTransactionArgs([]string{
		"--config", configPath,
		"--log-format", "json",
	})
	if err != nil {
		t.Fatalf("parseVerifyTransactionArgs: %v", err)
	}
	if cfg.Logging.Level != "error" {
		t.Fatalf("expected config log level error, got %s", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "json" {
		t.Fatalf("expected log format override json, got %s", cfg.Logging.Format)
	}
}

func TestParseVerifyTransactionArgsRejectsInvalidLogFormat(t *testing.T) {
	_, err := parseVerifyTransactionArgs([]string{
		"--rpc", "http://127.0.0.1:8545",
		"--min-rpcs", "1",
		"--proof", "tx.json",
		"--log-format", "yaml",
	})
	if err == nil {
		t.Fatal("expected invalid log format to fail")
	}
	if !strings.Contains(err.Error(), "unsupported log format") {
		t.Fatalf("unexpected error: %v", err)
	}
}
