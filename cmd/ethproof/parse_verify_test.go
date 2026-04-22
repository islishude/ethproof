package main

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

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
      "rpcs": ["http://127.0.0.1:9545", "http://127.0.0.1:9546", "http://127.0.0.1:9547"],
      "minRpcs": 2,
      "proof": "receipt-from-config.json",
      "expectEmitter": "0x2222222222222222222222222222222222222222",
      "expectTopics": ["0x01", "0x02"],
      "expectData": "0xaa"
    }
  }
}`)
				cfg, err := parseVerifyReceiptArgs([]string{"--config", configPath, "--expect-data", "0xbb"})
				if err != nil {
					t.Fatalf("parseVerifyReceiptArgs: %v", err)
				}
				if cfg.ProofPath != "receipt-from-config.json" || cfg.VerifyRequest.MinRPCSources != 2 {
					t.Fatalf("unexpected verify receipt config: %+v", cfg)
				}
				if got := strings.Join(cfg.VerifyRequest.RPCURLs, ","); got != "http://127.0.0.1:9545,http://127.0.0.1:9546,http://127.0.0.1:9547" {
					t.Fatalf("unexpected rpc urls: %s", got)
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
			name: "state requires independent rpcs",
			run: func(t *testing.T) {
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
				if err == nil || !strings.Contains(err.Error(), "verify state requires independent RPCs") {
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
				if cfg.ProofPath != "override.json" || cfg.VerifyRequest.MinRPCSources != 1 {
					t.Fatalf("unexpected verify tx config: %+v", cfg)
				}
				if got := strings.Join(cfg.VerifyRequest.RPCURLs, ","); got != "http://127.0.0.1:8545" {
					t.Fatalf("unexpected rpc override: %s", got)
				}
			},
		},
		{
			name: "uses logging config and flag override",
			run: func(t *testing.T) {
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
				cfg, err := parseVerifyTransactionArgs([]string{"--config", configPath, "--log-format", "json"})
				if err != nil {
					t.Fatalf("parseVerifyTransactionArgs: %v", err)
				}
				if cfg.Logging.Level != "error" || cfg.Logging.Format != "json" {
					t.Fatalf("unexpected logging config: %+v", cfg.Logging)
				}
			},
		},
		{
			name: "rejects invalid log format",
			run: func(t *testing.T) {
				_, err := parseVerifyTransactionArgs([]string{
					"--rpc", "http://127.0.0.1:8545",
					"--min-rpcs", "1",
					"--proof", "tx.json",
					"--log-format", "yaml",
				})
				if err == nil || !strings.Contains(err.Error(), "unsupported log format") {
					t.Fatalf("unexpected error: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}
