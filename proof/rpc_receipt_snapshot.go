package proof

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/islishude/ethproof/internal/proofutil"
)

func fetchReceiptSnapshot(ctx context.Context, source *rpcSource, txHash common.Hash, logIndex uint) (*receiptSnapshot, error) {
	logger := loggerFromContext(ctx)
	logger.Debug("fetching receipt snapshot", "rpc_url", source.url, "tx_hash", txHash, "log_index", logIndex)
	// Reuse the normalized transaction snapshot so receipt proof generation inherits the exact
	// transaction bytes and block context that transaction proof generation would see.
	txSnapshot, err := fetchTransactionSnapshot(ctx, source, txHash)
	if err != nil {
		return nil, err
	}
	receipt, err := source.eth.TransactionReceipt(ctx, txHash)
	if err != nil {
		return nil, fmt.Errorf("fetch target receipt: %w", err)
	}
	if logIndex >= uint(len(receipt.Logs)) {
		return nil, fmt.Errorf("log-index %d out of range (receipt has %d logs)", logIndex, len(receipt.Logs))
	}
	if receipt.BlockHash != txSnapshot.Header.BlockHash {
		return nil, fmt.Errorf("target receipt block hash mismatch: got %s want %s", receipt.BlockHash, txSnapshot.Header.BlockHash)
	}
	if uint64(receipt.TransactionIndex) != txSnapshot.TxIndex {
		return nil, fmt.Errorf("target receipt transaction index mismatch: got %d want %d", receipt.TransactionIndex, txSnapshot.TxIndex)
	}
	if receipt.TxHash != txHash {
		return nil, fmt.Errorf("target receipt tx hash mismatch: got %s want %s", receipt.TxHash, txHash)
	}

	// Encode the target receipt and then require it to match the same bytes recovered from the
	// full block receipt list. This catches inconsistencies between point lookups and block scans.
	receiptRLP, err := proofutil.EncodeReceipt(receipt)
	if err != nil {
		return nil, fmt.Errorf("encode target receipt: %w", err)
	}
	blockReceipts, err := fetchBlockReceipts(ctx, source, txSnapshot.Header.BlockHash, len(txSnapshot.BlockTransactions))
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(blockReceipts[txSnapshot.TxIndex], receiptRLP) {
		return nil, fmt.Errorf("receipt bytes mismatch between block receipts and target receipt lookup")
	}
	logger.Debug("validated receipt snapshot locally", "rpc_url", source.url, "block_hash", txSnapshot.Header.BlockHash)

	// Persist the claimed event as simple address/topics/data fields so package verification does
	// not depend on any geth-specific receipt representation.
	log := receipt.Logs[logIndex]
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

func fetchBlockReceipts(ctx context.Context, source *rpcSource, blockHash common.Hash, expectedCount int) ([]hexutil.Bytes, error) {
	receipts, err := source.eth.BlockReceipts(ctx, rpc.BlockNumberOrHashWithHash(blockHash, true))
	if err != nil {
		// Some providers do not implement eth_getBlockReceipts. Fall back to scanning each tx so
		// receipt proof generation still works while remaining strict about normalized results.
		if isRPCMethodNotFound(err) {
			return fetchBlockReceiptsWithFallback(ctx, source, blockHash, expectedCount, func() ([]hexutil.Bytes, error) {
				return fetchBlockReceiptsByTransactionScan(ctx, source, blockHash, expectedCount)
			})
		}
		return nil, fmt.Errorf("fetch block receipts: %w", err)
	}
	return encodeAndValidateBlockReceipts(receipts, blockHash, expectedCount)
}

func fetchBlockReceiptsWithFallback(ctx context.Context, source *rpcSource, blockHash common.Hash, expectedCount int, fallback func() ([]hexutil.Bytes, error)) ([]hexutil.Bytes, error) {
	logger := loggerFromContext(ctx)
	logger.Warn("eth_getBlockReceipts unavailable; falling back to transaction scan",
		"rpc_url", source.url,
		"block_hash", blockHash,
		"expected_count", expectedCount,
	)
	return fallback()
}

func fetchBlockReceiptsByTransactionScan(ctx context.Context, source *rpcSource, blockHash common.Hash, expectedCount int) ([]hexutil.Bytes, error) {
	block, err := source.eth.BlockByHash(ctx, blockHash)
	if err != nil {
		return nil, fmt.Errorf("fetch block for receipts: %w", err)
	}
	if len(block.Transactions()) != expectedCount {
		return nil, fmt.Errorf("block transaction count %d does not match expected count %d", len(block.Transactions()), expectedCount)
	}

	// When eth_getBlockReceipts is unavailable, rebuild the receipt list one transaction at a time
	// and validate that each receipt still points back to the expected block position.
	blockReceipts := make([]hexutil.Bytes, len(block.Transactions()))
	for i, blockTx := range block.Transactions() {
		receipt, receiptErr := source.eth.TransactionReceipt(ctx, blockTx.Hash())
		if receiptErr != nil {
			return nil, fmt.Errorf("fetch receipt %d/%d (%s): %w", i+1, len(block.Transactions()), blockTx.Hash(), receiptErr)
		}
		if receipt.BlockHash != blockHash {
			return nil, fmt.Errorf("receipt %d block hash mismatch: got %s want %s", i, receipt.BlockHash, blockHash)
		}
		if receipt.TransactionIndex != uint(i) {
			return nil, fmt.Errorf("receipt %d transaction index mismatch: got %d want %d", i, receipt.TransactionIndex, i)
		}
		if receipt.TxHash != blockTx.Hash() {
			return nil, fmt.Errorf("receipt %d tx hash mismatch: got %s want %s", i, receipt.TxHash, blockTx.Hash())
		}
		encoded, encErr := proofutil.EncodeReceipt(receipt)
		if encErr != nil {
			return nil, fmt.Errorf("encode receipt %d: %w", i, encErr)
		}
		blockReceipts[i] = encoded
	}
	return blockReceipts, nil
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
		encoded, err := proofutil.EncodeReceipt(receipt)
		if err != nil {
			return nil, fmt.Errorf("encode receipt %d: %w", i, err)
		}
		out[i] = encoded
	}
	return out, nil
}

func isRPCMethodNotFound(err error) bool {
	var rpcErr rpc.Error
	return errors.As(err, &rpcErr) && rpcErr.ErrorCode() == -32601
}
