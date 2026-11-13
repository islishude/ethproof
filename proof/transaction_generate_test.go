package proof

import (
	"context"
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

func TestFetchTransactionData(t *testing.T) {
	source, txHash := mustTransactionSource(t)

	data, err := fetchTransactionData(context.Background(), source, txHash)
	if err != nil {
		t.Fatalf("fetchTransactionData: %v", err)
	}
	if data.receipt.TxHash != txHash {
		t.Fatalf("unexpected tx hash: got %s want %s", data.receipt.TxHash, txHash)
	}
	if got, want := len(data.encodedTransactions), len(source.block.Transactions()); got != want {
		t.Fatalf("unexpected block transaction count: got %d want %d", got, want)
	}
}

func TestFetchTransactionDataFailures(t *testing.T) {
	tests := []struct {
		name string
		run  func(*fakeReceiptSource, common.Hash) error
		want string
	}{
		{
			name: "nil transaction",
			run: func(source *fakeReceiptSource, txHash common.Hash) error {
				source.txsByHash[txHash] = nil
				_, err := fetchTransactionData(context.Background(), source, txHash)
				return err
			},
			want: "nil transaction",
		},
		{
			name: "nil receipt",
			run: func(source *fakeReceiptSource, txHash common.Hash) error {
				source.receiptsByTxHash[txHash] = nil
				_, err := fetchTransactionData(context.Background(), source, txHash)
				return err
			},
			want: "nil receipt",
		},
		{
			name: "pending transaction",
			run: func(source *fakeReceiptSource, txHash common.Hash) error {
				source.pending = true
				_, err := fetchTransactionData(context.Background(), source, txHash)
				return err
			},
			want: "transaction is still pending",
		},
		{
			name: "transaction index out of range",
			run: func(source *fakeReceiptSource, txHash common.Hash) error {
				source.receiptsByTxHash[txHash].TransactionIndex = uint(len(source.block.Transactions()))
				_, err := fetchTransactionData(context.Background(), source, txHash)
				return err
			},
			want: "out of range",
		},
		{
			name: "maximum transaction index",
			run: func(source *fakeReceiptSource, txHash common.Hash) error {
				source.receiptsByTxHash[txHash].TransactionIndex = ^uint(0)
				_, err := fetchTransactionData(context.Background(), source, txHash)
				return err
			},
			want: "out of range",
		},
		{
			name: "receipt block hash mismatch",
			run: func(source *fakeReceiptSource, txHash common.Hash) error {
				txs := append(types.Transactions(nil), source.block.Transactions()...)
				overrideHeader := cloneHeader(source.block.Header())
				overrideHeader.ParentHash = common.HexToHash("0xbeef")
				overrideBlock := types.NewBlock(overrideHeader, &types.Body{Transactions: txs}, cloneReceiptList(source.blockReceipts), trie.NewStackTrie(nil))
				_, err := fetchTransactionData(context.Background(), &overrideBlockReceiptSource{
					fakeReceiptSource: source,
					override:          overrideBlock,
				}, txHash)
				return err
			},
			want: "receipt block hash",
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
				source.receiptsByTxHash[txHash].BlockHash = overrideBlock.Hash()
				_, err := fetchTransactionData(context.Background(), &overrideBlockReceiptSource{
					fakeReceiptSource: source,
					override:          overrideBlock,
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
				t.Fatal("expected fetchTransactionData to fail")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestGenerateTransactionProof(t *testing.T) {
	req, source := testTransactionRequest(t)

	pkg, err := GenerateTransactionProof(context.Background(), source, req)
	if err != nil {
		t.Fatalf("GenerateTransactionProof: %v", err)
	}
	if err := VerifyTransactionProofPackage(pkg); err != nil {
		t.Fatalf("VerifyTransactionProofPackage: %v", err)
	}
	if err := VerifyTransactionProofPackageAgainstSource(context.Background(), pkg, source); err != nil {
		t.Fatalf("VerifyTransactionProofPackageAgainstSource: %v", err)
	}
}

func mustTransactionSource(t *testing.T) (*fakeReceiptSource, common.Hash) {
	t.Helper()

	source, txHash, _ := testReceiptSource(t)
	return source, txHash
}
