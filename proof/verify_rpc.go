package proof

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/islishude/ethproof/internal/proofutil"
)

// VerifyStateProofPackageAgainstRPCs verifies the state proof locally and then checks that the
// embedded block context matches a fresh independent RPC consensus.
func VerifyStateProofPackageAgainstRPCs(ctx context.Context, pkg *StateProofPackage, req VerifyRPCRequest) error {
	return verifyStateProofPackageAgainstRPCsWithFetcher(ctx, pkg, req, fetchBlockHeadersFromRPCs)
}

func verifyStateProofPackageAgainstRPCsWithFetcher(ctx context.Context, pkg *StateProofPackage, req VerifyRPCRequest, fetcher blockHeaderFetcher) error {
	logger := loggerFromContext(ctx).With("proof_type", "state")
	return verifyPackageAgainstRPCs(ctx, logger, pkg.Block, req, func() error {
		return verifyStateProofPackageWithLogger(logger, pkg)
	}, fetcher)
}

// VerifyReceiptProofPackageWithExpectationsAgainstRPCs verifies the receipt proof locally,
// applies optional caller expectations, and then checks the embedded block context against an
// independent RPC consensus.
func VerifyReceiptProofPackageWithExpectationsAgainstRPCs(ctx context.Context, pkg *ReceiptProofPackage, expect *ReceiptExpectations, req VerifyRPCRequest) error {
	return verifyReceiptProofPackageWithExpectationsAgainstRPCsWithFetcher(ctx, pkg, expect, req, fetchBlockHeadersFromRPCs)
}

func verifyReceiptProofPackageWithExpectationsAgainstRPCsWithFetcher(ctx context.Context, pkg *ReceiptProofPackage, expect *ReceiptExpectations, req VerifyRPCRequest, fetcher blockHeaderFetcher) error {
	logger := loggerFromContext(ctx).With("proof_type", "receipt")
	return verifyPackageAgainstRPCs(ctx, logger, pkg.Block, req, func() error {
		return verifyReceiptProofPackageWithExpectationsWithLogger(logger, pkg, expect)
	}, fetcher)
}

// VerifyTransactionProofPackageAgainstRPCs verifies the transaction proof locally and then checks
// that the embedded block context matches a fresh independent RPC consensus.
func VerifyTransactionProofPackageAgainstRPCs(ctx context.Context, pkg *TransactionProofPackage, req VerifyRPCRequest) error {
	return verifyTransactionProofPackageAgainstRPCsWithFetcher(ctx, pkg, req, fetchBlockHeadersFromRPCs)
}

func verifyTransactionProofPackageAgainstRPCsWithFetcher(ctx context.Context, pkg *TransactionProofPackage, req VerifyRPCRequest, fetcher blockHeaderFetcher) error {
	logger := loggerFromContext(ctx).With("proof_type", "transaction")
	return verifyPackageAgainstRPCs(ctx, logger, pkg.Block, req, func() error {
		return verifyTransactionProofPackageWithLogger(logger, pkg)
	}, fetcher)
}

func verifyPackageAgainstRPCs(ctx context.Context, logger *slog.Logger, block BlockContext, req VerifyRPCRequest, verifyLocal func() error, fetcher blockHeaderFetcher) error {
	// Always verify the package locally before touching independent RPCs so malformed proofs fail
	// fast even if the block header itself still exists on chain.
	logger.Info("verify proof started", "block_hash", block.BlockHash, "block_number", block.BlockNumber)
	if err := verifyLocal(); err != nil {
		return err
	}
	logger.Debug("local proof verification completed", "block_hash", block.BlockHash)
	if err := verifyBlockContextAgainstRPCs(ctx, logger, block, req, fetcher); err != nil {
		return err
	}
	logger.Info("verify proof completed", "block_hash", block.BlockHash, "block_number", block.BlockNumber)
	return nil
}

func verifyBlockContextAgainstRPCs(ctx context.Context, logger *slog.Logger, block BlockContext, req VerifyRPCRequest, fetcher blockHeaderFetcher) error {
	// Verify uses its own independent RPC set; it does not trust generation metadata.
	rpcs, err := normalizeRPCURLs(req.RPCURLs, req.MinRPCSources)
	if err != nil {
		return err
	}
	logger.Debug("verifying block context against independent rpcs", "rpc_count", len(rpcs), "block_hash", block.BlockHash)
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
	logger.Debug("fetched verify rpc headers", "header_count", len(headers))

	// First require the verify RPC sources to agree with each other.
	base := headers[0]
	for i := 1; i < len(headers); i++ {
		if err := combineMismatch(base.source, headers[i].source, compareHeader(base.header, headers[i].header)); err != nil {
			return err
		}
	}
	logger.Debug("independent rpc consensus established", "rpc_count", len(rpcs), "block_hash", base.header.BlockHash)

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
