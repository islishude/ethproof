package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/islishude/ethproof/proof"
)

func TestFinalityCLIConfigurationAndPrecedence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	config := `{"finality":{"network":"mainnet","beaconRpc":"https://beacon.example","checkpointRpcs":["https://one.example","https://two.example","https://three.example"]},"verify":{"tx":{"rpc":"https://verify.example","finality":true},"state":{"rpc":"https://verify.example"},"receipt":{"rpc":"https://verify.example","finality":true}}}`
	if err := os.WriteFile(path, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := parseVerifyTransactionArgs([]string{"--config", path})
	if err != nil || cfg.Finality == nil {
		t.Fatalf("%+v %v", cfg, err)
	}
	cfg, err = parseVerifyTransactionArgs([]string{"--config", path, "--finality=false"})
	if err != nil || cfg.Finality != nil {
		t.Fatalf("%+v %v", cfg, err)
	}
	state, err := parseVerifyStateArgs([]string{"--config", path, "--finality"})
	if err != nil || state.Finality == nil {
		t.Fatalf("%+v %v", state, err)
	}
	receipt, err := parseVerifyReceiptArgs([]string{"--config", path})
	if err != nil || receipt.Finality == nil {
		t.Fatalf("%+v %v", receipt, err)
	}
	if _, err = parseVerifyTransactionArgs([]string{"--rpc", "https://verify.example", "--finality"}); err == nil {
		t.Fatal("accepted missing finality config")
	}
	if _, err = parseVerifyTransactionArgs([]string{"--config", path, "--finality", "--rpc", ""}); err == nil {
		t.Fatal("finality bypassed independent RPC requirement")
	}
}
func TestFinalityCLIFailureNeverDowngrades(t *testing.T) {
	previous := cliDeps
	t.Cleanup(func() { cliDeps = previous })
	calls := 0
	sentinel := errors.New("finality unavailable")
	cliDeps.verifyFinality = func(_ context.Context, _ proof.BlockContext, rpc string, _ proof.FinalityConfig, _ func(proof.FinalityTrustEvent)) (proof.BlockAnchor, error) {
		calls++
		if rpc != "verify-rpc" {
			t.Fatal(rpc)
		}
		return proof.BlockAnchor{}, sentinel
	}
	if err := verifyFinality(t.Context(), proof.BlockContext{}, "verify-rpc", nil); err != nil || calls != 0 {
		t.Fatal("disabled finality performed network call")
	}
	if err := verifyFinality(t.Context(), proof.BlockContext{}, "verify-rpc", &proof.FinalityConfig{}); !errors.Is(err, sentinel) || calls != 1 {
		t.Fatal(err)
	}
}
