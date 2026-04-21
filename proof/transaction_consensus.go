package proof

import (
	"bytes"
	"fmt"

	"github.com/islishude/ethproof/internal/proofutil"
)

func consensusForTransactionSnapshots(rpcs []string, snapshots []*transactionSnapshot) (*transactionSnapshot, SourceConsensus, error) {
	// Pick a canonical snapshot only after every source agrees on the normalized transaction view.
	base, err := requireMatchingSnapshots(rpcs, snapshots, compareTransactionSnapshot)
	if err != nil {
		return nil, SourceConsensus{}, err
	}
	// Persist both digests and readable fields so the proof records exactly what was held constant
	// across RPC sources.
	consensus, err := buildTransactionConsensus(base, rpcs)
	if err != nil {
		return nil, SourceConsensus{}, err
	}
	return base, consensus, nil
}

func compareTransactionSnapshot(base, other *transactionSnapshot) []string {
	var diffs []string
	diffs = append(diffs, compareHeader(base.Header, other.Header)...)
	if base.TxHash != other.TxHash {
		diffs = append(diffs, "txHash mismatch")
	}
	if base.TxIndex != other.TxIndex {
		diffs = append(diffs, "txIndex mismatch")
	}
	if !bytes.Equal(base.TransactionRLP, other.TransactionRLP) {
		diffs = append(diffs, "transactionRlp mismatch")
	}
	diffs = append(diffs, compareByteSlices("blockTransactions", base.BlockTransactions, other.BlockTransactions)...)
	return diffs
}

func buildTransactionConsensus(base *transactionSnapshot, rpcs []string) (SourceConsensus, error) {
	headerDigest, err := proofutil.CanonicalDigest(base.Header)
	if err != nil {
		return SourceConsensus{}, err
	}
	blockTransactionsDigest, err := proofutil.CanonicalDigest(base.BlockTransactions)
	if err != nil {
		return SourceConsensus{}, err
	}
	targetTransactionDigest, err := proofutil.CanonicalDigest(base.TransactionRLP)
	if err != nil {
		return SourceConsensus{}, err
	}
	return sourceConsensus(
		"live-rpc",
		rpcs,
		[]ConsensusDigest{
			{Name: "header", Digest: headerDigest},
			{Name: "blockTransactions", Digest: blockTransactionsDigest},
			{Name: "targetTransaction", Digest: targetTransactionDigest},
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
		},
	), nil
}
