package proofutil

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestBuildIndexTrieProofRejectsOutOfRangeIndex(t *testing.T) {
	_, _, err := BuildIndexTrieProof([]hexutil.Bytes{{0x01}}, 1, common.Hash{}, "transaction")
	if err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEncodeHelpersRejectNilValues(t *testing.T) {
	if _, err := EncodeTransaction(nil); err == nil {
		t.Fatal("expected nil transaction to fail")
	}
	if _, err := EncodeReceipt(nil); err == nil {
		t.Fatal("expected nil receipt to fail")
	}
}

func TestDecodeStorageProofValueRejectsOversizedWord(t *testing.T) {
	raw, err := rlp.EncodeToBytes(make([]byte, common.HashLength+1))
	if err != nil {
		t.Fatalf("rlp encode: %v", err)
	}
	if _, err := DecodeStorageProofValue(raw); err == nil {
		t.Fatal("expected oversized storage word to fail")
	}
}
