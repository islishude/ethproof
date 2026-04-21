package main

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/islishude/ethproof/proof"
)

type StateProofPackage = proof.StateProofPackage
type ReceiptProofPackage = proof.ReceiptProofPackage
type TransactionProofPackage = proof.TransactionProofPackage
type StateAccountClaim = proof.StateAccountClaim
type EventClaim = proof.EventClaim
type SourceConsensus = proof.SourceConsensus
type ConsensusDigest = proof.ConsensusDigest
type ConsensusField = proof.ConsensusField

// OfflineFixtures groups the deterministic offline fixtures checked into proof/testdata.
type OfflineFixtures struct {
	State       *StateProofPackage       `json:"state"`
	Receipt     *ReceiptProofPackage     `json:"receipt"`
	Transaction *TransactionProofPackage `json:"transaction"`
}

type blockSnapshotHeader struct {
	ChainID          *uint256.Int `json:"chainId"`
	BlockNumber      uint64       `json:"blockNumber"`
	BlockHash        common.Hash  `json:"blockHash"`
	ParentHash       common.Hash  `json:"parentHash"`
	StateRoot        common.Hash  `json:"stateRoot"`
	TransactionsRoot common.Hash  `json:"transactionsRoot"`
	ReceiptsRoot     common.Hash  `json:"receiptsRoot"`
}

// BuildOfflineFixtures constructs the deterministic fixture set used by offline tests.
func BuildOfflineFixtures() (*OfflineFixtures, error) {
	txReceiptHeader, txs, receipts, txIndex, receiptConsensus, err := buildOfflineTransactionReceiptFixture()
	if err != nil {
		return nil, err
	}
	transactionFixture, err := transactionSnapshotFromBlock(txReceiptHeader, txs, txIndex, receiptConsensus)
	if err != nil {
		return nil, err
	}
	receiptFixture, err := buildOfflineReceiptFixture(txReceiptHeader, txs, receipts, txIndex, receiptConsensus)
	if err != nil {
		return nil, err
	}
	stateFixture, err := buildOfflineStateFixture()
	if err != nil {
		return nil, err
	}
	return &OfflineFixtures{
		State:       stateFixture,
		Receipt:     receiptFixture,
		Transaction: transactionFixture,
	}, nil
}
