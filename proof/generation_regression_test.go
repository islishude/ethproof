package proof

import (
	"errors"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestGenerationDoesNotFetchHeaderByHash(t *testing.T) {
	// The full block already includes its header. A source's verification-only
	// HeaderByHash method must not be needed by either transaction-based generator.
	txReq, source := testTransactionRequest(t)
	source.headerErr = errors.New("unexpected header lookup")
	tx, err := GenerateTransactionProof(t.Context(), struct{ TransactionSource }{source}, txReq)
	if err != nil {
		t.Fatal(err)
	}
	if tx.Block.BlockHash != source.block.Hash() {
		t.Fatal("wrong block header")
	}
	receipt, err := GenerateReceiptProof(t.Context(), struct{ ReceiptSource }{source}, ReceiptProofRequest{TxHash: txReq.TxHash})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Block.BlockHash != tx.Block.BlockHash {
		t.Fatal("generators used different block headers")
	}
	stateReq, stateSource := testStateRequest(t)
	stateSource.headerErr = errors.New("unexpected header lookup")
	if _, err := GenerateStateProof(t.Context(), struct{ StateSource }{stateSource}, stateReq); err != nil {
		t.Fatal(err)
	}
}

func TestGenerationRejectsBlockBodyRootMismatch(t *testing.T) {
	for _, receiptProof := range []bool{false, true} {
		t.Run(map[bool]string{false: "transaction", true: "receipt"}[receiptProof], func(t *testing.T) {
			req, source := testTransactionRequest(t)
			txs := append(types.Transactions(nil), source.block.Transactions()...)
			to := common.HexToAddress("0x1111111111111111111111111111111111111111")
			txs[0] = types.NewTx(&types.LegacyTx{Nonce: 999, To: &to})
			// Keep the original header/root but replace a non-target transaction.
			source.block = source.block.WithBody(types.Body{Transactions: txs})
			source.blockReceipts[0].TxHash = txs[0].Hash()
			var err error
			if receiptProof {
				_, err = GenerateReceiptProof(t.Context(), source, ReceiptProofRequest{TxHash: req.TxHash})
			} else {
				_, err = GenerateTransactionProof(t.Context(), source, req)
			}
			if err == nil || !strings.Contains(err.Error(), "transactionsRoot mismatch") {
				t.Fatalf("unexpected changed-transaction result: %v", err)
			}
		})
	}
}

func TestGenerationRejectsReceiptRootMismatch(t *testing.T) {
	req, source := testReceiptRequest(t)
	source.blockReceipts[0].CumulativeGasUsed++
	if _, err := GenerateReceiptProof(t.Context(), source, req); err == nil || !strings.Contains(err.Error(), "receiptsRoot mismatch") {
		t.Fatalf("unexpected changed-receipt result: %v", err)
	}
}

func TestStateGenerationRejectsCorruptResponse(t *testing.T) {
	for _, tt := range []struct {
		name   string
		change func(*fakeStateSource)
	}{
		{"account claim", func(s *fakeStateSource) { s.proof.Nonce++ }},
		{"storage claim", func(s *fakeStateSource) {
			s.proof.StorageProof[0].Value.Add(s.proof.StorageProof[0].Value, common.Big1)
		}},
		{"account proof", func(s *fakeStateSource) { s.proof.AccountProof = nil }},
		{"storage proof", func(s *fakeStateSource) { s.proof.StorageProof[0].Proof = nil }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req, source := testStateRequest(t)
			tt.change(source)
			if _, err := GenerateStateProof(t.Context(), source, req); err == nil {
				t.Fatal("accepted corrupt state response")
			}
		})
	}
}
