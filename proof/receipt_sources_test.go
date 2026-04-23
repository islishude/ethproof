package proof

import (
	"bytes"
	"context"
	"slices"
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

type secondReceiptSource struct {
	*fakeReceiptSource
	second *types.Receipt
	calls  int
}

func (s *secondReceiptSource) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	s.calls++
	if s.calls == 1 || s.second == nil {
		return s.fakeReceiptSource.TransactionReceipt(ctx, txHash)
	}
	return cloneReceipt(s.second), nil
}

func TestFetchReceiptSnapshot(t *testing.T) {
	source, txHash, logIndex := mustReceiptSource(t)

	snapshot, err := fetchReceiptSnapshot(context.Background(), source, txHash, logIndex)
	if err != nil {
		t.Fatalf("fetchReceiptSnapshot: %v", err)
	}
	if snapshot.TxHash != txHash {
		t.Fatalf("unexpected tx hash: got %s want %s", snapshot.TxHash, txHash)
	}
	if snapshot.LogIndex != logIndex {
		t.Fatalf("unexpected log index: got %d want %d", snapshot.LogIndex, logIndex)
	}
	if got, want := len(snapshot.BlockReceipts), len(source.blockReceipts); got != want {
		t.Fatalf("unexpected block receipt count: got %d want %d", got, want)
	}
}

func TestFetchReceiptSnapshotFailures(t *testing.T) {
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
			want: "target receipt block hash mismatch",
		},
		{
			name: "target receipt transaction index mismatch",
			mutate: func(receipt *types.Receipt) {
				receipt.TransactionIndex = 0
			},
			want: "target receipt transaction index mismatch",
		},
		{
			name: "target receipt tx hash mismatch",
			mutate: func(receipt *types.Receipt) {
				receipt.TxHash = common.HexToHash("0xbeef")
			},
			want: "target receipt tx hash mismatch",
		},
		{
			name: "receipt bytes mismatch",
			mutate: func(receipt *types.Receipt) {
				receipt.Status = types.ReceiptStatusFailed
			},
			want: "receipt bytes mismatch between block receipts and target receipt lookup",
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
			second := cloneReceipt(source.receiptsByTxHash[txHash])
			tt.mutate(second)

			_, err := fetchReceiptSnapshot(context.Background(), &secondReceiptSource{
				fakeReceiptSource: source,
				second:            second,
			}, txHash, logIndex)
			if err == nil {
				t.Fatal("expected fetchReceiptSnapshot to fail")
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

		got, err := fetchBlockReceipts(context.Background(), source, source.block.Hash(), len(source.block.Transactions()))
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

		got, err := fetchBlockReceipts(context.Background(), source, source.block.Hash(), len(source.block.Transactions()))
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

		_, err := fetchBlockReceiptsByTransactionScan(context.Background(), source, source.block.Hash(), len(source.block.Transactions()))
		if err == nil {
			t.Fatal("expected fetchBlockReceiptsByTransactionScan to fail")
		}
		if !strings.Contains(err.Error(), "block hash mismatch") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestGenerateReceiptProofFromSources(t *testing.T) {
	req, verifyReq, wantNames := testReceiptProofSourcesRequest(t)

	pkg, err := GenerateReceiptProofFromSources(context.Background(), req)
	if err != nil {
		t.Fatalf("GenerateReceiptProofFromSources: %v", err)
	}
	if got := pkg.Block.SourceConsensus.RPCs; !slices.Equal(got, wantNames) {
		t.Fatalf("unexpected source names: got %v want %v", got, wantNames)
	}
	expect := &ReceiptExpectations{
		Emitter: &pkg.Event.Address,
		Topics:  append([]common.Hash(nil), pkg.Event.Topics...),
		Data:    append([]byte(nil), pkg.Event.Data...),
	}
	if err := VerifyReceiptProofPackageWithExpectations(pkg, expect); err != nil {
		t.Fatalf("VerifyReceiptProofPackageWithExpectations: %v", err)
	}
	if err := VerifyReceiptProofPackageWithExpectationsAgainstSources(context.Background(), pkg, expect, verifyReq); err != nil {
		t.Fatalf("VerifyReceiptProofPackageWithExpectationsAgainstSources: %v", err)
	}
}

func mustReceiptSource(t *testing.T) (*fakeReceiptSource, common.Hash, uint) {
	t.Helper()

	sources, txHash, logIndex, _ := testReceiptSourceSet(t)
	source, ok := sources[0].(*fakeReceiptSource)
	if !ok {
		t.Fatal("expected fakeReceiptSource")
	}
	return source, txHash, logIndex
}
