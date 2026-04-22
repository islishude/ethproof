package proof

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestVerifyStateProofPackageAgainstSources(t *testing.T) {
	req, verifyReq, _ := testStateProofSourcesRequest(t)

	pkg, err := GenerateStateProofFromSources(context.Background(), req)
	if err != nil {
		t.Fatalf("GenerateStateProofFromSources: %v", err)
	}
	if err := VerifyStateProofPackageAgainstSources(context.Background(), pkg, verifyReq); err != nil {
		t.Fatalf("VerifyStateProofPackageAgainstSources: %v", err)
	}
}

func TestVerifyReceiptProofPackageWithExpectationsAgainstSources(t *testing.T) {
	req, verifyReq, _ := testReceiptProofSourcesRequest(t)

	pkg, err := GenerateReceiptProofFromSources(context.Background(), req)
	if err != nil {
		t.Fatalf("GenerateReceiptProofFromSources: %v", err)
	}
	expect := &ReceiptExpectations{
		Emitter: &pkg.Event.Address,
		Topics:  append([]common.Hash(nil), pkg.Event.Topics...),
		Data:    append([]byte(nil), pkg.Event.Data...),
	}
	if err := VerifyReceiptProofPackageWithExpectationsAgainstSources(context.Background(), pkg, expect, verifyReq); err != nil {
		t.Fatalf("VerifyReceiptProofPackageWithExpectationsAgainstSources: %v", err)
	}
}

func TestVerifyTransactionProofPackageAgainstRPCsIgnoresGenerationRPCMetadata(t *testing.T) {
	pkg := mustLoadTransactionFixture(t)
	pkg.Block.SourceConsensus.RPCs = []string{"http://generate-rpc.invalid"}

	req := VerifyRPCRequest{
		RPCURLs:       []string{"https://verify-1.example", "https://verify-2.example", "https://verify-3.example"},
		MinRPCSources: 3,
	}
	if err := verifyTransactionProofPackageAgainstRPCsWithFetcher(context.Background(), &pkg, req, fixedBlockHeaderFetcher(pkg.Block)); err != nil {
		t.Fatalf("verifyTransactionProofPackageAgainstRPCsWithFetcher: %v", err)
	}
}

func TestVerifyTransactionProofPackageAgainstRPCsRejectsTamperedBlockHash(t *testing.T) {
	pkg := mustLoadTransactionFixture(t)
	originalBlock := pkg.Block
	originalHash := pkg.Block.BlockHash
	pkg.Block.BlockHash = common.HexToHash("0x1234")

	req := VerifyRPCRequest{
		RPCURLs:       []string{"https://verify-1.example", "https://verify-2.example", "https://verify-3.example"},
		MinRPCSources: 3,
	}
	err := verifyTransactionProofPackageAgainstRPCsWithFetcher(context.Background(), &pkg, req, func(_ context.Context, sources []HeaderSource, blockHash common.Hash) ([]blockHeaderSource, error) {
		if blockHash != originalHash {
			return nil, fmt.Errorf("fetch header: block %s not found", blockHash)
		}
		return fixedBlockHeaderFetcher(originalBlock)(context.Background(), sources, blockHash)
	})
	if err == nil {
		t.Fatal("expected tampered block hash to fail rpc-aware verification")
	}
	if !strings.Contains(err.Error(), "fetch header: block") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyTransactionProofPackageAgainstSourcesRejectsVerifySourceMismatch(t *testing.T) {
	pkg := mustLoadTransactionFixture(t)
	req := VerifySourcesRequest{
		Sources: []HeaderSource{
			&fakeHeaderSource{name: "verify-a"},
			&fakeHeaderSource{name: "verify-b"},
			&fakeHeaderSource{name: "verify-c"},
		},
		MinRPCSources: 3,
	}

	base := blockSnapshotHeaderFromBlockContext(pkg.Block)
	mismatch := cloneBlockSnapshotHeader(base)
	mismatch.ParentHash = common.HexToHash("0xbeef")

	err := verifyTransactionProofPackageAgainstSourcesWithFetcher(context.Background(), &pkg, req, func(_ context.Context, sources []HeaderSource, blockHash common.Hash) ([]blockHeaderSource, error) {
		if blockHash != pkg.Block.BlockHash {
			return nil, fmt.Errorf("fetch header: block %s not found", blockHash)
		}
		out := make([]blockHeaderSource, 0, len(sources))
		for i, source := range sources {
			header := cloneBlockSnapshotHeader(base)
			if i == 1 {
				header = cloneBlockSnapshotHeader(mismatch)
			}
			out = append(out, blockHeaderSource{
				source: source.SourceName(),
				header: header,
			})
		}
		return out, nil
	})
	if err == nil {
		t.Fatal("expected mismatched verify sources to fail verification")
	}
	if !strings.Contains(err.Error(), "normalized data mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func fixedBlockHeaderFetcher(block BlockContext) blockHeaderFetcher {
	expectedHash := block.BlockHash
	template := blockSnapshotHeaderFromBlockContext(block)
	return func(_ context.Context, sources []HeaderSource, blockHash common.Hash) ([]blockHeaderSource, error) {
		if blockHash != expectedHash {
			return nil, fmt.Errorf("fetch header: block %s not found", blockHash)
		}
		out := make([]blockHeaderSource, 0, len(sources))
		for _, source := range sources {
			out = append(out, blockHeaderSource{
				source: source.SourceName(),
				header: cloneBlockSnapshotHeader(template),
			})
		}
		return out, nil
	}
}
