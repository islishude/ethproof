package main

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/islishude/ethproof/proof"
)

func TestParseGenerateRequests(t *testing.T) {
	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "state direct flags",
			run: func(t *testing.T) {
				cfg, err := parseGenerateStateArgs([]string{
					"--rpc", "http://127.0.0.1:8545",
					"--min-rpcs", "1",
					"--block", "12",
					"--account", "0x1111111111111111111111111111111111111111",
					"--slot", "0x01",
					"--slot", "0x02",
					"--out", "state.json",
				})
				if err != nil {
					t.Fatalf("parseGenerateStateArgs: %v", err)
				}
				if cfg.Request.MinRPCSources != 1 || cfg.Request.BlockNumber != 12 {
					t.Fatalf("unexpected state request: %+v", cfg.Request)
				}
				if cfg.Request.Account != common.HexToAddress("0x1111111111111111111111111111111111111111") {
					t.Fatalf("unexpected account: %s", cfg.Request.Account)
				}
				if len(cfg.Request.Slots) != 2 || cfg.Request.Slots[0] != common.HexToHash("0x01") || cfg.Request.Slots[1] != common.HexToHash("0x02") {
					t.Fatalf("unexpected slots: %v", cfg.Request.Slots)
				}
				if cfg.Out != "state.json" {
					t.Fatalf("unexpected output path: %s", cfg.Out)
				}
			},
		},
		{
			name: "receipt default min rpcs",
			run: func(t *testing.T) {
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
				if cfg.Request.MinRPCSources != proof.DefaultMinRPCSources {
					t.Fatalf("expected default MinRPCSources=%d, got %d", proof.DefaultMinRPCSources, cfg.Request.MinRPCSources)
				}
				if cfg.Request.LogIndex != 3 {
					t.Fatalf("unexpected log index: %d", cfg.Request.LogIndex)
				}
			},
		},
		{
			name: "transaction direct flags",
			run: func(t *testing.T) {
				cfg, err := parseGenerateTransactionArgs([]string{
					"--rpc", "http://127.0.0.1:8545",
					"--min-rpcs", "1",
					"--tx", "0x03",
					"--out", "tx.json",
				})
				if err != nil {
					t.Fatalf("parseGenerateTransactionArgs: %v", err)
				}
				if cfg.Request.MinRPCSources != 1 || cfg.Request.TxHash != common.HexToHash("0x03") {
					t.Fatalf("unexpected tx request: %+v", cfg.Request)
				}
				if cfg.Out != "tx.json" {
					t.Fatalf("unexpected output path: %s", cfg.Out)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}

func TestParseGenerateStateArgsScenarios(t *testing.T) {
	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "uses config",
			run: func(t *testing.T) {
				configPath := writeTestConfig(t, `{
  "generate": {
    "state": {
      "rpcs": ["http://127.0.0.1:9545", "http://127.0.0.1:9546", "http://127.0.0.1:9547"],
      "minRpcs": 2,
      "block": 99,
      "account": "0x1111111111111111111111111111111111111111",
      "slots": ["0x04", "0x05"],
      "out": "from-config.json"
    }
  }
}`)
				cfg, err := parseGenerateStateArgs([]string{"--config", configPath})
				if err != nil {
					t.Fatalf("parseGenerateStateArgs: %v", err)
				}
				if got := strings.Join(cfg.Request.RPCURLs, ","); got != "http://127.0.0.1:9545,http://127.0.0.1:9546,http://127.0.0.1:9547" {
					t.Fatalf("unexpected rpc urls: %s", got)
				}
				if cfg.Request.MinRPCSources != 2 || cfg.Request.BlockNumber != 99 || cfg.Out != "from-config.json" {
					t.Fatalf("unexpected config merge result: %+v", cfg)
				}
			},
		},
		{
			name: "flags override config",
			run: func(t *testing.T) {
				configPath := writeTestConfig(t, `{
  "generate": {
    "state": {
      "rpcs": ["http://127.0.0.1:9545", "http://127.0.0.1:9546", "http://127.0.0.1:9547"],
      "minRpcs": 3,
      "block": 99,
      "account": "0x1111111111111111111111111111111111111111",
      "slots": ["0x04"],
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
					t.Fatalf("unexpected rpc override: %s", got)
				}
				if cfg.Request.MinRPCSources != 1 || cfg.Out != "override.json" || cfg.Request.BlockNumber != 99 {
					t.Fatalf("unexpected override result: %+v", cfg)
				}
			},
		},
		{
			name: "rejects removed logging config",
			run: func(t *testing.T) {
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
      "slots": ["0x04"]
    }
  }
}`)
				_, err := parseGenerateStateArgs([]string{"--config", configPath})
				if err == nil || !strings.Contains(err.Error(), "unknown field \"logging\"") {
					t.Fatalf("unexpected error: %v", err)
				}
			},
		},
		{
			name: "rejects removed log level flag",
			run: func(t *testing.T) {
				_, err := parseGenerateStateArgs([]string{
					"--rpc", "http://127.0.0.1:8545",
					"--min-rpcs", "1",
					"--block", "12",
					"--account", "0x1111111111111111111111111111111111111111",
					"--slot", "0x01",
					"--log-level", "verbose",
				})
				if err == nil || !strings.Contains(err.Error(), "flag provided but not defined: -log-level") {
					t.Fatalf("unexpected error: %v", err)
				}
			},
		},
		{
			name: "rejects legacy slot config field",
			run: func(t *testing.T) {
				configPath := writeTestConfig(t, `{
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
				_, err := parseGenerateStateArgs([]string{"--config", configPath})
				if err == nil || !strings.Contains(err.Error(), "unknown field \"slot\"") {
					t.Fatalf("unexpected error: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}
