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

type receiptSnapshotCollector struct {
	txHash   common.Hash
	logIndex uint64
}

// GenerateReceiptProof fetches the target receipt data from every RPC source, requires normalized
// agreement, rebuilds the receipts trie locally, and returns the inclusion proof package.
func GenerateReceiptProof(ctx context.Context, req ReceiptProofRequest) (*ReceiptProofPackage, error) {
	sourceSet, err := openNormalizedRPCSources(ctx, req.RPCURLs, req.MinRPCSources)
	if err != nil {
		return nil, err
	}
	defer sourceSet.Close()

	return GenerateReceiptProofFromSources(ctx, ReceiptProofSourcesRequest{
		Sources:       sourceSet.ReceiptSources(),
		MinRPCSources: req.MinRPCSources,
		TxHash:        req.TxHash,
		LogIndex:      req.LogIndex,
	})
}

// GenerateReceiptProofFromSources fetches the target receipt data from every source, requires
// normalized agreement, rebuilds the receipts trie locally, and returns the inclusion proof package.
func GenerateReceiptProofFromSources(ctx context.Context, req ReceiptProofSourcesRequest) (*ReceiptProofPackage, error) {
	sourceNames, err := normalizeSourceNames(req.Sources, req.MinRPCSources)
	if err != nil {
		return nil, err
	}
	snapshots, err := collectReceiptSnapshots(ctx, req)
	if err != nil {
		return nil, err
	}
	base, consensus, err := consensusForReceiptSnapshots(sourceNames, snapshots)
	if err != nil {
		return nil, err
	}

	// Rebuild the receipts trie locally from the agreed receipt bytes so the returned proof is
	// anchored to the same receiptsRoot that appears in the agreed block header.
	blockReceipts, err := decodeReceiptList(base.BlockReceipts)
	if err != nil {
		return nil, err
	}
	derivedRoot := types.DeriveSha(blockReceipts, trie.NewStackTrie(nil))
	if derivedRoot != base.Header.ReceiptsRoot {
		return nil, fmt.Errorf("derived receiptsRoot mismatch: local=%s expected=%s", derivedRoot, base.Header.ReceiptsRoot)
	}
	transactionRLP, transactionProofNodes, err := buildTransactionTrieAndProof(base.BlockTransactions, base.TxIndex, base.Header.TransactionsRoot)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(transactionRLP, base.TransactionRLP) {
		return nil, fmt.Errorf("transaction proof bytes do not match target transaction bytes")
	}
	receiptRLP, proofNodes, err := buildReceiptTrieAndProof(base.BlockReceipts, base.TxIndex, base.Header.ReceiptsRoot)
	if err != nil {
		return nil, err
	}
	pkg := &ReceiptProofPackage{
		Block:                 buildBlockContext(base.Header, consensus),
		TxHash:                base.TxHash,
		TxIndex:               base.TxIndex,
		LogIndex:              base.LogIndex,
		TransactionRLP:        transactionRLP,
		TransactionProofNodes: transactionProofNodes,
		ReceiptRLP:            receiptRLP,
		ProofNodes:            proofNodes,
		Event:                 base.Event,
	}
	if err := VerifyReceiptProofPackageAgainstEmbeddedRoots(pkg); err != nil {
		return nil, fmt.Errorf("verify generated receipt proof package: %w", err)
	}
	return pkg, nil
}

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
	if _, err := blockSnapshotHeaderFromContext(pkg.Block); err != nil {
		return err
	}
	if err := validateSourceConsensusMetadata(pkg.Block.SourceConsensus); err != nil {
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
	if diffs := compareHashSlices("event.topics", log.Topics, pkg.Event.Topics); len(diffs) > 0 {
		return fmt.Errorf("%s", diffs[0])
	}
	if expect != nil {
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
	}
	return nil
}

func decodeReceiptList(hexReceipts []hexutil.Bytes) (types.Receipts, error) {
	out := make(types.Receipts, len(hexReceipts))
	for i, receiptHex := range hexReceipts {
		receipt, _, err := proofutil.DecodeReceipt(receiptHex)
		if err != nil {
			return nil, fmt.Errorf("decode receipt %d: %w", i, err)
		}
		out[i] = receipt
	}
	return out, nil
}

func collectReceiptSnapshots(ctx context.Context, req ReceiptProofSourcesRequest) ([]*receiptSnapshot, error) {
	collector := receiptSnapshotCollector{
		txHash:   req.TxHash,
		logIndex: req.LogIndex,
	}
	return collectFromSources(ctx, req.Sources, collector.fetch)
}

func (c receiptSnapshotCollector) fetch(ctx context.Context, source ReceiptSource) (*receiptSnapshot, error) {
	return fetchReceiptSnapshot(ctx, source, c.txHash, c.logIndex)
}
