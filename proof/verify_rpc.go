package proof

import (
	"context"
	"fmt"

	"github.com/islishude/ethproof/internal/proofutil"
)

// VerifyStateProofPackageAgainstRPCs verifies the state proof locally and then checks that the
// embedded block context matches a fresh independent RPC consensus.
func VerifyStateProofPackageAgainstRPCs(ctx context.Context, pkg *StateProofPackage, req VerifyRPCRequest) error {
	return verifyStateProofPackageAgainstRPCsWithFetcher(ctx, pkg, req, fetchBlockHeadersFromRPCs)
}

func verifyStateProofPackageAgainstRPCsWithFetcher(ctx context.Context, pkg *StateProofPackage, req VerifyRPCRequest, fetcher blockHeaderFetcher) error {
	return verifyPackageAgainstRPCs(ctx, pkg.Block, req, func() error {
		return VerifyStateProofPackage(pkg)
	}, fetcher)
}

// VerifyReceiptProofPackageWithExpectationsAgainstRPCs verifies the receipt proof locally,
// applies optional caller expectations, and then checks the embedded block context against an
// independent RPC consensus.
func VerifyReceiptProofPackageWithExpectationsAgainstRPCs(ctx context.Context, pkg *ReceiptProofPackage, expect *ReceiptExpectations, req VerifyRPCRequest) error {
	return verifyReceiptProofPackageWithExpectationsAgainstRPCsWithFetcher(ctx, pkg, expect, req, fetchBlockHeadersFromRPCs)
}

func verifyReceiptProofPackageWithExpectationsAgainstRPCsWithFetcher(ctx context.Context, pkg *ReceiptProofPackage, expect *ReceiptExpectations, req VerifyRPCRequest, fetcher blockHeaderFetcher) error {
	return verifyPackageAgainstRPCs(ctx, pkg.Block, req, func() error {
		return VerifyReceiptProofPackageWithExpectations(pkg, expect)
	}, fetcher)
}

// VerifyTransactionProofPackageAgainstRPCs verifies the transaction proof locally and then checks
// that the embedded block context matches a fresh independent RPC consensus.
func VerifyTransactionProofPackageAgainstRPCs(ctx context.Context, pkg *TransactionProofPackage, req VerifyRPCRequest) error {
	return verifyTransactionProofPackageAgainstRPCsWithFetcher(ctx, pkg, req, fetchBlockHeadersFromRPCs)
}

func verifyTransactionProofPackageAgainstRPCsWithFetcher(ctx context.Context, pkg *TransactionProofPackage, req VerifyRPCRequest, fetcher blockHeaderFetcher) error {
	return verifyPackageAgainstRPCs(ctx, pkg.Block, req, func() error {
		return VerifyTransactionProofPackage(pkg)
	}, fetcher)
}

func verifyPackageAgainstRPCs(ctx context.Context, block BlockContext, req VerifyRPCRequest, verifyLocal func() error, fetcher blockHeaderFetcher) error {
	// Always verify the package locally before touching independent RPCs so malformed proofs fail
	// fast even if the block header itself still exists on chain.
	if err := verifyLocal(); err != nil {
		return err
	}
	return verifyBlockContextAgainstRPCs(ctx, block, req, fetcher)
}

func verifyBlockContextAgainstRPCs(ctx context.Context, block BlockContext, req VerifyRPCRequest, fetcher blockHeaderFetcher) error {
	// Verify uses its own independent RPC set; it does not trust generation metadata.
	rpcs, err := normalizeRPCURLs(req.RPCURLs, req.MinRPCSources)
	if err != nil {
		return err
	}
	headers, err := fetcher(ctx, rpcs, block.BlockHash)
	if err != nil {
		return err
	}
	if len(headers) == 0 {
		return fmt.Errorf("no rpc headers returned")
	}
	if len(headers) != len(rpcs) {
		return fmt.Errorf("expected %d rpc headers, got %d", len(rpcs), len(headers))
	}

	// First require the verify RPC sources to agree with each other.
	base := headers[0]
	for i := 1; i < len(headers); i++ {
		if err := combineMismatch(base.source, headers[i].source, compareHeader(base.header, headers[i].header)); err != nil {
			return err
		}
	}

	// Then compare the proof package's embedded block context against that agreed independent view.
	if err := combineMismatch("proof package", base.source, compareHeader(blockSnapshotHeader{
		ChainID:          proofutil.CloneChainID(block.ChainID),
		BlockNumber:      block.BlockNumber,
		BlockHash:        block.BlockHash,
		ParentHash:       block.ParentHash,
		StateRoot:        block.StateRoot,
		TransactionsRoot: block.TransactionsRoot,
		ReceiptsRoot:     block.ReceiptsRoot,
	}, base.header)); err != nil {
		return err
	}
	return nil
}
