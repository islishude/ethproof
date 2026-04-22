package main

import (
	"bytes"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/islishude/ethproof/internal/proofutil"
)

func TestCanonicalOfflineReceiptDigestsStable(t *testing.T) {
	header, txs, receipts, txIndex, _, err := buildOfflineTransactionReceiptFixture()
	if err != nil {
		t.Fatalf("buildOfflineTransactionReceiptFixture: %v", err)
	}

	blockTransactions := make([]hexutil.Bytes, len(txs))
	blockReceipts := make([]hexutil.Bytes, len(receipts))
	for i := range txs {
		blockTransactions[i], err = proofutil.EncodeTransaction(txs[i])
		if err != nil {
			t.Fatalf("EncodeTransaction(%d): %v", i, err)
		}
		blockReceipts[i], err = proofutil.EncodeReceipt(receipts[i])
		if err != nil {
			t.Fatalf("EncodeReceipt(%d): %v", i, err)
		}
	}

	first, err := canonicalOfflineReceiptDigests(header, blockTransactions, blockReceipts, blockTransactions[txIndex], blockReceipts[txIndex], receipts[txIndex].Logs[0])
	if err != nil {
		t.Fatalf("canonicalOfflineReceiptDigests(first): %v", err)
	}
	second, err := canonicalOfflineReceiptDigests(header, blockTransactions, blockReceipts, blockTransactions[txIndex], blockReceipts[txIndex], receipts[txIndex].Logs[0])
	if err != nil {
		t.Fatalf("canonicalOfflineReceiptDigests(second): %v", err)
	}
	if !slices.Equal(first, second) {
		t.Fatal("expected receipt digests to be stable")
	}
}

func TestCanonicalOfflineStateDigestsStable(t *testing.T) {
	fixtures, err := BuildOfflineFixtures()
	if err != nil {
		t.Fatalf("BuildOfflineFixtures: %v", err)
	}

	statePkg := fixtures.State
	header := blockSnapshotHeader{
		ChainID:          proofutil.CloneChainID(statePkg.Block.ChainID),
		BlockNumber:      statePkg.Block.BlockNumber,
		BlockHash:        statePkg.Block.BlockHash,
		ParentHash:       statePkg.Block.ParentHash,
		StateRoot:        statePkg.Block.StateRoot,
		TransactionsRoot: statePkg.Block.TransactionsRoot,
		ReceiptsRoot:     statePkg.Block.ReceiptsRoot,
	}
	first, err := canonicalOfflineStateDigests(header, statePkg.AccountRLP, statePkg.AccountProofNodes, statePkg.StorageProofs)
	if err != nil {
		t.Fatalf("canonicalOfflineStateDigests(first): %v", err)
	}
	second, err := canonicalOfflineStateDigests(header, statePkg.AccountRLP, statePkg.AccountProofNodes, statePkg.StorageProofs)
	if err != nil {
		t.Fatalf("canonicalOfflineStateDigests(second): %v", err)
	}
	if !slices.Equal(first, second) {
		t.Fatal("expected state digests to be stable")
	}
}

func TestDumpProofNodesSortsByKey(t *testing.T) {
	db := memorydb.New()
	if err := db.Put([]byte{0x10}, []byte("later")); err != nil {
		t.Fatalf("Put(later): %v", err)
	}
	if err := db.Put([]byte{0x01}, []byte("first")); err != nil {
		t.Fatalf("Put(first): %v", err)
	}

	nodes, err := proofutil.DumpProofNodes(db)
	if err != nil {
		t.Fatalf("DumpProofNodes: %v", err)
	}
	if got, want := len(nodes), 2; got != want {
		t.Fatalf("unexpected node count: got %d want %d", got, want)
	}
	if !bytes.Equal(nodes[0], []byte("first")) || !bytes.Equal(nodes[1], []byte("later")) {
		t.Fatalf("unexpected order: %#v", nodes)
	}
}

func TestEncodingRoundTrip(t *testing.T) {
	header, txs, receipts, txIndex, _, err := buildOfflineTransactionReceiptFixture()
	if err != nil {
		t.Fatalf("buildOfflineTransactionReceiptFixture: %v", err)
	}
	_ = header

	txBytes, err := proofutil.EncodeTransaction(txs[txIndex])
	if err != nil {
		t.Fatalf("EncodeTransaction: %v", err)
	}
	decodedTx, roundTripTxBytes, err := proofutil.DecodeTransaction(txBytes)
	if err != nil {
		t.Fatalf("DecodeTransaction: %v", err)
	}
	if !bytes.Equal(txBytes, roundTripTxBytes) {
		t.Fatal("transaction bytes changed after roundtrip")
	}
	if decodedTx.Hash() != txs[txIndex].Hash() {
		t.Fatalf("transaction hash mismatch: got %s want %s", decodedTx.Hash(), txs[txIndex].Hash())
	}

	receiptBytes, err := proofutil.EncodeReceipt(receipts[txIndex])
	if err != nil {
		t.Fatalf("EncodeReceipt: %v", err)
	}
	decodedReceipt, roundTripReceiptBytes, err := proofutil.DecodeReceipt(receiptBytes)
	if err != nil {
		t.Fatalf("DecodeReceipt: %v", err)
	}
	if !bytes.Equal(receiptBytes, roundTripReceiptBytes) {
		t.Fatal("receipt bytes changed after roundtrip")
	}
	if decodedReceipt.Status != receipts[txIndex].Status {
		t.Fatalf("receipt status mismatch: got %d want %d", decodedReceipt.Status, receipts[txIndex].Status)
	}
	if decodedReceipt.Logs[0].Address != common.HexToAddress("0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC") {
		t.Fatalf("unexpected receipt log address: %s", decodedReceipt.Logs[0].Address)
	}
}

func TestOfflineTransactionFieldsStable(t *testing.T) {
	header, txs, _, _, _, err := buildOfflineTransactionReceiptFixture()
	if err != nil {
		t.Fatalf("buildOfflineTransactionReceiptFixture: %v", err)
	}

	first := offlineTransactionFields(txs[0].Hash(), 0, header)
	second := offlineTransactionFields(txs[0].Hash(), 0, header)
	if !slices.Equal(first, second) {
		t.Fatal("expected transaction fields to be stable")
	}
}
