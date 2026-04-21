package proof

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
)

func generateTransactionProof(ctx context.Context, req TransactionProofRequest) (*TransactionProofPackage, error) {
	rpcs, err := normalizeRPCURLs(req.RPCURLs, req.MinRPCSources)
	if err != nil {
		return nil, err
	}
	sources, err := openRPCSources(ctx, rpcs)
	if err != nil {
		return nil, err
	}
	defer closeRPCSources(sources)

	snapshots := make([]*transactionSnapshot, 0, len(sources))
	for _, source := range sources {
		snapshot, snapErr := fetchTransactionSnapshot(ctx, source, req.TxHash)
		if snapErr != nil {
			return nil, fmt.Errorf("%s: %w", source.url, snapErr)
		}
		snapshots = append(snapshots, snapshot)
	}

	base, consensus, err := consensusForTransactionSnapshots(rpcs, snapshots)
	if err != nil {
		return nil, err
	}
	blockTransactions, err := decodeTransactionList(base.BlockTransactions)
	if err != nil {
		return nil, err
	}
	derivedRoot := types.DeriveSha(blockTransactions, trie.NewStackTrie(nil))
	if derivedRoot != base.Header.TransactionsRoot {
		return nil, fmt.Errorf("derived transactionsRoot mismatch: local=%s expected=%s", derivedRoot, base.Header.TransactionsRoot)
	}
	transactionRLP, proofNodes, err := buildTransactionTrieAndProof(base.BlockTransactions, base.TxIndex, base.Header.TransactionsRoot)
	if err != nil {
		return nil, err
	}
	return &TransactionProofPackage{
		Block:          buildBlockContext(base.Header, consensus),
		TxHash:         base.TxHash,
		TxIndex:        base.TxIndex,
		TransactionRLP: transactionRLP,
		ProofNodes:     proofNodes,
	}, nil
}

func verifyTransactionProofPackage(pkg *TransactionProofPackage) error {
	proofDB, err := proofDBFromHexNodes(pkg.ProofNodes)
	if err != nil {
		return err
	}
	verifiedTransaction, err := trie.VerifyProof(pkg.Block.TransactionsRoot, trieIndexKey(pkg.TxIndex), proofDB)
	if err != nil {
		return fmt.Errorf("verify transaction proof: %w", err)
	}
	tx, claimedTransaction, err := decodeTransaction(pkg.TransactionRLP)
	if err != nil {
		return fmt.Errorf("decode claimed transaction: %w", err)
	}
	if !bytes.Equal(verifiedTransaction, claimedTransaction) {
		return fmt.Errorf("verified transaction bytes do not match claimed transaction bytes")
	}
	if tx.Hash() != pkg.TxHash {
		return fmt.Errorf("transaction hash mismatch: got %s want %s", tx.Hash(), pkg.TxHash)
	}
	return nil
}

func consensusForTransactionSnapshots(rpcs []string, snapshots []*transactionSnapshot) (*transactionSnapshot, SourceConsensus, error) {
	base := snapshots[0]
	for i := 1; i < len(snapshots); i++ {
		other := snapshots[i]
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
		if err := combineMismatch(rpcs[0], rpcs[i], diffs); err != nil {
			return nil, SourceConsensus{}, err
		}
	}
	headerDigest, err := canonicalDigest(base.Header)
	if err != nil {
		return nil, SourceConsensus{}, err
	}
	blockTransactionsDigest, err := canonicalDigest(base.BlockTransactions)
	if err != nil {
		return nil, SourceConsensus{}, err
	}
	targetTransactionDigest, err := canonicalDigest(base.TransactionRLP)
	if err != nil {
		return nil, SourceConsensus{}, err
	}
	consensus := sourceConsensus(
		"live-rpc",
		rpcs,
		[]ConsensusDigest{
			{Name: "header", Digest: headerDigest},
			{Name: "blockTransactions", Digest: blockTransactionsDigest},
			{Name: "targetTransaction", Digest: targetTransactionDigest},
		},
		[]ConsensusField{
			{Name: "chainId", Value: chainIDString(base.Header.ChainID), Consistent: true},
			{Name: "blockNumber", Value: fmt.Sprintf("%d", base.Header.BlockNumber), Consistent: true},
			{Name: "blockHash", Value: base.Header.BlockHash.Hex(), Consistent: true},
			{Name: "parentHash", Value: base.Header.ParentHash.Hex(), Consistent: true},
			{Name: "stateRoot", Value: base.Header.StateRoot.Hex(), Consistent: true},
			{Name: "transactionsRoot", Value: base.Header.TransactionsRoot.Hex(), Consistent: true},
			{Name: "receiptsRoot", Value: base.Header.ReceiptsRoot.Hex(), Consistent: true},
			{Name: "txHash", Value: base.TxHash.Hex(), Consistent: true},
			{Name: "txIndex", Value: fmt.Sprintf("%d", base.TxIndex), Consistent: true},
		},
	)
	return base, consensus, nil
}

func decodeTransactionList(hexTransactions []hexutil.Bytes) (types.Transactions, error) {
	out := make(types.Transactions, len(hexTransactions))
	for i, txHex := range hexTransactions {
		tx, _, err := decodeTransaction(txHex)
		if err != nil {
			return nil, fmt.Errorf("decode transaction %d: %w", i, err)
		}
		out[i] = tx
	}
	return out, nil
}
