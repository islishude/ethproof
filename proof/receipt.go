package proof

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
)

func generateReceiptProof(ctx context.Context, req ReceiptProofRequest) (*ReceiptProofPackage, error) {
	rpcs, err := normalizeRPCURLs(req.RPCURLs, req.MinRPCSources)
	if err != nil {
		return nil, err
	}
	sources, err := openRPCSources(ctx, rpcs)
	if err != nil {
		return nil, err
	}
	defer closeRPCSources(sources)

	snapshots := make([]*receiptSnapshot, 0, len(sources))
	for _, source := range sources {
		snapshot, snapErr := fetchReceiptSnapshot(ctx, source, req.TxHash, req.LogIndex)
		if snapErr != nil {
			return nil, fmt.Errorf("%s: %w", source.url, snapErr)
		}
		snapshots = append(snapshots, snapshot)
	}
	base, consensus, err := consensusForReceiptSnapshots(rpcs, snapshots)
	if err != nil {
		return nil, err
	}
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
	return &ReceiptProofPackage{
		Block:          buildBlockContext(base.Header, consensus),
		TxHash:         base.TxHash,
		TxIndex:        base.TxIndex,
		LogIndex:       base.LogIndex,
		TransactionRLP: base.TransactionRLP,
		ReceiptRLP:     receiptRLP,
		ProofNodes:     proofNodes,
		Event:          base.Event,
	}, nil
}

func verifyReceiptProofPackage(pkg *ReceiptProofPackage, expect *ReceiptExpectations) error {
	proofDB, err := proofDBFromHexNodes(pkg.ProofNodes)
	if err != nil {
		return err
	}
	verifiedReceipt, err := trie.VerifyProof(pkg.Block.ReceiptsRoot, trieIndexKey(pkg.TxIndex), proofDB)
	if err != nil {
		return fmt.Errorf("verify receipt proof: %w", err)
	}
	receipt, claimedReceipt, err := decodeReceipt(pkg.ReceiptRLP)
	if err != nil {
		return fmt.Errorf("decode claimed receipt: %w", err)
	}
	if !bytes.Equal(verifiedReceipt, claimedReceipt) {
		return fmt.Errorf("verified receipt bytes do not match claimed receipt bytes")
	}
	tx, _, err := decodeTransaction(pkg.TransactionRLP)
	if err != nil {
		return fmt.Errorf("decode claimed transaction: %w", err)
	}
	if tx.Hash() != pkg.TxHash {
		return fmt.Errorf("transaction hash mismatch: got %s want %s", tx.Hash(), pkg.TxHash)
	}
	if int(pkg.LogIndex) >= len(receipt.Logs) {
		return fmt.Errorf("log index %d out of range for receipt with %d logs", pkg.LogIndex, len(receipt.Logs))
	}
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

func consensusForReceiptSnapshots(rpcs []string, snapshots []*receiptSnapshot) (*receiptSnapshot, SourceConsensus, error) {
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
		if base.LogIndex != other.LogIndex {
			diffs = append(diffs, "logIndex mismatch")
		}
		if !bytes.Equal(base.TransactionRLP, other.TransactionRLP) {
			diffs = append(diffs, "transactionRlp mismatch")
		}
		if !bytes.Equal(base.ReceiptRLP, other.ReceiptRLP) {
			diffs = append(diffs, "receiptRlp mismatch")
		}
		diffs = append(diffs, compareByteSlices("blockTransactions", base.BlockTransactions, other.BlockTransactions)...)
		diffs = append(diffs, compareByteSlices("blockReceipts", base.BlockReceipts, other.BlockReceipts)...)
		diffs = append(diffs, compareEvent(base.Event, other.Event)...)
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
	blockReceiptsDigest, err := canonicalDigest(base.BlockReceipts)
	if err != nil {
		return nil, SourceConsensus{}, err
	}
	targetReceiptDigest, err := canonicalDigest(struct {
		TransactionRLP hexutil.Bytes `json:"transactionRlp"`
		ReceiptRLP     hexutil.Bytes `json:"receiptRlp"`
		Event          EventClaim    `json:"event"`
	}{
		TransactionRLP: base.TransactionRLP,
		ReceiptRLP:     base.ReceiptRLP,
		Event:          base.Event,
	})
	if err != nil {
		return nil, SourceConsensus{}, err
	}
	consensus := sourceConsensus(
		"live-rpc",
		rpcs,
		[]ConsensusDigest{
			{Name: "header", Digest: headerDigest},
			{Name: "blockTransactions", Digest: blockTransactionsDigest},
			{Name: "blockReceipts", Digest: blockReceiptsDigest},
			{Name: "targetReceipt", Digest: targetReceiptDigest},
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
			{Name: "logIndex", Value: fmt.Sprintf("%d", base.LogIndex), Consistent: true},
			{Name: "event.address", Value: base.Event.Address.Hex(), Consistent: true},
			{Name: "event.topics", Value: fmt.Sprintf("%v", base.Event.Topics), Consistent: true},
			{Name: "event.data", Value: hexutil.Encode(base.Event.Data), Consistent: true},
		},
	)
	return base, consensus, nil
}

func decodeReceiptList(hexReceipts []hexutil.Bytes) (types.Receipts, error) {
	out := make(types.Receipts, len(hexReceipts))
	for i, receiptHex := range hexReceipts {
		receipt, _, err := decodeReceipt(receiptHex)
		if err != nil {
			return nil, fmt.Errorf("decode receipt %d: %w", i, err)
		}
		out[i] = receipt
	}
	return out, nil
}
