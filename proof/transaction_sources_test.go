package proof

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
)

type overrideBlockReceiptSource struct {
	*fakeReceiptSource
	override *types.Block
}

func (s *overrideBlockReceiptSource) BlockByHash(context.Context, common.Hash) (*types.Block, error) {
	return s.override, nil
}

type overrideBlockAndHeaderSource struct {
	*overrideBlockReceiptSource
}

func (s *overrideBlockAndHeaderSource) HeaderByHash(context.Context, common.Hash) (*types.Header, error) {
	return cloneHeader(s.override.Header()), nil
}

func TestFetchTransactionSnapshot(t *testing.T) {
	source, txHash := mustTransactionSource(t)

	snapshot, err := fetchTransactionSnapshot(context.Background(), source, txHash)
	if err != nil {
		t.Fatalf("fetchTransactionSnapshot: %v", err)
	}
	if snapshot.TxHash != txHash {
		t.Fatalf("unexpected tx hash: got %s want %s", snapshot.TxHash, txHash)
	}
	if got, want := len(snapshot.BlockTransactions), len(source.block.Transactions()); got != want {
		t.Fatalf("unexpected block transaction count: got %d want %d", got, want)
	}
}

func TestFetchTransactionSnapshotFailures(t *testing.T) {
	tests := []struct {
		name string
		run  func(*fakeReceiptSource, common.Hash) error
		want string
	}{
		{
			name: "pending transaction",
			run: func(source *fakeReceiptSource, txHash common.Hash) error {
				source.pending = true
				_, err := fetchTransactionSnapshot(context.Background(), source, txHash)
				return err
			},
			want: "transaction is still pending",
		},
		{
			name: "transaction index out of range",
			run: func(source *fakeReceiptSource, txHash common.Hash) error {
				source.receiptsByTxHash[txHash].TransactionIndex = uint(len(source.block.Transactions()))
				_, err := fetchTransactionSnapshot(context.Background(), source, txHash)
				return err
			},
			want: "out of range",
		},
		{
			name: "header hash mismatch",
			run: func(source *fakeReceiptSource, txHash common.Hash) error {
				txs := append(types.Transactions(nil), source.block.Transactions()...)
				overrideHeader := cloneHeader(source.block.Header())
				overrideHeader.ParentHash = common.HexToHash("0xbeef")
				overrideBlock := types.NewBlock(overrideHeader, &types.Body{Transactions: txs}, cloneReceiptList(source.blockReceipts), trie.NewStackTrie(nil))
				_, err := fetchTransactionSnapshot(context.Background(), &overrideBlockReceiptSource{
					fakeReceiptSource: source,
					override:          overrideBlock,
				}, txHash)
				return err
			},
			want: "header hash",
		},
		{
			name: "block body tx mismatch",
			run: func(source *fakeReceiptSource, txHash common.Hash) error {
				txs := append(types.Transactions(nil), source.block.Transactions()...)
				wrongRecipient := common.HexToAddress("0x9999999999999999999999999999999999999999")
				txs[1] = types.NewTx(&types.LegacyTx{
					Nonce:    9,
					To:       &wrongRecipient,
					Value:    common.Big1,
					Gas:      21_000,
					GasPrice: common.Big1,
				})
				overrideBlock := types.NewBlock(cloneHeader(source.block.Header()), &types.Body{Transactions: txs}, cloneReceiptList(source.blockReceipts), trie.NewStackTrie(nil))
				_, err := fetchTransactionSnapshot(context.Background(), &overrideBlockAndHeaderSource{
					overrideBlockReceiptSource: &overrideBlockReceiptSource{
						fakeReceiptSource: source,
						override:          overrideBlock,
					},
				}, txHash)
				return err
			},
			want: "block transaction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, txHash := mustTransactionSource(t)
			err := tt.run(source, txHash)
			if err == nil {
				t.Fatal("expected fetchTransactionSnapshot to fail")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestGenerateTransactionProofFromSources(t *testing.T) {
	req, verifyReq, wantNames := testTransactionProofSourcesRequest(t)

	pkg, err := GenerateTransactionProofFromSources(context.Background(), req)
	if err != nil {
		t.Fatalf("GenerateTransactionProofFromSources: %v", err)
	}
	if got := pkg.Block.SourceConsensus.RPCs; !slices.Equal(got, wantNames) {
		t.Fatalf("unexpected source names: got %v want %v", got, wantNames)
	}
	if err := VerifyTransactionProofPackage(pkg); err != nil {
		t.Fatalf("VerifyTransactionProofPackage: %v", err)
	}
	if err := VerifyTransactionProofPackageAgainstSources(context.Background(), pkg, verifyReq); err != nil {
		t.Fatalf("VerifyTransactionProofPackageAgainstSources: %v", err)
	}
}

func mustTransactionSource(t *testing.T) (*fakeReceiptSource, common.Hash) {
	t.Helper()

	sources, txHash, _, _ := testReceiptSourceSet(t)
	source, ok := sources[0].(*fakeReceiptSource)
	if !ok {
		t.Fatal("expected fakeReceiptSource")
	}
	return source, txHash
}
