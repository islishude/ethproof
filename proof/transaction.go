package proof

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/islishude/ethproof/internal/proofutil"
)

// GenerateTransactionProof fetches the target transaction data from every RPC source, requires
// normalized agreement, rebuilds the transactions trie locally, and returns the inclusion proof package.
func GenerateTransactionProof(ctx context.Context, req TransactionProofRequest) (*TransactionProofPackage, error) {
	logger := loggerFromContext(ctx).With("proof_type", "transaction")
	logger.Info("generate proof started", "tx_hash", req.TxHash)

	// Normalize the RPC set up front so consensus is evaluated over a stable source list.
	rpcs, err := normalizeRPCURLs(req.RPCURLs, req.MinRPCSources)
	if err != nil {
		return nil, err
	}
	logger.Debug("normalized rpc sources", "rpc_count", len(rpcs))

	// Each source yields a normalized transaction snapshot that includes both the target
	// transaction bytes and the full block transaction list needed to rebuild the trie locally.
	snapshots, err := collectFromRPCs(ctx, rpcs, func(ctx context.Context, source *rpcSource) (*transactionSnapshot, error) {
		return fetchTransactionSnapshot(ctx, source, req.TxHash)
	})
	if err != nil {
		return nil, err
	}
	logger.Debug("fetched transaction snapshots", "snapshot_count", len(snapshots))

	// Transaction proof generation is strict: every normalized field must match across sources.
	base, consensus, err := consensusForTransactionSnapshots(rpcs, snapshots)
	if err != nil {
		return nil, err
	}
	logger.Info("rpc consensus established", "rpc_count", len(rpcs), "block_hash", base.Header.BlockHash)

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
	logger.Debug("rebuilt transaction trie locally", "tx_index", base.TxIndex, "transaction_count", len(base.BlockTransactions))
	pkg := &TransactionProofPackage{
		Block:          buildBlockContext(base.Header, consensus),
		TxHash:         base.TxHash,
		TxIndex:        base.TxIndex,
		TransactionRLP: transactionRLP,
		ProofNodes:     proofNodes,
	}
	logger.Info("generate proof completed", "block_number", pkg.Block.BlockNumber, "transactions_root", pkg.Block.TransactionsRoot)
	return pkg, nil
}

// VerifyTransactionProofPackage verifies the embedded transaction proof locally.
func VerifyTransactionProofPackage(pkg *TransactionProofPackage) error {
	return verifyTransactionProofPackageWithLogger(discardLogger, pkg)
}

func verifyTransactionProofPackageWithLogger(logger *slog.Logger, pkg *TransactionProofPackage) error {
	// Verify inclusion first using the supplied proof nodes and transactionsRoot.
	logger.Debug("verifying local transaction proof", "block_hash", pkg.Block.BlockHash, "tx_hash", pkg.TxHash)
	proofDB, err := proofutil.ProofDBFromHexNodes(pkg.ProofNodes)
	if err != nil {
		return err
	}
	verifiedTransaction, err := trie.VerifyProof(pkg.Block.TransactionsRoot, proofutil.TrieIndexKey(pkg.TxIndex), proofDB)
	if err != nil {
		return fmt.Errorf("verify transaction proof: %w", err)
	}
	tx, claimedTransaction, err := proofutil.DecodeTransaction(pkg.TransactionRLP)
	if err != nil {
		return fmt.Errorf("decode claimed transaction: %w", err)
	}

	// The proof must reproduce the exact canonical transaction bytes stored in the package.
	if !bytes.Equal(verifiedTransaction, claimedTransaction) {
		return fmt.Errorf("verified transaction bytes do not match claimed transaction bytes")
	}

	// Finally confirm that the claimed bytes actually hash to the advertised transaction hash.
	if tx.Hash() != pkg.TxHash {
		return fmt.Errorf("transaction hash mismatch: got %s want %s", tx.Hash(), pkg.TxHash)
	}
	logger.Debug("local transaction proof verified", "block_hash", pkg.Block.BlockHash, "tx_hash", pkg.TxHash)
	return nil
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
