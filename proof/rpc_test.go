package proof

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestEncodeAndValidateBlockReceipts(t *testing.T) {
	blockHash, receipts := testBlockReceipts()

	blockReceipts := receiptsWithInclusionInfo(blockHash, receipts)
	got, err := encodeAndValidateBlockReceipts(blockReceipts, blockHash, len(receipts))
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
		if !bytes.Equal(got[i], want) {
			t.Fatalf("receipt %d mismatch", i)
		}
	}
}

func TestEncodeAndValidateBlockReceiptsRejectsMismatchedIndex(t *testing.T) {
	blockHash, receipts := testBlockReceipts()

	blockReceipts := receiptsWithInclusionInfo(blockHash, receipts)
	blockReceipts[1].TransactionIndex = 0

	_, err := encodeAndValidateBlockReceipts(blockReceipts, blockHash, len(receipts))
	if err == nil {
		t.Fatal("expected mismatched transaction index to fail")
	}
	if !strings.Contains(err.Error(), "transaction index mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func testBlockReceipts() (common.Hash, types.Receipts) {
	receipts := types.Receipts{
		{
			Type:              types.LegacyTxType,
			Status:            types.ReceiptStatusSuccessful,
			CumulativeGasUsed: 21_000,
			Logs: []*types.Log{{
				Address: common.HexToAddress("0x1000000000000000000000000000000000000001"),
			}},
		},
		{
			Type:              types.DynamicFeeTxType,
			Status:            types.ReceiptStatusSuccessful,
			CumulativeGasUsed: 42_000,
			Logs: []*types.Log{{
				Address: common.HexToAddress("0x2000000000000000000000000000000000000002"),
			}},
		},
	}
	for _, receipt := range receipts {
		receipt.Bloom = types.CreateBloom(receipt)
	}
	return common.HexToHash("0x1234"), receipts
}

func receiptsWithInclusionInfo(blockHash common.Hash, receipts types.Receipts) []*types.Receipt {
	out := make([]*types.Receipt, len(receipts))
	for i, receipt := range receipts {
		cloned := *receipt
		cloned.BlockHash = blockHash
		cloned.TransactionIndex = uint(i)
		out[i] = &cloned
	}
	return out
}
