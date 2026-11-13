package proof

import (
	"bytes"
	"fmt"
	"slices"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/islishude/ethproof/internal/proofutil"
)

// VerifyReceiptProofPackage verifies internal consistency against the roots embedded in pkg.
//
// Deprecated: use VerifyReceiptProofPackageAgainstBlockAnchor when a caller-trusted block anchor is
// available, or VerifyReceiptProofPackageAgainstEmbeddedRoots when only structural verification is intended.
func VerifyReceiptProofPackage(pkg *ReceiptProofPackage) error {
	return VerifyReceiptProofPackageAgainstEmbeddedRoots(pkg)
}

// VerifyReceiptProofPackageWithExpectations verifies internal consistency and optional expectations
// against roots embedded in pkg.
//
// Deprecated: use VerifyReceiptProofPackageWithExpectationsAgainstBlockAnchor when a caller-trusted
// block anchor is available, or VerifyReceiptProofPackageWithExpectationsAgainstEmbeddedRoots when
// only structural verification is intended.
func VerifyReceiptProofPackageWithExpectations(pkg *ReceiptProofPackage, expect *ReceiptExpectations) error {
	return VerifyReceiptProofPackageWithExpectationsAgainstEmbeddedRoots(pkg, expect)
}

// VerifyReceiptProofPackageAgainstEmbeddedRoots verifies receipt and transaction inclusion against
// roots carried by pkg without authenticating those roots as belonging to a trusted block.
func VerifyReceiptProofPackageAgainstEmbeddedRoots(pkg *ReceiptProofPackage) error {
	return VerifyReceiptProofPackageWithExpectationsAgainstEmbeddedRoots(pkg, nil)
}

// VerifyReceiptProofPackageWithExpectationsAgainstEmbeddedRoots verifies receipt and transaction
// inclusion plus optional event expectations against roots carried by pkg.
func VerifyReceiptProofPackageWithExpectationsAgainstEmbeddedRoots(pkg *ReceiptProofPackage, expect *ReceiptExpectations) error {
	return verifyReceiptProofPackageLocal(pkg, expect)
}

// VerifyReceiptProofPackageAgainstBlockAnchor verifies the receipt package and requires every
// embedded block field to match a caller-trusted block anchor.
func VerifyReceiptProofPackageAgainstBlockAnchor(pkg *ReceiptProofPackage, anchor BlockAnchor) error {
	return VerifyReceiptProofPackageWithExpectationsAgainstBlockAnchor(pkg, nil, anchor)
}

// VerifyReceiptProofPackageWithExpectationsAgainstBlockAnchor verifies the receipt package,
// optional event expectations, and every block field against a caller-trusted block anchor.
func VerifyReceiptProofPackageWithExpectationsAgainstBlockAnchor(pkg *ReceiptProofPackage, expect *ReceiptExpectations, anchor BlockAnchor) error {
	if err := VerifyReceiptProofPackageWithExpectationsAgainstEmbeddedRoots(pkg, expect); err != nil {
		return err
	}
	return verifyBlockContextAgainstAnchor(pkg.Block, anchor)
}

func verifyReceiptProofPackageLocal(pkg *ReceiptProofPackage, expect *ReceiptExpectations) error {
	if pkg == nil {
		return fmt.Errorf("receipt proof package is nil")
	}
	if err := validateBlockContext(pkg.Block); err != nil {
		return err
	}
	if len(pkg.ProofNodes) == 0 {
		return fmt.Errorf("receipt proof package must contain receipt proof nodes")
	}
	if len(pkg.TransactionProofNodes) == 0 {
		return fmt.Errorf("receipt proof package must contain transaction proof nodes")
	}

	// Verify receipt inclusion using receiptsRoot and TxIndex.
	proofDB, err := proofutil.ProofDBFromHexNodes(pkg.ProofNodes)
	if err != nil {
		return err
	}
	verifiedReceipt, err := trie.VerifyProof(pkg.Block.ReceiptsRoot, proofutil.TrieIndexKey(pkg.TxIndex), proofDB)
	if err != nil {
		return fmt.Errorf("verify receipt proof: %w", err)
	}
	receipt, claimedReceipt, err := proofutil.DecodeReceipt(pkg.ReceiptRLP)
	if err != nil {
		return fmt.Errorf("decode claimed receipt: %w", err)
	}
	if !bytes.Equal(verifiedReceipt, claimedReceipt) {
		return fmt.Errorf("verified receipt bytes do not match claimed receipt bytes")
	}

	// Prove that the claimed transaction is stored at the same index under transactionsRoot.
	tx, err := verifyTransactionInclusion(pkg.Block.TransactionsRoot, pkg.TxIndex, pkg.TransactionRLP, pkg.TransactionProofNodes, pkg.TxHash)
	if err != nil {
		return err
	}
	if receipt.Type != tx.Type() {
		return fmt.Errorf("receipt type mismatch: got %d want transaction type %d", receipt.Type, tx.Type())
	}
	if pkg.LogIndex >= uint64(len(receipt.Logs)) {
		return fmt.Errorf("log index %d out of range for receipt with %d logs", pkg.LogIndex, len(receipt.Logs))
	}

	log := receipt.Logs[pkg.LogIndex]
	if log == nil {
		return fmt.Errorf("receipt log %d is nil", pkg.LogIndex)
	}
	if log.Address != pkg.Event.Address {
		return fmt.Errorf("event address mismatch: got %s want %s", log.Address, pkg.Event.Address)
	}
	if !bytes.Equal(log.Data, pkg.Event.Data) {
		return fmt.Errorf("event data mismatch")
	}
	if !slices.Equal(log.Topics, pkg.Event.Topics) {
		return fmt.Errorf("event topics mismatch")
	}
	return verifyReceiptExpectations(log, expect)
}

func verifyReceiptExpectations(log *types.Log, expect *ReceiptExpectations) error {
	if expect == nil {
		return nil
	}
	if expect.Emitter != nil && log.Address != *expect.Emitter {
		return fmt.Errorf("expected emitter mismatch: got %s want %s", log.Address, *expect.Emitter)
	}
	if len(expect.Topics) > 0 {
		if len(log.Topics) < len(expect.Topics) {
			return fmt.Errorf("expected topic count mismatch: got %d want at least %d", len(log.Topics), len(expect.Topics))
		}
		for i := range expect.Topics {
			if log.Topics[i] != expect.Topics[i] {
				return fmt.Errorf("expected topic[%d] mismatch: got %s want %s", i, log.Topics[i], expect.Topics[i])
			}
		}
	}
	if expect.Data != nil && !bytes.Equal(log.Data, expect.Data) {
		return fmt.Errorf("expected data mismatch")
	}
	return nil
}
