package proof

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type methodNotFoundRPCError struct{}

func (methodNotFoundRPCError) Error() string {
	return "method not found"
}

func (methodNotFoundRPCError) ErrorCode() int {
	return -32601
}

func TestGenerateReceiptProofResponse(t *testing.T) {
	source, txHash, logIndex := mustReceiptSource(t)

	pkg, err := GenerateReceiptProof(context.Background(), source, ReceiptProofRequest{TxHash: txHash, LogIndex: logIndex})
	if err != nil {
		t.Fatalf("GenerateReceiptProof: %v", err)
	}
	if pkg.TxHash != txHash {
		t.Fatalf("unexpected tx hash: got %s want %s", pkg.TxHash, txHash)
	}
	if pkg.LogIndex != logIndex {
		t.Fatalf("unexpected log index: got %d want %d", pkg.LogIndex, logIndex)
	}
	if err := VerifyReceiptProofPackageAgainstEmbeddedRoots(pkg); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateReceiptProofResponseFailures(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*types.Receipt)
		want   string
	}{
		{
			name: "log index out of range",
			mutate: func(receipt *types.Receipt) {
				receipt.Logs = nil
			},
			want: "log-index",
		},
		{
			name: "target receipt block hash mismatch",
			mutate: func(receipt *types.Receipt) {
				receipt.BlockHash = common.HexToHash("0xbeef")
			},
			want: "fetch block",
		},
		{
			name: "target receipt transaction index mismatch",
			mutate: func(receipt *types.Receipt) {
				receipt.TransactionIndex = 0
			},
			want: "block transaction",
		},
		{
			name: "target receipt tx hash mismatch",
			mutate: func(receipt *types.Receipt) {
				receipt.TxHash = common.HexToHash("0xbeef")
			},
			want: "receipt tx hash mismatch",
		},
		{
			name: "receipt bytes mismatch",
			mutate: func(receipt *types.Receipt) {
				receipt.Status = types.ReceiptStatusFailed
			},
			want: "receipt bytes mismatch between block receipts and target receipt lookup",
		},
		{
			name: "target receipt nil log",
			mutate: func(receipt *types.Receipt) {
				receipt.Logs[0] = nil
			},
			want: "target receipt log 0 is nil",
		},
		{
			name: "target receipt log removed",
			mutate: func(receipt *types.Receipt) {
				receipt.Logs[0].Removed = true
			},
			want: "target receipt log 0 is marked removed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, txHash, logIndex := mustReceiptSource(t)
			tt.mutate(source.receiptsByTxHash[txHash])
			_, err := GenerateReceiptProof(context.Background(), source, ReceiptProofRequest{TxHash: txHash, LogIndex: logIndex})
			if err == nil {
				t.Fatal("expected GenerateReceiptProof to fail")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestFetchBlockReceipts(t *testing.T) {
	t.Run("direct block receipts path", func(t *testing.T) {
		source, _, _ := mustReceiptSource(t)

		got, err := fetchBlockReceipts(context.Background(), source, source.block.Hash(), source.block.Transactions(), nil)
		if err != nil {
			t.Fatalf("fetchBlockReceipts: %v", err)
		}
		want, err := encodeAndValidateBlockReceipts(source.blockReceipts, source.block.Hash(), len(source.block.Transactions()))
		if err != nil {
			t.Fatalf("encodeAndValidateBlockReceipts: %v", err)
		}
		for i := range got {
			if !bytes.Equal(got[i], want[i]) {
				t.Fatalf("receipt %d mismatch", i)
			}
		}
	})

	t.Run("falls back to transaction scan when method missing", func(t *testing.T) {
		source, _, _ := mustReceiptSource(t)
		source.blockReceiptsErr = methodNotFoundRPCError{}

		got, err := fetchBlockReceipts(context.Background(), source, source.block.Hash(), source.block.Transactions(), nil)
		if err != nil {
			t.Fatalf("fetchBlockReceipts: %v", err)
		}
		want, err := encodeAndValidateBlockReceipts(source.blockReceipts, source.block.Hash(), len(source.block.Transactions()))
		if err != nil {
			t.Fatalf("encodeAndValidateBlockReceipts: %v", err)
		}
		for i := range got {
			if !bytes.Equal(got[i], want[i]) {
				t.Fatalf("receipt %d mismatch", i)
			}
		}
	})

	t.Run("scan rejects receipt mismatch", func(t *testing.T) {
		source, txHash, _ := mustReceiptSource(t)
		source.receiptsByTxHash[txHash].BlockHash = common.HexToHash("0xbeef")

		_, err := fetchBlockReceiptsByTransactionScan(context.Background(), source, source.block.Hash(), source.block.Transactions(), nil)
		if err == nil {
			t.Fatal("expected fetchBlockReceiptsByTransactionScan to fail")
		}
		if !strings.Contains(err.Error(), "block hash mismatch") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestGenerateReceiptProof(t *testing.T) {
	req, source := testReceiptRequest(t)

	pkg, err := GenerateReceiptProof(context.Background(), source, req)
	if err != nil {
		t.Fatalf("GenerateReceiptProof: %v", err)
	}
	expect := &ReceiptExpectations{
		Emitter: &pkg.Event.Address,
		Topics:  append([]common.Hash(nil), pkg.Event.Topics...),
		Data:    append([]byte(nil), pkg.Event.Data...),
	}
	if err := VerifyReceiptProofPackageWithExpectations(pkg, expect); err != nil {
		t.Fatalf("VerifyReceiptProofPackageWithExpectations: %v", err)
	}
	if err := VerifyReceiptProofPackageWithExpectationsAgainstSource(context.Background(), pkg, expect, source); err != nil {
		t.Fatalf("VerifyReceiptProofPackageWithExpectationsAgainstSource: %v", err)
	}
}

func mustReceiptSource(t *testing.T) (*fakeReceiptSource, common.Hash, uint64) {
	t.Helper()

	source, txHash, logIndex := testReceiptSource(t)
	return source, txHash, logIndex
}
