package main

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/islishude/ethproof/internal/proofutil"
)

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
	const txIndex = 0
	header, txs, receipts, err := buildOfflineTransactionReceiptFixture()
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
