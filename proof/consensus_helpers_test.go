package proof

import (
	"slices"
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
		other.StorageProofs[0].Value = common.HexToHash("0x9999")

		diffs := compareStateSnapshot(base, other)
		if !slices.Contains(diffs, "storageProofs[0].value mismatch") {
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

func testAccountSnapshot() *accountSnapshot {
	return &accountSnapshot{
		Header: testBlockHeader(),
		Account: common.HexToAddress(
			"0x1111111111111111111111111111111111111111",
		),
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
		StorageProofs: []StateStorageProof{{
			Slot:  common.HexToHash("0x01"),
			Value: common.HexToHash("0x3000"),
			ProofNodes: []hexutil.Bytes{
				{0x03, 0x04},
			},
		}, {
			Slot:  common.HexToHash("0x02"),
			Value: common.HexToHash("0x4000"),
			ProofNodes: []hexutil.Bytes{
				{0x05, 0x06},
			},
		}},
	}
}

func cloneAccountSnapshot(in *accountSnapshot) *accountSnapshot {
	out := *in
	out.Header = cloneBlockSnapshotHeader(in.Header)
	out.AccountRLP = proofutil.CanonicalBytes(in.AccountRLP)
	out.AccountProof = cloneHexBytesList(in.AccountProof)
	out.StorageProofs = cloneStateStorageProofs(in.StorageProofs)
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
