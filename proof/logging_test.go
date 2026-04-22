package proof

import (
	"bytes"
	"context"
	"testing"
)

type methodNotFoundRPCError struct{}

func (methodNotFoundRPCError) Error() string {
	return "method not found"
}

func (methodNotFoundRPCError) ErrorCode() int {
	return -32601
}

func TestVerifyTransactionProofPackageAgainstSourcesWithoutLogger(t *testing.T) {
	req, verifyReq, _ := testTransactionProofSourcesRequest(t)

	pkg, err := GenerateTransactionProofFromSources(context.Background(), req)
	if err != nil {
		t.Fatalf("GenerateTransactionProofFromSources: %v", err)
	}
	if err := VerifyTransactionProofPackageAgainstSources(context.Background(), pkg, verifyReq); err != nil {
		t.Fatalf("VerifyTransactionProofPackageAgainstSources: %v", err)
	}
}

func TestFetchBlockReceiptsFallsBackWithoutLogger(t *testing.T) {
	sources, _, _, _, _ := testReceiptSourceSet(t)
	source := sources[0].(*fakeReceiptSource)
	source.blockReceiptsErr = methodNotFoundRPCError{}

	blockHash := source.block.Hash()
	expectedCount := len(source.block.Transactions())
	want, err := encodeAndValidateBlockReceipts(source.blockReceipts, blockHash, expectedCount)
	if err != nil {
		t.Fatalf("encodeAndValidateBlockReceipts: %v", err)
	}

	got, err := fetchBlockReceipts(context.Background(), source, blockHash, expectedCount)
	if err != nil {
		t.Fatalf("fetchBlockReceipts: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("unexpected receipt count: got %d want %d", len(got), len(want))
	}
	for i := range got {
		if !bytes.Equal(got[i], want[i]) {
			t.Fatalf("receipt %d mismatch", i)
		}
	}
}
