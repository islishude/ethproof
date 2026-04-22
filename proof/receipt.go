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
	logIndex uint
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
	receiptRLP, proofNodes, err := buildReceiptTrieAndProof(base.BlockReceipts, base.TxIndex, base.Header.ReceiptsRoot)
	if err != nil {
		return nil, err
	}
	pkg := &ReceiptProofPackage{
		Block:          buildBlockContext(base.Header, consensus),
		TxHash:         base.TxHash,
		TxIndex:        base.TxIndex,
		LogIndex:       base.LogIndex,
		TransactionRLP: base.TransactionRLP,
		ReceiptRLP:     receiptRLP,
		ProofNodes:     proofNodes,
		Event:          base.Event,
	}
	return pkg, nil
}

// VerifyReceiptProofPackage verifies the embedded receipt proof without extra caller expectations.
func VerifyReceiptProofPackage(pkg *ReceiptProofPackage) error {
	return verifyReceiptProofPackageLocal(pkg, nil)
}

// VerifyReceiptProofPackageWithExpectations verifies the receipt proof locally and optionally checks
// additional caller-provided expectations against the claimed log.
func VerifyReceiptProofPackageWithExpectations(pkg *ReceiptProofPackage, expect *ReceiptExpectations) error {
	return verifyReceiptProofPackageLocal(pkg, expect)
}

func verifyReceiptProofPackageLocal(pkg *ReceiptProofPackage, expect *ReceiptExpectations) error {
	// Verify inclusion first using the provided proof nodes and the package's receiptsRoot.
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

	// The proof must reproduce the exact claimed receipt bytes, not merely a receipt that
	// decodes to the same high-level fields.
	if !bytes.Equal(verifiedReceipt, claimedReceipt) {
		return fmt.Errorf("verified receipt bytes do not match claimed receipt bytes")
	}

	// Cross-check the claimed transaction bytes because the proof package stores both the
	// receipt inclusion witness and the transaction identity it is supposed to belong to.
	tx, _, err := proofutil.DecodeTransaction(pkg.TransactionRLP)
	if err != nil {
		return fmt.Errorf("decode claimed transaction: %w", err)
	}
	if tx.Hash() != pkg.TxHash {
		return fmt.Errorf("transaction hash mismatch: got %s want %s", tx.Hash(), pkg.TxHash)
	}
	if int(pkg.LogIndex) >= len(receipt.Logs) {
		return fmt.Errorf("log index %d out of range for receipt with %d logs", pkg.LogIndex, len(receipt.Logs))
	}

	// After inclusion is established, validate the claimed event payload at the target log index.
	log := receipt.Logs[pkg.LogIndex]
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
		// Caller expectations are additive checks on top of the package's own claims.
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
