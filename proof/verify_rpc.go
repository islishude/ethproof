package proof

import (
	"context"
	"fmt"

	"github.com/islishude/ethproof/internal/proofutil"
)

// VerifyStateProofPackageAgainstRPCs verifies the state proof locally and then checks that the
// embedded block context matches a fresh independent RPC consensus.
func VerifyStateProofPackageAgainstRPCs(ctx context.Context, pkg *StateProofPackage, req VerifyRPCRequest) error {
	return verifyStateProofPackageAgainstRPCsWithFetcher(ctx, pkg, req, fetchBlockHeadersFromSources)
}

func verifyStateProofPackageAgainstRPCsWithFetcher(ctx context.Context, pkg *StateProofPackage, req VerifyRPCRequest, fetcher blockHeaderFetcher) error {
	_, err := withNormalizedRPCSources(ctx, req.RPCURLs, req.MinRPCSources, func(sources []*rpcSource) (struct{}, error) {
		return struct{}{}, verifyStateProofPackageAgainstSourcesWithFetcher(ctx, pkg, VerifySourcesRequest{
			Sources:       headerSourcesFromRPCSources(sources),
			MinRPCSources: req.MinRPCSources,
		}, fetcher)
	})
	return err
}

// VerifyStateProofPackageAgainstSources verifies the state proof locally and then checks that the
// embedded block context matches a fresh independent source consensus.
func VerifyStateProofPackageAgainstSources(ctx context.Context, pkg *StateProofPackage, req VerifySourcesRequest) error {
	return verifyStateProofPackageAgainstSourcesWithFetcher(ctx, pkg, req, fetchBlockHeadersFromSources)
}

func verifyStateProofPackageAgainstSourcesWithFetcher(ctx context.Context, pkg *StateProofPackage, req VerifySourcesRequest, fetcher blockHeaderFetcher) error {
	return verifyPackageAgainstSources(ctx, pkg.Block, req, func() error {
		return verifyStateProofPackage(pkg)
	}, fetcher)
}

// VerifyReceiptProofPackageWithExpectationsAgainstRPCs verifies the receipt proof locally,
// applies optional caller expectations, and then checks the embedded block context against an
// independent RPC consensus.
func VerifyReceiptProofPackageWithExpectationsAgainstRPCs(ctx context.Context, pkg *ReceiptProofPackage, expect *ReceiptExpectations, req VerifyRPCRequest) error {
	return verifyReceiptProofPackageWithExpectationsAgainstRPCsWithFetcher(ctx, pkg, expect, req, fetchBlockHeadersFromSources)
}

func verifyReceiptProofPackageWithExpectationsAgainstRPCsWithFetcher(ctx context.Context, pkg *ReceiptProofPackage, expect *ReceiptExpectations, req VerifyRPCRequest, fetcher blockHeaderFetcher) error {
	_, err := withNormalizedRPCSources(ctx, req.RPCURLs, req.MinRPCSources, func(sources []*rpcSource) (struct{}, error) {
		return struct{}{}, verifyReceiptProofPackageWithExpectationsAgainstSourcesWithFetcher(ctx, pkg, expect, VerifySourcesRequest{
			Sources:       headerSourcesFromRPCSources(sources),
			MinRPCSources: req.MinRPCSources,
		}, fetcher)
	})
	return err
}

// VerifyReceiptProofPackageWithExpectationsAgainstSources verifies the receipt proof locally,
// applies optional caller expectations, and then checks the embedded block context against an
// independent source consensus.
func VerifyReceiptProofPackageWithExpectationsAgainstSources(ctx context.Context, pkg *ReceiptProofPackage, expect *ReceiptExpectations, req VerifySourcesRequest) error {
	return verifyReceiptProofPackageWithExpectationsAgainstSourcesWithFetcher(ctx, pkg, expect, req, fetchBlockHeadersFromSources)
}

func verifyReceiptProofPackageWithExpectationsAgainstSourcesWithFetcher(ctx context.Context, pkg *ReceiptProofPackage, expect *ReceiptExpectations, req VerifySourcesRequest, fetcher blockHeaderFetcher) error {
	return verifyPackageAgainstSources(ctx, pkg.Block, req, func() error {
		return verifyReceiptProofPackageLocal(pkg, expect)
	}, fetcher)
}

// VerifyTransactionProofPackageAgainstRPCs verifies the transaction proof locally and then checks
// that the embedded block context matches a fresh independent RPC consensus.
func VerifyTransactionProofPackageAgainstRPCs(ctx context.Context, pkg *TransactionProofPackage, req VerifyRPCRequest) error {
	return verifyTransactionProofPackageAgainstRPCsWithFetcher(ctx, pkg, req, fetchBlockHeadersFromSources)
}

func verifyTransactionProofPackageAgainstRPCsWithFetcher(ctx context.Context, pkg *TransactionProofPackage, req VerifyRPCRequest, fetcher blockHeaderFetcher) error {
	_, err := withNormalizedRPCSources(ctx, req.RPCURLs, req.MinRPCSources, func(sources []*rpcSource) (struct{}, error) {
		return struct{}{}, verifyTransactionProofPackageAgainstSourcesWithFetcher(ctx, pkg, VerifySourcesRequest{
			Sources:       headerSourcesFromRPCSources(sources),
			MinRPCSources: req.MinRPCSources,
		}, fetcher)
	})
	return err
}

// VerifyTransactionProofPackageAgainstSources verifies the transaction proof locally and then checks
// that the embedded block context matches a fresh independent source consensus.
func VerifyTransactionProofPackageAgainstSources(ctx context.Context, pkg *TransactionProofPackage, req VerifySourcesRequest) error {
	return verifyTransactionProofPackageAgainstSourcesWithFetcher(ctx, pkg, req, fetchBlockHeadersFromSources)
}

func verifyTransactionProofPackageAgainstSourcesWithFetcher(ctx context.Context, pkg *TransactionProofPackage, req VerifySourcesRequest, fetcher blockHeaderFetcher) error {
	return verifyPackageAgainstSources(ctx, pkg.Block, req, func() error {
		return verifyTransactionProofPackage(pkg)
	}, fetcher)
}

func verifyPackageAgainstSources(ctx context.Context, block BlockContext, req VerifySourcesRequest, verifyLocal func() error, fetcher blockHeaderFetcher) error {
	if err := verifyLocal(); err != nil {
		return err
	}
	return verifyBlockContextAgainstSources(ctx, block, req, fetcher)
}

func verifyBlockContextAgainstSources(ctx context.Context, block BlockContext, req VerifySourcesRequest, fetcher blockHeaderFetcher) error {
	sourceNames, err := normalizeSourceNames(req.Sources, req.MinRPCSources)
	if err != nil {
		return err
	}
	headers, err := fetcher(ctx, req.Sources, block.BlockHash)
	if err != nil {
		return err
	}
	if len(headers) == 0 {
		return fmt.Errorf("no rpc headers returned")
	}
	if len(headers) != len(sourceNames) {
		return fmt.Errorf("expected %d rpc headers, got %d", len(sourceNames), len(headers))
	}

	// First require the verify sources to agree with each other.
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
