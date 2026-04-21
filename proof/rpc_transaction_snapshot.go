package proof

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/islishude/ethproof/internal/proofutil"
)

func fetchTransactionSnapshot(ctx context.Context, source *rpcSource, txHash common.Hash) (*transactionSnapshot, error) {
	chainID, err := source.eth.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("chain id: %w", err)
	}
	tx, isPending, err := source.eth.TransactionByHash(ctx, txHash)
	if err != nil {
		return nil, fmt.Errorf("fetch tx: %w", err)
	}
	if isPending {
		return nil, fmt.Errorf("transaction is still pending")
	}

	// Use the receipt to anchor the transaction to a block hash and transaction index before
	// rebuilding the surrounding block transaction list.
	receipt, err := source.eth.TransactionReceipt(ctx, txHash)
	if err != nil {
		return nil, fmt.Errorf("fetch receipt: %w", err)
	}
	block, err := source.eth.BlockByHash(ctx, receipt.BlockHash)
	if err != nil {
		return nil, fmt.Errorf("fetch block: %w", err)
	}
	if int(receipt.TransactionIndex) >= len(block.Transactions()) {
		return nil, fmt.Errorf("transaction index %d out of range for block size %d", receipt.TransactionIndex, len(block.Transactions()))
	}
	header, headerSnapshot, err := fetchBlockHeader(ctx, source, receipt.BlockHash, chainID)
	if err != nil {
		return nil, err
	}
	if header.Hash() != block.Hash() {
		return nil, fmt.Errorf("header hash %s does not match block hash %s", header.Hash(), block.Hash())
	}

	// Canonicalize the entire block transaction list so proof generation can rebuild the trie
	// locally and compare normalized bytes across sources.
	blockTxs := block.Transactions()
	transactionRLP, err := proofutil.EncodeTransaction(tx)
	if err != nil {
		return nil, fmt.Errorf("encode target tx: %w", err)
	}
	blockTransactions := make([]hexutil.Bytes, len(blockTxs))
	for i, blockTx := range blockTxs {
		encoded, encErr := proofutil.EncodeTransaction(blockTx)
		if encErr != nil {
			return nil, fmt.Errorf("encode block tx %d: %w", i, encErr)
		}
		blockTransactions[i] = encoded
	}

	// Cross-check the point lookup result against the transaction found at receipt.TransactionIndex
	// in the full block body.
	targetIndex := uint64(receipt.TransactionIndex)
	if blockTxs[targetIndex].Hash() != txHash {
		return nil, fmt.Errorf("block transaction[%d] hash mismatch: got %s want %s", targetIndex, blockTxs[targetIndex].Hash(), txHash)
	}
	if !bytes.Equal(blockTransactions[targetIndex], transactionRLP) {
		return nil, fmt.Errorf("transaction bytes mismatch between block body and tx lookup")
	}

	return &transactionSnapshot{
		Header:            headerSnapshot,
		TxHash:            txHash,
		TxIndex:           targetIndex,
		TransactionRLP:    transactionRLP,
		BlockTransactions: blockTransactions,
	}, nil
}
