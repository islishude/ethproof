package main

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestParseGenerateStateArgsPassesMinRPCs(t *testing.T) {
	cfg := parseGenerateStateArgs([]string{
		"--rpc", "http://127.0.0.1:8545",
		"--min-rpcs", "1",
		"--block", "12",
		"--account", "0x1111111111111111111111111111111111111111",
		"--slot", "0x01",
		"--out", "state.json",
	})
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
	cfg := parseGenerateReceiptArgs([]string{
		"--rpc", "http://127.0.0.1:8545",
		"--tx", "0x02",
		"--log-index", "3",
	})
	if cfg.Request.MinRPCSources != proofMinRPCsDefault() {
		t.Fatalf("expected default MinRPCSources=%d, got %d", proofMinRPCsDefault(), cfg.Request.MinRPCSources)
	}
	if cfg.Request.LogIndex != 3 {
		t.Fatalf("expected log index 3, got %d", cfg.Request.LogIndex)
	}
}

func TestParseGenerateTransactionArgsPassesMinRPCs(t *testing.T) {
	cfg := parseGenerateTransactionArgs([]string{
		"--rpc", "http://127.0.0.1:8545",
		"--min-rpcs", "1",
		"--tx", "0x03",
		"--out", "tx.json",
	})
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
