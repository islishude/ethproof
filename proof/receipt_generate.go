package proof

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/islishude/ethproof/internal/proofutil"
)

// GenerateReceiptProof builds transaction and receipt inclusion proofs from one caller-owned source.
func GenerateReceiptProof(ctx context.Context, source ReceiptSource, req ReceiptProofRequest) (*ReceiptProofPackage, error) {
	if err := validateSource(ctx, source); err != nil {
		return nil, err
	}
	txHash, logIndex := req.TxHash, req.LogIndex
	data, err := fetchTransactionData(ctx, source, txHash)
	if err != nil {
		return nil, err
	}
	receipt := data.receipt
	if logIndex >= uint64(len(receipt.Logs)) {
		return nil, fmt.Errorf("log-index %d out of range (receipt has %d logs)", logIndex, len(receipt.Logs))
	}
	log := receipt.Logs[logIndex]
	if log == nil {
		return nil, fmt.Errorf("target receipt log %d is nil", logIndex)
	}
	if log.Removed {
		return nil, fmt.Errorf("target receipt log %d is marked removed", logIndex)
	}

	// Encode the target receipt and then require it to match the same bytes recovered from the
	// full block receipt list. With block-receipts RPC this cross-checks the point
	// lookup; fallback reuses that receipt and later checks the reconstructed trie root.
	receiptRLP, err := proofutil.EncodeReceipt(receipt)
	if err != nil {
		return nil, fmt.Errorf("encode target receipt: %w", err)
	}
	blockReceipts, err := fetchBlockReceipts(ctx, source, data.block.BlockHash, data.transactions, receipt)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(blockReceipts[uint64(receipt.TransactionIndex)], receiptRLP) {
		return nil, fmt.Errorf("receipt bytes mismatch between block receipts and target receipt lookup")
	}

	txProof, err := data.prove(txHash)
	if err != nil {
		return nil, err
	}
	encodedReceipt, nodes, err := buildReceiptTrieAndProof(blockReceipts, txProof.TxIndex, data.block.ReceiptsRoot)
	if err != nil {
		return nil, err
	}
	pkg := &ReceiptProofPackage{
		Block:                 data.block,
		TxHash:                txHash,
		TxIndex:               txProof.TxIndex,
		LogIndex:              logIndex,
		TransactionRLP:        txProof.TransactionRLP,
		TransactionProofNodes: txProof.ProofNodes,
		ReceiptRLP:            encodedReceipt,
		ProofNodes:            nodes,
		Event:                 EventClaim{Address: log.Address, Topics: append([]common.Hash(nil), log.Topics...), Data: proofutil.CanonicalBytes(log.Data)},
	}
	if err := VerifyReceiptProofPackageAgainstEmbeddedRoots(pkg); err != nil {
		return nil, fmt.Errorf("verify generated receipt proof package: %w", err)
	}
	return pkg, nil
}
