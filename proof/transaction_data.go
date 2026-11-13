package proof

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/islishude/ethproof/internal/proofutil"
)

type transactionData struct {
	block               BlockContext
	receipt             *types.Receipt
	transactions        types.Transactions
	encodedTransactions []hexutil.Bytes
}

func fetchTransactionData(ctx context.Context, source TransactionSource, txHash common.Hash) (*transactionData, error) {
	chainID, err := source.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("chain id: %w", err)
	}
	tx, isPending, err := source.TransactionByHash(ctx, txHash)
	if err != nil {
		return nil, fmt.Errorf("fetch tx: %w", err)
	}
	if tx == nil {
		return nil, fmt.Errorf("fetch tx returned nil transaction")
	}
	if isPending {
		return nil, fmt.Errorf("transaction is still pending")
	}

	// Use the receipt to anchor the transaction to a block hash and transaction index before
	// rebuilding the surrounding block transaction list.
	receipt, err := source.TransactionReceipt(ctx, txHash)
	if err != nil {
		return nil, fmt.Errorf("fetch receipt: %w", err)
	}
	if receipt == nil {
		return nil, fmt.Errorf("fetch receipt returned nil receipt")
	}
	if receipt.TxHash != txHash {
		return nil, fmt.Errorf("receipt tx hash mismatch: got %s want %s", receipt.TxHash, txHash)
	}
	block, err := source.BlockByHash(ctx, receipt.BlockHash)
	if err != nil {
		return nil, fmt.Errorf("fetch block: %w", err)
	}
	if block == nil {
		return nil, fmt.Errorf("fetch block returned nil block")
	}
	blockTxs := block.Transactions()
	if receipt.TransactionIndex >= uint(len(blockTxs)) {
		return nil, fmt.Errorf("transaction index %d out of range for block size %d", receipt.TransactionIndex, len(blockTxs))
	}
	blockContext, err := BlockContextFromHeader(chainID, block.Header())
	if err != nil {
		return nil, err
	}

	if block.Hash() != receipt.BlockHash {
		return nil, fmt.Errorf("receipt block hash mismatch: got %s want %s", receipt.BlockHash, block.Hash())
	}

	// Canonicalize the entire block transaction list so proof generation can rebuild the trie
	// locally.
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

	return &transactionData{
		block:               blockContext,
		receipt:             receipt,
		transactions:        blockTxs,
		encodedTransactions: blockTransactions,
	}, nil
}

func (data *transactionData) prove(txHash common.Hash) (*TransactionProofPackage, error) {
	index := uint64(data.receipt.TransactionIndex)
	encoded, nodes, err := buildTransactionTrieAndProof(data.encodedTransactions, index, data.block.TransactionsRoot)
	if err != nil {
		return nil, err
	}
	return &TransactionProofPackage{Block: data.block, TxHash: txHash, TxIndex: index, TransactionRLP: encoded, ProofNodes: nodes}, nil
}
