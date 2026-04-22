package main

import (
	"strings"
	"testing"

	"github.com/islishude/ethproof/proof"
)

func TestParseResolveSlotArgs(t *testing.T) {
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
	if cfg.CompilerOutput != "out/Fixture.json" {
		t.Fatalf("unexpected compiler output path: %s", cfg.CompilerOutput)
	}
	if cfg.Contract != "contracts/Fixture.sol:Fixture" {
		t.Fatalf("unexpected contract selector: %s", cfg.Contract)
	}
	if cfg.Variable != "data[4][9].b" {
		t.Fatalf("unexpected variable query: %s", cfg.Variable)
	}
	if cfg.Format != proof.StorageLayoutFormatBuildInfo {
		t.Fatalf("unexpected format: %s", cfg.Format)
	}
	if cfg.Out != "slot.json" {
		t.Fatalf("unexpected output path: %s", cfg.Out)
	}
}

func TestParseResolveSlotArgsDefaultsFormat(t *testing.T) {
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
}

func TestParseResolveSlotArgsRejectsMissingRequiredFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "missing compiler output",
			args: []string{"--contract", "Fixture", "--var", "value"},
			want: "requires --compiler-output",
		},
		{
			name: "missing contract",
			args: []string{"--compiler-output", "out/Fixture.json", "--var", "value"},
			want: "requires --contract",
		},
		{
			name: "missing query",
			args: []string{"--compiler-output", "out/Fixture.json", "--contract", "Fixture"},
			want: "requires --var",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseResolveSlotArgs(tt.args)
			if err == nil {
				t.Fatal("expected parseResolveSlotArgs to fail")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestParseResolveSlotArgsRejectsInvalidFormat(t *testing.T) {
	_, err := parseResolveSlotArgs([]string{
		"--compiler-output", "out/Fixture.json",
		"--contract", "Fixture",
		"--var", "value",
		"--format", "weird",
	})
	if err == nil {
		t.Fatal("expected invalid format to fail")
	}
	if !strings.Contains(err.Error(), "unsupported storage layout format") {
		t.Fatalf("unexpected error: %v", err)
	}
}
