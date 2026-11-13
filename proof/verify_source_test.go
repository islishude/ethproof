package proof

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestVerifyStateProofPackageAgainstSource(t *testing.T) {
	req, source := testStateRequest(t)

	pkg, err := GenerateStateProof(context.Background(), source, req)
	if err != nil {
		t.Fatalf("GenerateStateProof: %v", err)
	}
	if err := VerifyStateProofPackageAgainstSource(context.Background(), pkg, source); err != nil {
		t.Fatalf("VerifyStateProofPackageAgainstSource: %v", err)
	}
}

func TestVerifyReceiptProofPackageWithExpectationsAgainstSource(t *testing.T) {
	req, source := testReceiptRequest(t)

	pkg, err := GenerateReceiptProof(context.Background(), source, req)
	if err != nil {
		t.Fatalf("GenerateReceiptProof: %v", err)
	}
	expect := &ReceiptExpectations{
		Emitter: &pkg.Event.Address,
		Topics:  append([]common.Hash(nil), pkg.Event.Topics...),
		Data:    append([]byte(nil), pkg.Event.Data...),
	}
	if err := VerifyReceiptProofPackageWithExpectationsAgainstSource(context.Background(), pkg, expect, source); err != nil {
		t.Fatalf("VerifyReceiptProofPackageWithExpectationsAgainstSource: %v", err)
	}
}
