package main

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/islishude/ethproof/internal/proofutil"
)

func canonicalOfflineReceiptDigests(header blockSnapshotHeader, blockTransactions, blockReceipts []hexutil.Bytes, transactionRLP, receiptRLP hexutil.Bytes, log *types.Log) ([]ConsensusDigest, error) {
	headerDigest, err := proofutil.CanonicalDigest(header)
	if err != nil {
		return nil, err
	}
	blockTransactionsDigest, err := proofutil.CanonicalDigest(blockTransactions)
	if err != nil {
		return nil, err
	}
	blockReceiptsDigest, err := proofutil.CanonicalDigest(blockReceipts)
	if err != nil {
		return nil, err
	}
	targetDigest, err := proofutil.CanonicalDigest(struct {
		TransactionRLP hexutil.Bytes `json:"transactionRlp"`
		ReceiptRLP     hexutil.Bytes `json:"receiptRlp"`
		Event          EventClaim    `json:"event"`
	}{
		TransactionRLP: transactionRLP,
		ReceiptRLP:     receiptRLP,
		Event: EventClaim{
			Address: log.Address,
			Topics:  append([]common.Hash(nil), log.Topics...),
			Data:    proofutil.CanonicalBytes(log.Data),
		},
	})
	if err != nil {
		return nil, err
	}
	return []ConsensusDigest{
		{Name: "header", Digest: headerDigest},
		{Name: "blockTransactions", Digest: blockTransactionsDigest},
		{Name: "blockReceipts", Digest: blockReceiptsDigest},
		{Name: "targetReceipt", Digest: targetDigest},
	}, nil
}

func canonicalOfflineStateDigests(header blockSnapshotHeader, accountRLP hexutil.Bytes, accountProof []hexutil.Bytes, slot, slotValue common.Hash, storageProof []hexutil.Bytes) ([]ConsensusDigest, error) {
	headerDigest, err := proofutil.CanonicalDigest(header)
	if err != nil {
		return nil, err
	}
	accountDigest, err := proofutil.CanonicalDigest(struct {
		AccountRLP hexutil.Bytes   `json:"accountRlp"`
		Proof      []hexutil.Bytes `json:"proof"`
	}{
		AccountRLP: accountRLP,
		Proof:      accountProof,
	})
	if err != nil {
		return nil, err
	}
	storageDigest, err := proofutil.CanonicalDigest(struct {
		Slot  common.Hash     `json:"slot"`
		Value common.Hash     `json:"value"`
		Proof []hexutil.Bytes `json:"proof"`
	}{
		Slot:  slot,
		Value: slotValue,
		Proof: storageProof,
	})
	if err != nil {
		return nil, err
	}
	return []ConsensusDigest{
		{Name: "header", Digest: headerDigest},
		{Name: "accountProof", Digest: accountDigest},
		{Name: "storageProof", Digest: storageDigest},
	}, nil
}

func offlineTransactionFields(txHash common.Hash, txIndex uint64, header blockSnapshotHeader) []ConsensusField {
	return []ConsensusField{
		{Name: "chainId", Value: proofutil.ChainIDString(header.ChainID), Consistent: true},
		{Name: "blockNumber", Value: fmt.Sprintf("%d", header.BlockNumber), Consistent: true},
		{Name: "blockHash", Value: header.BlockHash.Hex(), Consistent: true},
		{Name: "parentHash", Value: header.ParentHash.Hex(), Consistent: true},
		{Name: "stateRoot", Value: header.StateRoot.Hex(), Consistent: true},
		{Name: "transactionsRoot", Value: header.TransactionsRoot.Hex(), Consistent: true},
		{Name: "receiptsRoot", Value: header.ReceiptsRoot.Hex(), Consistent: true},
		{Name: "txHash", Value: txHash.Hex(), Consistent: true},
		{Name: "txIndex", Value: fmt.Sprintf("%d", txIndex), Consistent: true},
	}
}

func offlineTransactionDigests(header blockSnapshotHeader, blockTransactions []hexutil.Bytes, transactionRLP hexutil.Bytes) ([]ConsensusDigest, error) {
	headerDigest, err := proofutil.CanonicalDigest(header)
	if err != nil {
		return nil, err
	}
	blockTransactionsDigest, err := proofutil.CanonicalDigest(blockTransactions)
	if err != nil {
		return nil, err
	}
	targetTransactionDigest, err := proofutil.CanonicalDigest(transactionRLP)
	if err != nil {
		return nil, err
	}
	return []ConsensusDigest{
		{Name: "header", Digest: headerDigest},
		{Name: "blockTransactions", Digest: blockTransactionsDigest},
		{Name: "targetTransaction", Digest: targetTransactionDigest},
	}, nil
}
