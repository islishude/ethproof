package main

import (
	"github.com/islishude/ethproof/proof"
)

type StateProofPackage = proof.StateProofPackage
type StateStorageProof = proof.StateStorageProof
type ReceiptProofPackage = proof.ReceiptProofPackage
type TransactionProofPackage = proof.TransactionProofPackage
type StateAccountClaim = proof.StateAccountClaim
type EventClaim = proof.EventClaim

// OfflineFixtures groups the deterministic offline fixtures checked into proof/testdata.
type OfflineFixtures struct {
	State       *StateProofPackage       `json:"state"`
	Receipt     *ReceiptProofPackage     `json:"receipt"`
	Transaction *TransactionProofPackage `json:"transaction"`
}

// BuildOfflineFixtures constructs the deterministic fixture set used by offline tests.
func BuildOfflineFixtures() (*OfflineFixtures, error) {
	const txIndex = 0
	txReceiptHeader, txs, receipts, err := buildOfflineTransactionReceiptFixture()
	if err != nil {
		return nil, err
	}
	transactionFixture, err := buildOfflineTransactionFixture(txReceiptHeader, txs, txIndex)
	if err != nil {
		return nil, err
	}
	receiptFixture, err := buildOfflineReceiptFixture(txReceiptHeader, txs, receipts, txIndex)
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
