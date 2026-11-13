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

// GenerateTransactionProof builds a transaction proof from one caller-owned source.
func GenerateTransactionProof(ctx context.Context, source TransactionSource, req TransactionProofRequest) (*TransactionProofPackage, error) {
	if err := validateSource(ctx, source); err != nil {
		return nil, err
	}
	data, err := fetchTransactionData(ctx, source, req.TxHash)
	if err != nil {
		return nil, err
	}
	pkg, err := data.prove(req.TxHash)
	if err != nil {
		return nil, err
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
	if err := validateBlockContext(pkg.Block); err != nil {
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
