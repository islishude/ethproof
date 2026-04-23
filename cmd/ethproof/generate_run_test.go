package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/islishude/ethproof/proof"
)

func TestRunGenerateCommandsSuccess(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		run     func([]string) error
		install func(*commandDeps, *generateCalls, *saveCall)
		assert  func(*testing.T, generateCalls, saveCall)
	}{
		{
			name: "state",
			args: []string{
				"--rpc", "http://127.0.0.1:8545",
				"--min-rpcs", "1",
				"--block", "12",
				"--account", "0x1111111111111111111111111111111111111111",
				"--slot", "0x01",
				"--out", "state.json",
			},
			run: runGenerateState,
			install: func(deps *commandDeps, calls *generateCalls, save *saveCall) {
				deps.generateState = func(_ context.Context, req proof.StateProofRequest) (*proof.StateProofPackage, error) {
					calls.state = &req
					return stubStateProofPackage(), nil
				}
				deps.saveJSON = func(path string, v any) error {
					save.path = path
					save.value = v
					return nil
				}
			},
			assert: func(t *testing.T, calls generateCalls, save saveCall) {
				if calls.state == nil || calls.state.BlockNumber != 12 || calls.state.Account != common.HexToAddress("0x1111111111111111111111111111111111111111") {
					t.Fatalf("unexpected state request: %+v", calls.state)
				}
				if save.path != "state.json" {
					t.Fatalf("unexpected save path: %s", save.path)
				}
				if _, ok := save.value.(*proof.StateProofPackage); !ok {
					t.Fatalf("unexpected saved value type: %T", save.value)
				}
			},
		},
		{
			name: "receipt",
			args: []string{
				"--rpc", "http://127.0.0.1:8545",
				"--min-rpcs", "1",
				"--tx", "0x02",
				"--log-index", "3",
				"--out", "receipt.json",
			},
			run: runGenerateReceipt,
			install: func(deps *commandDeps, calls *generateCalls, save *saveCall) {
				deps.generateReceipt = func(_ context.Context, req proof.ReceiptProofRequest) (*proof.ReceiptProofPackage, error) {
					calls.receipt = &req
					return stubReceiptProofPackage(), nil
				}
				deps.saveJSON = func(path string, v any) error {
					save.path = path
					save.value = v
					return nil
				}
			},
			assert: func(t *testing.T, calls generateCalls, save saveCall) {
				if calls.receipt == nil || calls.receipt.TxHash != common.HexToHash("0x02") || calls.receipt.LogIndex != 3 {
					t.Fatalf("unexpected receipt request: %+v", calls.receipt)
				}
				if save.path != "receipt.json" {
					t.Fatalf("unexpected save path: %s", save.path)
				}
				if _, ok := save.value.(*proof.ReceiptProofPackage); !ok {
					t.Fatalf("unexpected saved value type: %T", save.value)
				}
			},
		},
		{
			name: "transaction",
			args: []string{
				"--rpc", "http://127.0.0.1:8545",
				"--min-rpcs", "1",
				"--tx", "0x03",
				"--out", "tx.json",
			},
			run: runGenerateTransaction,
			install: func(deps *commandDeps, calls *generateCalls, save *saveCall) {
				deps.generateTransaction = func(_ context.Context, req proof.TransactionProofRequest) (*proof.TransactionProofPackage, error) {
					calls.transaction = &req
					return stubTransactionProofPackage(), nil
				}
				deps.saveJSON = func(path string, v any) error {
					save.path = path
					save.value = v
					return nil
				}
			},
			assert: func(t *testing.T, calls generateCalls, save saveCall) {
				if calls.transaction == nil || calls.transaction.TxHash != common.HexToHash("0x03") {
					t.Fatalf("unexpected tx request: %+v", calls.transaction)
				}
				if save.path != "tx.json" {
					t.Fatalf("unexpected save path: %s", save.path)
				}
				if _, ok := save.value.(*proof.TransactionProofPackage); !ok {
					t.Fatalf("unexpected saved value type: %T", save.value)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls generateCalls
			var save saveCall
			withCLIDeps(t, func(deps *commandDeps) {
				tt.install(deps, &calls, &save)
			})
			if err := tt.run(tt.args); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.assert(t, calls, save)
		})
	}
}

func TestRunGenerateCommandsWrapErrors(t *testing.T) {
	tests := []struct {
		name    string
		run     func([]string) error
		args    []string
		install func(*commandDeps, error)
		want    string
	}{
		{
			name: "state generate error",
			run:  runGenerateState,
			args: []string{"--rpc", "http://127.0.0.1:8545", "--min-rpcs", "1", "--block", "12", "--account", "0x1111111111111111111111111111111111111111", "--slot", "0x01"},
			install: func(deps *commandDeps, err error) {
				deps.generateState = func(context.Context, proof.StateProofRequest) (*proof.StateProofPackage, error) { return nil, err }
			},
			want: "generate state proof",
		},
		{
			name: "receipt save error",
			run:  runGenerateReceipt,
			args: []string{"--rpc", "http://127.0.0.1:8545", "--min-rpcs", "1", "--tx", "0x02", "--log-index", "3"},
			install: func(deps *commandDeps, err error) {
				deps.generateReceipt = func(context.Context, proof.ReceiptProofRequest) (*proof.ReceiptProofPackage, error) {
					return stubReceiptProofPackage(), nil
				}
				deps.saveJSON = func(string, any) error { return err }
			},
			want: "write receipt proof",
		},
		{
			name: "transaction generate error",
			run:  runGenerateTransaction,
			args: []string{"--rpc", "http://127.0.0.1:8545", "--min-rpcs", "1", "--tx", "0x03"},
			install: func(deps *commandDeps, err error) {
				deps.generateTransaction = func(context.Context, proof.TransactionProofRequest) (*proof.TransactionProofPackage, error) {
					return nil, err
				}
			},
			want: "generate transaction proof",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentinel := errors.New("boom")
			withCLIDeps(t, func(deps *commandDeps) {
				tt.install(deps, sentinel)
			})
			err := tt.run(tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) || !strings.Contains(err.Error(), "boom") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestRunMainGenerateRuntimeErrorWritesErrorLog(t *testing.T) {
	withCLIDeps(t, func(deps *commandDeps) {
		deps.generateTransaction = func(context.Context, proof.TransactionProofRequest) (*proof.TransactionProofPackage, error) {
			return nil, errors.New("boom")
		}
	})

	var exit int
	stdout, stderr := captureCommandOutput(t, func() {
		exit = runMain([]string{
			"generate", "tx",
			"--rpc", "http://127.0.0.1:8545",
			"--min-rpcs", "1",
			"--tx", "0x03",
		})
	})

	if exit != 1 || stdout != "" {
		t.Fatalf("unexpected result: exit=%d stdout=%q", exit, stdout)
	}
	if !strings.Contains(stderr, "generate transaction proof") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

type generateCalls struct {
	state       *proof.StateProofRequest
	receipt     *proof.ReceiptProofRequest
	transaction *proof.TransactionProofRequest
}

type saveCall struct {
	path  string
	value any
}

func stubStateProofPackage() *proof.StateProofPackage {
	return &proof.StateProofPackage{
		Block: proof.BlockContext{
			BlockNumber: 12,
			StateRoot:   common.HexToHash("0x01"),
		},
		Account: common.HexToAddress("0x1111111111111111111111111111111111111111"),
		StorageProofs: []proof.StateStorageProof{{
			Slot: common.Hash{},
		}},
	}
}

func stubReceiptProofPackage() *proof.ReceiptProofPackage {
	return &proof.ReceiptProofPackage{
		Block: proof.BlockContext{
			BlockNumber:  34,
			ReceiptsRoot: common.HexToHash("0x02"),
		},
		TxIndex: 2,
	}
}

func stubTransactionProofPackage() *proof.TransactionProofPackage {
	return &proof.TransactionProofPackage{
		Block: proof.BlockContext{
			BlockNumber:      56,
			TransactionsRoot: common.HexToHash("0x03"),
		},
		TxIndex: 1,
	}
}
