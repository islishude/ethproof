package proof

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
	"github.com/islishude/ethproof/internal/proofutil"
)

func TestCompareSnapshotsReportChangedFields(t *testing.T) {
	t.Run("state", func(t *testing.T) {
		base := testAccountSnapshot()
		other := cloneAccountSnapshot(base)
		other.StorageValue = common.HexToHash("0x9999")

		diffs := compareStateSnapshot(base, other)
		if !slices.Contains(diffs, "storageValue mismatch") {
			t.Fatalf("expected storage value mismatch, got %v", diffs)
		}
	})

	t.Run("receipt", func(t *testing.T) {
		base := testReceiptSnapshot()
		other := cloneReceiptSnapshot(base)
		other.LogIndex++

		diffs := compareReceiptSnapshot(base, other)
		if !slices.Contains(diffs, "logIndex mismatch") {
			t.Fatalf("expected log index mismatch, got %v", diffs)
		}
	})

	t.Run("transaction", func(t *testing.T) {
		base := testTransactionSnapshot()
		other := cloneTransactionSnapshot(base)
		other.TxIndex++

		diffs := compareTransactionSnapshot(base, other)
		if !slices.Contains(diffs, "txIndex mismatch") {
			t.Fatalf("expected tx index mismatch, got %v", diffs)
		}
	})
}

func TestBuildConsensusBuilders(t *testing.T) {
	t.Run("state", func(t *testing.T) {
		consensus, err := buildStateConsensus(testAccountSnapshot(), []string{"rpc-a", "rpc-b", "rpc-c"})
		if err != nil {
			t.Fatalf("buildStateConsensus: %v", err)
		}
		if consensus.Mode != "live-rpc" {
			t.Fatalf("unexpected mode: %s", consensus.Mode)
		}
		if got, want := len(consensus.Digests), 3; got != want {
			t.Fatalf("unexpected digest count: got %d want %d", got, want)
		}
	})

	t.Run("receipt", func(t *testing.T) {
		consensus, err := buildReceiptConsensus(testReceiptSnapshot(), []string{"rpc-a", "rpc-b", "rpc-c"})
		if err != nil {
			t.Fatalf("buildReceiptConsensus: %v", err)
		}
		if consensus.Mode != "live-rpc" {
			t.Fatalf("unexpected mode: %s", consensus.Mode)
		}
		if got, want := len(consensus.Digests), 4; got != want {
			t.Fatalf("unexpected digest count: got %d want %d", got, want)
		}
	})

	t.Run("transaction", func(t *testing.T) {
		consensus, err := buildTransactionConsensus(testTransactionSnapshot(), []string{"rpc-a", "rpc-b", "rpc-c"})
		if err != nil {
			t.Fatalf("buildTransactionConsensus: %v", err)
		}
		if consensus.Mode != "live-rpc" {
			t.Fatalf("unexpected mode: %s", consensus.Mode)
		}
		if got, want := len(consensus.Digests), 3; got != want {
			t.Fatalf("unexpected digest count: got %d want %d", got, want)
		}
	})
}

func TestRequireMatchingSnapshotsRejectsMismatch(t *testing.T) {
	base := testTransactionSnapshot()
	other := cloneTransactionSnapshot(base)
	other.TxHash = common.HexToHash("0x7777")

	_, err := requireMatchingSnapshots([]string{"rpc-a", "rpc-b"}, []*transactionSnapshot{base, other}, compareTransactionSnapshot)
	if err == nil {
		t.Fatal("expected mismatch to fail")
	}
	if !strings.Contains(err.Error(), "normalized data mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCollectFromRPCsWrapsSourceErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := collectFromRPCs(context.Background(), []string{server.URL}, func(_ context.Context, _ *rpcSource) (int, error) {
		return 0, errors.New("boom")
	})
	if err == nil {
		t.Fatal("expected source error")
	}
	if !strings.Contains(err.Error(), server.URL+": boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func testAccountSnapshot() *accountSnapshot {
	return &accountSnapshot{
		Header: testBlockHeader(),
		Account: common.HexToAddress(
			"0x1111111111111111111111111111111111111111",
		),
		Slot:       common.HexToHash("0x01"),
		AccountRLP: hexutil.Bytes{0xaa, 0xbb},
		AccountProof: []hexutil.Bytes{
			{0x01, 0x02},
		},
		AccountClaim: StateAccountClaim{
			Nonce:       7,
			Balance:     "0x1234",
			StorageRoot: common.HexToHash("0x1000"),
			CodeHash:    common.HexToHash("0x2000"),
		},
		StorageValue: common.HexToHash("0x3000"),
		StorageProof: []hexutil.Bytes{
			{0x03, 0x04},
		},
	}
}

func cloneAccountSnapshot(in *accountSnapshot) *accountSnapshot {
	out := *in
	out.Header = cloneBlockSnapshotHeader(in.Header)
	out.AccountRLP = proofutil.CanonicalBytes(in.AccountRLP)
	out.AccountProof = cloneHexBytesList(in.AccountProof)
	out.StorageProof = cloneHexBytesList(in.StorageProof)
	return &out
}

func testReceiptSnapshot() *receiptSnapshot {
	return &receiptSnapshot{
		Header:         testBlockHeader(),
		TxHash:         common.HexToHash("0x4000"),
		TxIndex:        1,
		LogIndex:       0,
		TransactionRLP: hexutil.Bytes{0x01},
		ReceiptRLP:     hexutil.Bytes{0x02},
		BlockTransactions: []hexutil.Bytes{
			{0x01},
			{0x02},
		},
		BlockReceipts: []hexutil.Bytes{
			{0x03},
			{0x04},
		},
		Event: EventClaim{
			Address: common.HexToAddress("0x2222222222222222222222222222222222222222"),
			Topics: []common.Hash{
				common.HexToHash("0x5000"),
			},
			Data: hexutil.Bytes{0x05},
		},
	}
}

func cloneReceiptSnapshot(in *receiptSnapshot) *receiptSnapshot {
	out := *in
	out.Header = cloneBlockSnapshotHeader(in.Header)
	out.TransactionRLP = proofutil.CanonicalBytes(in.TransactionRLP)
	out.ReceiptRLP = proofutil.CanonicalBytes(in.ReceiptRLP)
	out.BlockTransactions = cloneHexBytesList(in.BlockTransactions)
	out.BlockReceipts = cloneHexBytesList(in.BlockReceipts)
	out.Event.Topics = append([]common.Hash(nil), in.Event.Topics...)
	out.Event.Data = proofutil.CanonicalBytes(in.Event.Data)
	return &out
}

func testTransactionSnapshot() *transactionSnapshot {
	return &transactionSnapshot{
		Header:         testBlockHeader(),
		TxHash:         common.HexToHash("0x6000"),
		TxIndex:        0,
		TransactionRLP: hexutil.Bytes{0x06},
		BlockTransactions: []hexutil.Bytes{
			{0x06},
			{0x07},
		},
	}
}

func cloneTransactionSnapshot(in *transactionSnapshot) *transactionSnapshot {
	out := *in
	out.Header = cloneBlockSnapshotHeader(in.Header)
	out.TransactionRLP = proofutil.CanonicalBytes(in.TransactionRLP)
	out.BlockTransactions = cloneHexBytesList(in.BlockTransactions)
	return &out
}

func testBlockHeader() blockSnapshotHeader {
	return blockSnapshotHeader{
		ChainID:          uint256.NewInt(1),
		BlockNumber:      99,
		BlockHash:        common.HexToHash("0x7000"),
		ParentHash:       common.HexToHash("0x8000"),
		StateRoot:        common.HexToHash("0x9000"),
		TransactionsRoot: common.HexToHash("0xa000"),
		ReceiptsRoot:     common.HexToHash("0xb000"),
	}
}
