package proof

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/islishude/ethproof/internal/proofutil"
)

func fetchReceiptSnapshot(ctx context.Context, source ReceiptSource, txHash common.Hash, logIndex uint64) (*receiptSnapshot, error) {
	// Reuse the normalized transaction snapshot so receipt proof generation inherits the exact
	// transaction bytes and block context that transaction proof generation would see.
	data, err := fetchTransactionData(ctx, source, txHash)
	if err != nil {
		return nil, err
	}
	txSnapshot, receipt := data.snapshot, data.receipt
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
	blockReceipts, err := fetchBlockReceipts(ctx, source, txSnapshot.Header.BlockHash, data.transactions, receipt)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(blockReceipts[txSnapshot.TxIndex], receiptRLP) {
		return nil, fmt.Errorf("receipt bytes mismatch between block receipts and target receipt lookup")
	}

	// Persist the claimed event as simple address/topics/data fields so package verification does
	// not depend on any geth-specific receipt representation.
	return &receiptSnapshot{
		Header:            txSnapshot.Header,
		TxHash:            txHash,
		TxIndex:           txSnapshot.TxIndex,
		LogIndex:          logIndex,
		TransactionRLP:    txSnapshot.TransactionRLP,
		ReceiptRLP:        receiptRLP,
		BlockTransactions: txSnapshot.BlockTransactions,
		BlockReceipts:     blockReceipts,
		Event: EventClaim{
			Address: log.Address,
			Topics:  append([]common.Hash(nil), log.Topics...),
			Data:    proofutil.CanonicalBytes(log.Data),
		},
	}, nil
}

func fetchBlockReceipts(ctx context.Context, source ReceiptSource, blockHash common.Hash, transactions types.Transactions, target *types.Receipt) ([]hexutil.Bytes, error) {
	receipts, err := source.BlockReceiptsByHash(ctx, blockHash)
	if err != nil {
		// Some providers do not implement eth_getBlockReceipts. Fall back to scanning each tx so
		// receipt proof generation still works while remaining strict about normalized results.
		if isRPCMethodNotFound(err) {
			return fetchBlockReceiptsByTransactionScan(ctx, source, blockHash, transactions, target)
		}
		return nil, fmt.Errorf("fetch block receipts: %w", err)
	}
	if len(receipts) != len(transactions) {
		return nil, fmt.Errorf("block receipt count %d does not match expected count %d", len(receipts), len(transactions))
	}
	for i, receipt := range receipts {
		if receipt == nil || transactions[i] == nil {
			return nil, fmt.Errorf("nil receipt or transaction at index %d", i)
		}
		if receipt.TxHash != transactions[i].Hash() {
			return nil, fmt.Errorf("receipt %d tx hash mismatch", i)
		}
	}
	return encodeAndValidateBlockReceipts(receipts, blockHash, len(transactions))
}

// Bound concurrent receipt requests per source, preserving transaction order in
// the result. The shared target receipt and block body were already fetched.
const receiptScanConcurrency = 8

func fetchBlockReceiptsByTransactionScan(ctx context.Context, source ReceiptSource, blockHash common.Hash, transactions types.Transactions, target *types.Receipt) ([]hexutil.Bytes, error) {
	scanCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	receipts := make([]*types.Receipt, len(transactions))
	var next atomic.Uint64
	var workers sync.WaitGroup
	for range min(receiptScanConcurrency, len(transactions)) {
		workers.Go(func() {
			for {
				if scanCtx.Err() != nil {
					return
				}
				index := next.Add(1) - 1
				if index >= uint64(len(transactions)) {
					return
				}
				receipt, err := fetchScannedReceipt(scanCtx, source, blockHash, transactions, target, index)
				if err != nil {
					cancel(err)
					return
				}
				receipts[index] = receipt
			}
		})
	}
	workers.Wait()
	if err := context.Cause(scanCtx); err != nil {
		return nil, err
	}
	return encodeAndValidateBlockReceipts(receipts, blockHash, len(transactions))
}

func fetchScannedReceipt(ctx context.Context, source ReceiptSource, blockHash common.Hash, transactions types.Transactions, target *types.Receipt, index uint64) (*types.Receipt, error) {
	tx := transactions[index]
	if tx == nil {
		return nil, fmt.Errorf("block transaction %d is nil", index)
	}
	receipt := target
	if target == nil || uint64(target.TransactionIndex) != index {
		var err error
		receipt, err = source.TransactionReceipt(ctx, tx.Hash())
		if err != nil {
			return nil, fmt.Errorf("fetch receipt %d/%d (%s): %w", index+1, len(transactions), tx.Hash(), err)
		}
	}
	if receipt == nil {
		return nil, fmt.Errorf("receipt %d is nil", index)
	}
	if receipt.BlockHash != blockHash {
		return nil, fmt.Errorf("receipt %d block hash mismatch: got %s want %s", index, receipt.BlockHash, blockHash)
	}
	if uint64(receipt.TransactionIndex) != index {
		return nil, fmt.Errorf("receipt %d transaction index mismatch", index)
	}
	if receipt.TxHash != tx.Hash() {
		return nil, fmt.Errorf("receipt %d tx hash mismatch", index)
	}
	if err := validateReceiptLogs(receipt, int(index)); err != nil {
		return nil, err
	}
	return receipt, nil
}

func encodeAndValidateBlockReceipts(receipts []*types.Receipt, blockHash common.Hash, expectedCount int) ([]hexutil.Bytes, error) {
	if len(receipts) != expectedCount {
		return nil, fmt.Errorf("block receipt count %d does not match expected count %d", len(receipts), expectedCount)
	}

	// Canonicalize every receipt only after checking that the provider returned a complete,
	// positionally consistent receipt list for the block.
	out := make([]hexutil.Bytes, len(receipts))
	for i, receipt := range receipts {
		if receipt == nil {
			return nil, fmt.Errorf("block receipt %d is nil", i)
		}
		if receipt.BlockHash != blockHash {
			return nil, fmt.Errorf("block receipt %d block hash mismatch: got %s want %s", i, receipt.BlockHash, blockHash)
		}
		if receipt.TransactionIndex != uint(i) {
			return nil, fmt.Errorf("block receipt %d transaction index mismatch: got %d want %d", i, receipt.TransactionIndex, i)
		}
		if err := validateReceiptLogs(receipt, i); err != nil {
			return nil, err
		}
		encoded, err := proofutil.EncodeReceipt(receipt)
		if err != nil {
			return nil, fmt.Errorf("encode receipt %d: %w", i, err)
		}
		out[i] = encoded
	}
	return out, nil
}

func validateReceiptLogs(receipt *types.Receipt, receiptIndex int) error {
	for logIndex, log := range receipt.Logs {
		if log == nil {
			return fmt.Errorf("block receipt %d log %d is nil", receiptIndex, logIndex)
		}
	}
	return nil
}

func isRPCMethodNotFound(err error) bool {
	var rpcErr rpc.Error
	return errors.As(err, &rpcErr) && rpcErr.ErrorCode() == -32601
}
