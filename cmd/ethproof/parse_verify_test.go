package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestVerifyStateAndTransactionConfigIsolation(t *testing.T) {
	parsers := []struct {
		kind  string
		parse func([]string) (verifyProofConfig, error)
	}{
		{"state", func(args []string) (verifyProofConfig, error) {
			cfg, err := parseVerifyStateArgs(args)
			return verifyProofConfig(cfg), err
		}},
		{"tx", func(args []string) (verifyProofConfig, error) {
			cfg, err := parseVerifyTransactionArgs(args)
			return verifyProofConfig(cfg), err
		}},
	}
	configPath := writeTestConfig(t, `{
  "verify": {
    "state": {"rpc": "http://state", "proof": "configured-state.json"},
    "tx": {"rpc": "http://tx", "proof": "configured-tx.json"}
  }
}`)
	for _, parser := range parsers {
		t.Run(parser.kind, func(t *testing.T) {
			cfg, err := parser.parse([]string{"--config", configPath})
			if err != nil {
				t.Fatal(err)
			}
			if cfg.ProofPath != "configured-"+parser.kind+".json" || cfg.RPCURL != "http://"+parser.kind {
				t.Fatalf("wrong config section: %+v", cfg)
			}
			cfg, err = parser.parse([]string{"--rpc", "http://override"})
			if err != nil || cfg.ProofPath != parser.kind+".json" {
				t.Fatalf("wrong default proof path: %+v, %v", cfg, err)
			}
			other := "state"
			if parser.kind == "state" {
				other = "tx"
			}
			otherConfig := writeTestConfig(t, fmt.Sprintf(`{"verify": {%q: {"rpc": "http://other"}}}`, other))
			_, err = parser.parse([]string{"--config", otherConfig})
			if err == nil || !strings.Contains(err.Error(), "verify "+parser.kind+" requires a separately configured RPC") {
				t.Fatalf("must not reuse the other section's RPC endpoint: %v", err)
			}
		})
	}
}

func TestParseVerifyArgsScenarios(t *testing.T) {
	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "receipt config merge and flag override",
			run: func(t *testing.T) {
				configPath := writeTestConfig(t, `{
  "verify": {
    "receipt": {
      "rpc": "http://127.0.0.1:9545",
      "proof": "receipt-from-config.json",
      "expectEmitter": "0x2222222222222222222222222222222222222222",
      "expectTopics": [
        "0x0000000000000000000000000000000000000000000000000000000000000001",
        "0x0000000000000000000000000000000000000000000000000000000000000002"
      ],
      "expectData": "0xaa"
    }
  }
}`)
				cfg, err := parseVerifyReceiptArgs([]string{"--config", configPath, "--expect-data", "0xbb"})
				if err != nil {
					t.Fatalf("parseVerifyReceiptArgs: %v", err)
				}
				if cfg.ProofPath != "receipt-from-config.json" {
					t.Fatalf("unexpected verify receipt config: %+v", cfg)
				}
				if got := cfg.RPCURL; got != "http://127.0.0.1:9545" {
					t.Fatalf("unexpected rpc URL: %s", got)
				}
				if cfg.Expectations == nil || cfg.Expectations.Emitter == nil || *cfg.Expectations.Emitter != common.HexToAddress("0x2222222222222222222222222222222222222222") {
					t.Fatalf("unexpected expectations: %+v", cfg.Expectations)
				}
				if got := common.Bytes2Hex(cfg.Expectations.Data); got != "bb" {
					t.Fatalf("expected expect-data override, got %s", got)
				}
			},
		},
		{
			name: "state requires separately configured rpc",
			run: func(t *testing.T) {
				configPath := writeTestConfig(t, `{
  "generate": {
    "state": {
      "rpc": "http://127.0.0.1:9545",
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
				if err == nil || !strings.Contains(err.Error(), "verify state requires a separately configured RPC") {
					t.Fatalf("unexpected error: %v", err)
				}
			},
		},
		{
			name: "transaction flags override config",
			run: func(t *testing.T) {
				configPath := writeTestConfig(t, `{
  "verify": {
    "tx": {
      "rpc": "http://127.0.0.1:9545",
      "proof": "from-config.json"
    }
  }
}`)
				cfg, err := parseVerifyTransactionArgs([]string{
					"--config", configPath,
					"--rpc", "http://127.0.0.1:8545",
					"--proof", "override.json",
				})
				if err != nil {
					t.Fatalf("parseVerifyTransactionArgs: %v", err)
				}
				if cfg.ProofPath != "override.json" {
					t.Fatalf("unexpected verify tx config: %+v", cfg)
				}
				if got := cfg.RPCURL; got != "http://127.0.0.1:8545" {
					t.Fatalf("unexpected rpc override: %s", got)
				}
			},
		},
		{
			name: "rejects removed logging config",
			run: func(t *testing.T) {
				configPath := writeTestConfig(t, `{
  "logging": {
    "level": "error",
    "format": "text"
  },
  "verify": {
    "tx": {
      "rpc": "http://127.0.0.1:9545",
      "proof": "from-config.json"
    }
  }
}`)
				_, err := parseVerifyTransactionArgs([]string{"--config", configPath})
				if err == nil || !strings.Contains(err.Error(), "unknown field \"logging\"") {
					t.Fatalf("unexpected error: %v", err)
				}
			},
		},
		{
			name: "rejects removed log format flag",
			run: func(t *testing.T) {
				_, err := parseVerifyTransactionArgs([]string{
					"--rpc", "http://127.0.0.1:8545",
					"--proof", "tx.json",
					"--log-format", "yaml",
				})
				if err == nil || !strings.Contains(err.Error(), "flag provided but not defined: -log-format") {
					t.Fatalf("unexpected error: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}
