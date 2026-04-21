package proof

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestEncodeAndValidateBlockReceipts(t *testing.T) {
	header, txs, receipts, _, _, err := buildOfflineTransactionReceiptFixture()
	if err != nil {
		t.Fatalf("buildOfflineTransactionReceiptFixture: %v", err)
	}

	blockReceipts := receiptsWithInclusionInfo(header.BlockHash, txs, receipts)
	got, err := encodeAndValidateBlockReceipts(blockReceipts, header.BlockHash, len(receipts))
	if err != nil {
		t.Fatalf("encodeAndValidateBlockReceipts: %v", err)
	}
	if len(got) != len(receipts) {
		t.Fatalf("unexpected receipt count: got %d want %d", len(got), len(receipts))
	}
	for i, receipt := range receipts {
		want, err := encodeReceipt(receipt)
		if err != nil {
			t.Fatalf("encodeReceipt(%d): %v", i, err)
		}
		if got[i] != want {
			t.Fatalf("receipt %d mismatch", i)
		}
	}
}

func TestEncodeAndValidateBlockReceiptsRejectsMismatchedIndex(t *testing.T) {
	header, txs, receipts, _, _, err := buildOfflineTransactionReceiptFixture()
	if err != nil {
		t.Fatalf("buildOfflineTransactionReceiptFixture: %v", err)
	}

	blockReceipts := receiptsWithInclusionInfo(header.BlockHash, txs, receipts)
	blockReceipts[1].TransactionIndex = 0

	_, err = encodeAndValidateBlockReceipts(blockReceipts, header.BlockHash, len(receipts))
	if err == nil {
		t.Fatal("expected mismatched transaction index to fail")
	}
	if !strings.Contains(err.Error(), "transaction index mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func receiptsWithInclusionInfo(blockHash common.Hash, txs types.Transactions, receipts types.Receipts) []*types.Receipt {
	out := make([]*types.Receipt, len(receipts))
	for i, receipt := range receipts {
		cloned := *receipt
		cloned.BlockHash = blockHash
		cloned.TransactionIndex = uint(i)
		cloned.TxHash = txs[i].Hash()
		out[i] = &cloned
	}
	return out
}
