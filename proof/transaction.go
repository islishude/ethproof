package proof

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/islishude/ethproof/internal/proofutil"
)

// GenerateTransactionProof fetches the target transaction data from every RPC source, requires
// normalized agreement, rebuilds the transactions trie locally, and returns the inclusion proof package.
func GenerateTransactionProof(ctx context.Context, req TransactionProofRequest) (*TransactionProofPackage, error) {
	// Normalize the RPC set up front so consensus is evaluated over a stable source list.
	rpcs, err := normalizeRPCURLs(req.RPCURLs, req.MinRPCSources)
	if err != nil {
		return nil, err
	}

	// Each source yields a normalized transaction snapshot that includes both the target
	// transaction bytes and the full block transaction list needed to rebuild the trie locally.
	snapshots, err := collectFromRPCs(ctx, rpcs, func(ctx context.Context, source *rpcSource) (*transactionSnapshot, error) {
		return fetchTransactionSnapshot(ctx, source, req.TxHash)
	})
	if err != nil {
		return nil, err
	}

	// Transaction proof generation is strict: every normalized field must match across sources.
	base, consensus, err := consensusForTransactionSnapshots(rpcs, snapshots)
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
	return &TransactionProofPackage{
		Block:          buildBlockContext(base.Header, consensus),
		TxHash:         base.TxHash,
		TxIndex:        base.TxIndex,
		TransactionRLP: transactionRLP,
		ProofNodes:     proofNodes,
	}, nil
}

// VerifyTransactionProofPackage verifies the embedded transaction proof locally.
func VerifyTransactionProofPackage(pkg *TransactionProofPackage) error {
	// Verify inclusion first using the supplied proof nodes and transactionsRoot.
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
