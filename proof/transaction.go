package proof

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/islishude/ethproof/internal/proofutil"
)

type transactionSnapshotCollector struct {
	txHash common.Hash
}

// GenerateTransactionProof fetches the target transaction data from every RPC source, requires
// normalized agreement, rebuilds the transactions trie locally, and returns the inclusion proof package.
func GenerateTransactionProof(ctx context.Context, req TransactionProofRequest) (*TransactionProofPackage, error) {
	sourceSet, err := openNormalizedRPCSources(ctx, req.RPCURLs, req.MinRPCSources)
	if err != nil {
		return nil, err
	}
	defer sourceSet.Close()

	return GenerateTransactionProofFromSources(ctx, TransactionProofSourcesRequest{
		Sources:       sourceSet.TransactionSources(),
		MinRPCSources: req.MinRPCSources,
		TxHash:        req.TxHash,
	})
}

// GenerateTransactionProofFromSources fetches the target transaction data from every source,
// requires normalized agreement, rebuilds the transactions trie locally, and returns the inclusion proof package.
func GenerateTransactionProofFromSources(ctx context.Context, req TransactionProofSourcesRequest) (*TransactionProofPackage, error) {
	sourceNames, err := normalizeSourceNames(req.Sources, req.MinRPCSources)
	if err != nil {
		return nil, err
	}
	snapshots, err := collectTransactionSnapshots(ctx, req)
	if err != nil {
		return nil, err
	}
	base, consensus, err := consensusForTransactionSnapshots(sourceNames, snapshots)
	if err != nil {
		return nil, err
	}

	// Rebuild the transactions trie locally from the agreed transaction bytes so the proof we
	// return is anchored to the transactionsRoot in the agreed block header.
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
	pkg := &TransactionProofPackage{
		Block:          buildBlockContext(base.Header, consensus),
		TxHash:         base.TxHash,
		TxIndex:        base.TxIndex,
		TransactionRLP: transactionRLP,
		ProofNodes:     proofNodes,
	}
	if err := VerifyTransactionProofPackageAgainstEmbeddedRoots(pkg); err != nil {
		return nil, fmt.Errorf("verify generated transaction proof package: %w", err)
	}
	return pkg, nil
}

// VerifyTransactionProofPackage verifies internal consistency against the roots embedded in pkg.
//
// Deprecated: use VerifyTransactionProofPackageAgainstBlockAnchor when a caller-trusted block
// anchor is available, or VerifyTransactionProofPackageAgainstEmbeddedRoots when only structural
// verification is intended.
func VerifyTransactionProofPackage(pkg *TransactionProofPackage) error {
	return VerifyTransactionProofPackageAgainstEmbeddedRoots(pkg)
}

// VerifyTransactionProofPackageAgainstEmbeddedRoots verifies transaction inclusion against the
// transactions root carried by pkg. It does not authenticate that root as belonging to a trusted block.
func VerifyTransactionProofPackageAgainstEmbeddedRoots(pkg *TransactionProofPackage) error {
	if pkg == nil {
		return fmt.Errorf("transaction proof package is nil")
	}
	if _, err := blockSnapshotHeaderFromContext(pkg.Block); err != nil {
		return err
	}
	if err := validateSourceConsensusMetadata(pkg.Block.SourceConsensus); err != nil {
		return err
	}
	_, err := verifyTransactionInclusion(pkg.Block.TransactionsRoot, pkg.TxIndex, pkg.TransactionRLP, pkg.ProofNodes, pkg.TxHash)
	return err
}

// VerifyTransactionProofPackageAgainstBlockAnchor verifies transaction inclusion and requires every
// embedded block field to match a caller-trusted block anchor.
func VerifyTransactionProofPackageAgainstBlockAnchor(pkg *TransactionProofPackage, anchor BlockAnchor) error {
	if err := VerifyTransactionProofPackageAgainstEmbeddedRoots(pkg); err != nil {
		return err
	}
	return verifyBlockContextAgainstAnchor(pkg.Block, anchor)
}

func verifyTransactionInclusion(
	transactionsRoot common.Hash,
	txIndex uint64,
	transactionRLP []byte,
	proofNodes []hexutil.Bytes,
	txHash common.Hash,
) (*types.Transaction, error) {
	if len(proofNodes) == 0 {
		return nil, fmt.Errorf("transaction proof package must contain transaction proof nodes")
	}
	// Verify inclusion first using the supplied proof nodes and transactionsRoot.
	proofDB, err := proofutil.ProofDBFromHexNodes(proofNodes)
	if err != nil {
		return nil, err
	}
	verifiedTransaction, err := trie.VerifyProof(transactionsRoot, proofutil.TrieIndexKey(txIndex), proofDB)
	if err != nil {
		return nil, fmt.Errorf("verify transaction proof: %w", err)
	}
	tx, claimedTransaction, err := proofutil.DecodeTransaction(transactionRLP)
	if err != nil {
		return nil, fmt.Errorf("decode claimed transaction: %w", err)
	}

	// The proof must reproduce the exact canonical transaction bytes stored in the package.
	if !bytes.Equal(verifiedTransaction, claimedTransaction) {
		return nil, fmt.Errorf("verified transaction bytes do not match claimed transaction bytes")
	}

	// Finally confirm that the claimed bytes actually hash to the advertised transaction hash.
	if tx.Hash() != txHash {
		return nil, fmt.Errorf("transaction hash mismatch: got %s want %s", tx.Hash(), txHash)
	}
	return tx, nil
}

func decodeTransactionList(hexTransactions []hexutil.Bytes) (types.Transactions, error) {
	out := make(types.Transactions, len(hexTransactions))
	for i, txHex := range hexTransactions {
		tx, _, err := proofutil.DecodeTransaction(txHex)
		if err != nil {
			return nil, fmt.Errorf("decode transaction %d: %w", i, err)
		}
		out[i] = tx
	}
	return out, nil
}

func collectTransactionSnapshots(ctx context.Context, req TransactionProofSourcesRequest) ([]*transactionSnapshot, error) {
	collector := transactionSnapshotCollector{txHash: req.TxHash}
	return collectFromSources(ctx, req.Sources, collector.fetch)
}

func (c transactionSnapshotCollector) fetch(ctx context.Context, source TransactionSource) (*transactionSnapshot, error) {
	return fetchTransactionSnapshot(ctx, source, c.txHash)
}
