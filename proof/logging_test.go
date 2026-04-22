package proof

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

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

func TestFetchBlockReceiptsWithFallbackWithoutLogger(t *testing.T) {
	source := &fakeReceiptSource{fakeHeaderSource: &fakeHeaderSource{name: "source-a"}}

	called := false
	got, err := fetchBlockReceiptsWithFallback(source, common.HexToHash("0x1234"), 2, func() ([]hexutil.Bytes, error) {
		called = true
		return []hexutil.Bytes{{0x01}}, nil
	})
	if err != nil {
		t.Fatalf("fetchBlockReceiptsWithFallback: %v", err)
	}
	if !called {
		t.Fatal("expected fallback to be called")
	}
	if len(got) != 1 {
		t.Fatalf("expected one receipt from fallback, got %d", len(got))
	}
}
