package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/islishude/ethproof/proof"
)

func TestRunVerifyCommandsSuccess(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		run     func([]string) error
		install func(*commandDeps, *verifyCalls)
		assert  func(*testing.T, verifyCalls)
	}{
		{
			name: "state",
			args: []string{"--rpc", "http://127.0.0.1:8545", "--min-rpcs", "1", "--proof", "state.json"},
			run:  runVerifyState,
			install: func(deps *commandDeps, calls *verifyCalls) {
				deps.loadJSON = func(path string, v any) error {
					calls.loadPath = path
					*(v.(*proof.StateProofPackage)) = *stubStateProofPackage()
					return nil
				}
				deps.verifyState = func(_ context.Context, pkg *proof.StateProofPackage, req proof.VerifyRPCRequest) error {
					calls.stateReq = &req
					calls.statePkg = pkg
					return nil
				}
			},
			assert: func(t *testing.T, calls verifyCalls) {
				if calls.loadPath != "state.json" || calls.stateReq == nil || calls.statePkg == nil {
					t.Fatalf("unexpected state verify calls: %+v", calls)
				}
			},
		},
		{
			name: "receipt",
			args: []string{
				"--rpc", "http://127.0.0.1:8545",
				"--min-rpcs", "1",
				"--proof", "receipt.json",
				"--expect-emitter", "0x2222222222222222222222222222222222222222",
				"--expect-topic", "0x01",
				"--expect-data", "0xaa",
			},
			run: runVerifyReceipt,
			install: func(deps *commandDeps, calls *verifyCalls) {
				deps.loadJSON = func(path string, v any) error {
					calls.loadPath = path
					*(v.(*proof.ReceiptProofPackage)) = *stubReceiptProofPackage()
					return nil
				}
				deps.verifyReceipt = func(_ context.Context, pkg *proof.ReceiptProofPackage, expect *proof.ReceiptExpectations, req proof.VerifyRPCRequest) error {
					calls.receiptReq = &req
					calls.receiptPkg = pkg
					calls.receiptExpect = expect
					return nil
				}
			},
			assert: func(t *testing.T, calls verifyCalls) {
				if calls.loadPath != "receipt.json" || calls.receiptReq == nil || calls.receiptPkg == nil {
					t.Fatalf("unexpected receipt verify calls: %+v", calls)
				}
				if calls.receiptExpect == nil || calls.receiptExpect.Emitter == nil || *calls.receiptExpect.Emitter != common.HexToAddress("0x2222222222222222222222222222222222222222") {
					t.Fatalf("unexpected receipt expectations: %+v", calls.receiptExpect)
				}
			},
		},
		{
			name: "transaction",
			args: []string{"--rpc", "http://127.0.0.1:8545", "--min-rpcs", "1", "--proof", "tx.json"},
			run:  runVerifyTransaction,
			install: func(deps *commandDeps, calls *verifyCalls) {
				deps.loadJSON = func(path string, v any) error {
					calls.loadPath = path
					*(v.(*proof.TransactionProofPackage)) = *stubTransactionProofPackage()
					return nil
				}
				deps.verifyTransaction = func(_ context.Context, pkg *proof.TransactionProofPackage, req proof.VerifyRPCRequest) error {
					calls.transactionReq = &req
					calls.transactionPkg = pkg
					return nil
				}
			},
			assert: func(t *testing.T, calls verifyCalls) {
				if calls.loadPath != "tx.json" || calls.transactionReq == nil || calls.transactionPkg == nil {
					t.Fatalf("unexpected transaction verify calls: %+v", calls)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls verifyCalls
			withCLIDeps(t, func(deps *commandDeps) {
				tt.install(deps, &calls)
			})
			if err := tt.run(tt.args); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.assert(t, calls)
		})
	}
}

func TestRunVerifyCommandsWrapErrors(t *testing.T) {
	tests := []struct {
		name    string
		run     func([]string) error
		args    []string
		install func(*commandDeps, error)
		want    string
	}{
		{
			name: "state load error",
			run:  runVerifyState,
			args: []string{"--rpc", "http://127.0.0.1:8545", "--min-rpcs", "1", "--proof", "state.json"},
			install: func(deps *commandDeps, err error) {
				deps.loadJSON = func(string, any) error { return err }
			},
			want: "read state proof json",
		},
		{
			name: "receipt verify error",
			run:  runVerifyReceipt,
			args: []string{"--rpc", "http://127.0.0.1:8545", "--min-rpcs", "1", "--proof", "receipt.json"},
			install: func(deps *commandDeps, err error) {
				deps.loadJSON = func(string, any) error {
					return nil
				}
				deps.verifyReceipt = func(context.Context, *proof.ReceiptProofPackage, *proof.ReceiptExpectations, proof.VerifyRPCRequest) error {
					return err
				}
			},
			want: "verify receipt proof",
		},
		{
			name: "transaction verify error",
			run:  runVerifyTransaction,
			args: []string{"--rpc", "http://127.0.0.1:8545", "--min-rpcs", "1", "--proof", "tx.json"},
			install: func(deps *commandDeps, err error) {
				deps.loadJSON = func(string, any) error {
					*(new(proof.TransactionProofPackage)) = *stubTransactionProofPackage()
					return nil
				}
				deps.verifyTransaction = func(context.Context, *proof.TransactionProofPackage, proof.VerifyRPCRequest) error {
					return err
				}
			},
			want: "verify transaction proof",
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

func TestRunMainVerifyRuntimeErrorWritesErrorLog(t *testing.T) {
	withCLIDeps(t, func(deps *commandDeps) {
		deps.loadJSON = func(string, any) error {
			return errors.New("boom")
		}
	})

	var exit int
	stdout, stderr := captureCommandOutput(t, func() {
		exit = runMain([]string{
			"verify", "tx",
			"--rpc", "http://127.0.0.1:8545",
			"--min-rpcs", "1",
			"--proof", "tx.json",
		})
	})

	if exit != 1 || stdout != "" {
		t.Fatalf("unexpected result: exit=%d stdout=%q", exit, stdout)
	}
	if !strings.Contains(stderr, "level=ERROR") || !strings.Contains(stderr, "read transaction proof json") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

type verifyCalls struct {
	loadPath       string
	stateReq       *proof.VerifyRPCRequest
	statePkg       *proof.StateProofPackage
	receiptReq     *proof.VerifyRPCRequest
	receiptPkg     *proof.ReceiptProofPackage
	receiptExpect  *proof.ReceiptExpectations
	transactionReq *proof.VerifyRPCRequest
	transactionPkg *proof.TransactionProofPackage
}
