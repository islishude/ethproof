package proof

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/islishude/ethproof/internal/proofutil"
)

func consensusForReceiptSnapshots(rpcs []string, snapshots []*receiptSnapshot) (*receiptSnapshot, SourceConsensus, error) {
	// Pick a canonical snapshot only after every source agrees on the normalized receipt view.
	base, err := requireMatchingSnapshots(rpcs, snapshots, compareReceiptSnapshot)
	if err != nil {
		return nil, SourceConsensus{}, err
	}
	// Persist both digests and readable fields so the proof records exactly what was held constant
	// across RPC sources.
	consensus, err := buildReceiptConsensus(base, rpcs)
	if err != nil {
		return nil, SourceConsensus{}, err
	}
	return base, consensus, nil
}

func compareReceiptSnapshot(base, other *receiptSnapshot) []string {
	var diffs []string
	diffs = append(diffs, compareHeader(base.Header, other.Header)...)
	if base.TxHash != other.TxHash {
		diffs = append(diffs, "txHash mismatch")
	}
	if base.TxIndex != other.TxIndex {
		diffs = append(diffs, "txIndex mismatch")
	}
	if base.LogIndex != other.LogIndex {
		diffs = append(diffs, "logIndex mismatch")
	}
	if !bytes.Equal(base.TransactionRLP, other.TransactionRLP) {
		diffs = append(diffs, "transactionRlp mismatch")
	}
	if !bytes.Equal(base.ReceiptRLP, other.ReceiptRLP) {
		diffs = append(diffs, "receiptRlp mismatch")
	}
	diffs = append(diffs, compareByteSlices("blockTransactions", base.BlockTransactions, other.BlockTransactions)...)
	diffs = append(diffs, compareByteSlices("blockReceipts", base.BlockReceipts, other.BlockReceipts)...)
	diffs = append(diffs, compareEvent(base.Event, other.Event)...)
	return diffs
}

func buildReceiptConsensus(base *receiptSnapshot, rpcs []string) (SourceConsensus, error) {
	headerDigest, err := proofutil.CanonicalDigest(base.Header)
	if err != nil {
		return SourceConsensus{}, err
	}
	blockTransactionsDigest, err := proofutil.CanonicalDigest(base.BlockTransactions)
	if err != nil {
		return SourceConsensus{}, err
	}
	blockReceiptsDigest, err := proofutil.CanonicalDigest(base.BlockReceipts)
	if err != nil {
		return SourceConsensus{}, err
	}
	targetReceiptDigest, err := proofutil.CanonicalDigest(struct {
		TransactionRLP hexutil.Bytes `json:"transactionRlp"`
		ReceiptRLP     hexutil.Bytes `json:"receiptRlp"`
		Event          EventClaim    `json:"event"`
	}{
		TransactionRLP: base.TransactionRLP,
		ReceiptRLP:     base.ReceiptRLP,
		Event:          base.Event,
	})
	if err != nil {
		return SourceConsensus{}, err
	}
	return sourceConsensus(
		"live-rpc",
		rpcs,
		[]ConsensusDigest{
			{Name: "header", Digest: headerDigest},
			{Name: "blockTransactions", Digest: blockTransactionsDigest},
			{Name: "blockReceipts", Digest: blockReceiptsDigest},
			{Name: "targetReceipt", Digest: targetReceiptDigest},
		},
		[]ConsensusField{
			{Name: "chainId", Value: proofutil.ChainIDString(base.Header.ChainID), Consistent: true},
			{Name: "blockNumber", Value: fmt.Sprintf("%d", base.Header.BlockNumber), Consistent: true},
			{Name: "blockHash", Value: base.Header.BlockHash.Hex(), Consistent: true},
			{Name: "parentHash", Value: base.Header.ParentHash.Hex(), Consistent: true},
			{Name: "stateRoot", Value: base.Header.StateRoot.Hex(), Consistent: true},
			{Name: "transactionsRoot", Value: base.Header.TransactionsRoot.Hex(), Consistent: true},
			{Name: "receiptsRoot", Value: base.Header.ReceiptsRoot.Hex(), Consistent: true},
			{Name: "txHash", Value: base.TxHash.Hex(), Consistent: true},
			{Name: "txIndex", Value: fmt.Sprintf("%d", base.TxIndex), Consistent: true},
			{Name: "logIndex", Value: fmt.Sprintf("%d", base.LogIndex), Consistent: true},
			{Name: "event.address", Value: base.Event.Address.Hex(), Consistent: true},
			{Name: "event.topics", Value: fmt.Sprintf("%v", base.Event.Topics), Consistent: true},
			{Name: "event.data", Value: hexutil.Encode(base.Event.Data), Consistent: true},
		},
	), nil
}
