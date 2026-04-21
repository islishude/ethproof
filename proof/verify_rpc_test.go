package proof

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestVerifyStateProofPackageAgainstRPCs(t *testing.T) {
	pkg := mustLoadStateFixture(t)
	req := VerifyRPCRequest{
		RPCURLs:       []string{"https://verify-1.example", "https://verify-2.example", "https://verify-3.example"},
		MinRPCSources: 3,
	}

	if err := verifyStateProofPackageAgainstRPCs(context.Background(), &pkg, req, fixedBlockHeaderFetcher(pkg.Block)); err != nil {
		t.Fatalf("verifyStateProofPackageAgainstRPCs: %v", err)
	}
}

func TestVerifyReceiptProofPackageWithExpectationsAgainstRPCs(t *testing.T) {
	pkg := mustLoadReceiptFixture(t)
	req := VerifyRPCRequest{
		RPCURLs:       []string{"https://verify-1.example", "https://verify-2.example", "https://verify-3.example"},
		MinRPCSources: 3,
	}
	expect := &ReceiptExpectations{
		Emitter: &pkg.Event.Address,
		Topics:  append([]common.Hash(nil), pkg.Event.Topics...),
		Data:    append([]byte(nil), pkg.Event.Data...),
	}

	if err := verifyReceiptProofPackageWithExpectationsAgainstRPCs(context.Background(), &pkg, expect, req, fixedBlockHeaderFetcher(pkg.Block)); err != nil {
		t.Fatalf("verifyReceiptProofPackageWithExpectationsAgainstRPCs: %v", err)
	}
}

func TestVerifyTransactionProofPackageAgainstRPCsIgnoresGenerationRPCMetadata(t *testing.T) {
	pkg := mustLoadTransactionFixture(t)
	pkg.Block.SourceConsensus.RPCs = []string{"http://generate-rpc.invalid"}
	req := VerifyRPCRequest{
		RPCURLs:       []string{"https://verify-1.example", "https://verify-2.example", "https://verify-3.example"},
		MinRPCSources: 3,
	}

	if err := verifyTransactionProofPackageAgainstRPCs(context.Background(), &pkg, req, fixedBlockHeaderFetcher(pkg.Block)); err != nil {
		t.Fatalf("verifyTransactionProofPackageAgainstRPCs: %v", err)
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

	err := verifyTransactionProofPackageAgainstRPCs(context.Background(), &pkg, req, func(_ context.Context, urls []string, blockHash common.Hash) ([]blockHeaderSource, error) {
		if blockHash != originalHash {
			return nil, fmt.Errorf("fetch header: block %s not found", blockHash)
		}
		return fixedBlockHeaderFetcher(originalBlock)(context.Background(), urls, blockHash)
	})
	if err == nil {
		t.Fatal("expected tampered block hash to fail rpc-aware verification")
	}
	if !strings.Contains(err.Error(), "fetch header: block") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyTransactionProofPackageAgainstRPCsRejectsVerifyRPCMismatch(t *testing.T) {
	pkg := mustLoadTransactionFixture(t)
	req := VerifyRPCRequest{
		RPCURLs:       []string{"https://verify-1.example", "https://verify-2.example", "https://verify-3.example"},
		MinRPCSources: 3,
	}
	base := blockContextHeader(pkg.Block)
	mismatch := cloneBlockSnapshotHeader(base)
	mismatch.ParentHash = common.HexToHash("0xbeef")

	err := verifyTransactionProofPackageAgainstRPCs(context.Background(), &pkg, req, func(_ context.Context, urls []string, blockHash common.Hash) ([]blockHeaderSource, error) {
		if blockHash != pkg.Block.BlockHash {
			return nil, fmt.Errorf("fetch header: block %s not found", blockHash)
		}
		out := make([]blockHeaderSource, 0, len(urls))
		for i, url := range urls {
			header := cloneBlockSnapshotHeader(base)
			if i == 1 {
				header = cloneBlockSnapshotHeader(mismatch)
			}
			out = append(out, blockHeaderSource{
				source: url,
				header: header,
			})
		}
		return out, nil
	})
	if err == nil {
		t.Fatal("expected mismatched verify rpc headers to fail verification")
	}
	if !strings.Contains(err.Error(), "normalized data mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func fixedBlockHeaderFetcher(block BlockContext) blockHeaderFetcher {
	expectedHash := block.BlockHash
	template := blockContextHeader(block)
	return func(_ context.Context, urls []string, blockHash common.Hash) ([]blockHeaderSource, error) {
		if blockHash != expectedHash {
			return nil, fmt.Errorf("fetch header: block %s not found", blockHash)
		}
		out := make([]blockHeaderSource, 0, len(urls))
		for _, url := range urls {
			out = append(out, blockHeaderSource{
				source: url,
				header: cloneBlockSnapshotHeader(template),
			})
		}
		return out, nil
	}
}

func cloneBlockSnapshotHeader(in blockSnapshotHeader) blockSnapshotHeader {
	out := in
	out.ChainID = cloneChainID(in.ChainID)
	return out
}
