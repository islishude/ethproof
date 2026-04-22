package main

import (
	"strings"
	"testing"

	"github.com/islishude/ethproof/proof"
)

func TestParseResolveSlotArgsScenarios(t *testing.T) {
	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "parses explicit format and output",
			run: func(t *testing.T) {
				cfg, err := parseResolveSlotArgs([]string{
					"--compiler-output", "out/Fixture.json",
					"--contract", "contracts/Fixture.sol:Fixture",
					"--var", "data[4][9].b",
					"--format", "build-info",
					"--out", "slot.json",
				})
				if err != nil {
					t.Fatalf("parseResolveSlotArgs: %v", err)
				}
				if cfg.CompilerOutput != "out/Fixture.json" || cfg.Contract != "contracts/Fixture.sol:Fixture" || cfg.Variable != "data[4][9].b" {
					t.Fatalf("unexpected resolve config: %+v", cfg)
				}
				if cfg.Format != proof.StorageLayoutFormatBuildInfo || cfg.Out != "slot.json" {
					t.Fatalf("unexpected resolve format/output: %+v", cfg)
				}
			},
		},
		{
			name: "defaults to auto format",
			run: func(t *testing.T) {
				cfg, err := parseResolveSlotArgs([]string{
					"--compiler-output", "out/Fixture.json",
					"--contract", "Fixture",
					"--var", "value",
				})
				if err != nil {
					t.Fatalf("parseResolveSlotArgs: %v", err)
				}
				if cfg.Format != proof.StorageLayoutFormatAuto {
					t.Fatalf("expected auto format, got %s", cfg.Format)
				}
			},
		},
		{
			name: "rejects missing required flags",
			run: func(t *testing.T) {
				tests := []struct {
					args []string
					want string
				}{
					{args: []string{"--contract", "Fixture", "--var", "value"}, want: "requires --compiler-output"},
					{args: []string{"--compiler-output", "out/Fixture.json", "--var", "value"}, want: "requires --contract"},
					{args: []string{"--compiler-output", "out/Fixture.json", "--contract", "Fixture"}, want: "requires --var"},
				}
				for _, tt := range tests {
					_, err := parseResolveSlotArgs(tt.args)
					if err == nil || !strings.Contains(err.Error(), tt.want) {
						t.Fatalf("unexpected error: %v", err)
					}
				}
			},
		},
		{
			name: "rejects invalid format",
			run: func(t *testing.T) {
				_, err := parseResolveSlotArgs([]string{
					"--compiler-output", "out/Fixture.json",
					"--contract", "Fixture",
					"--var", "value",
					"--format", "weird",
				})
				if err == nil || !strings.Contains(err.Error(), "unsupported storage layout format") {
					t.Fatalf("unexpected error: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}
