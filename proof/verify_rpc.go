package proof

import (
	"context"
	"fmt"
)

// VerifyStateProofPackageAgainstRPC verifies the proof locally, then compares its block context
// with a header from the caller-configured RPC. The caller establishes trust in that source.
func VerifyStateProofPackageAgainstRPC(ctx context.Context, pkg *StateProofPackage, rpcURL string) error {
	if err := VerifyStateProofPackageAgainstEmbeddedRoots(pkg); err != nil {
		return err
	}
	source, err := openRPCSource(ctx, rpcURL)
	if err != nil {
		return err
	}
	defer source.Close()
	return verifyBlockContextAgainstSource(ctx, pkg.Block, source)
}

// VerifyStateProofPackageAgainstSource verifies the proof locally, then compares its block context
// with a header from the caller-configured Source. The caller establishes trust in that source.
func VerifyStateProofPackageAgainstSource(ctx context.Context, pkg *StateProofPackage, source HeaderSource) error {
	if err := VerifyStateProofPackageAgainstEmbeddedRoots(pkg); err != nil {
		return err
	}
	return verifyBlockContextAgainstSource(ctx, pkg.Block, source)
}

// VerifyTransactionProofPackageAgainstRPC verifies the proof locally, then compares its block context
// with a header from the caller-configured RPC. The caller establishes trust in that source.
func VerifyTransactionProofPackageAgainstRPC(ctx context.Context, pkg *TransactionProofPackage, rpcURL string) error {
	if err := VerifyTransactionProofPackageAgainstEmbeddedRoots(pkg); err != nil {
		return err
	}
	source, err := openRPCSource(ctx, rpcURL)
	if err != nil {
		return err
	}
	defer source.Close()
	return verifyBlockContextAgainstSource(ctx, pkg.Block, source)
}

// VerifyTransactionProofPackageAgainstSource verifies the proof locally, then compares its block context
// with a header from the caller-configured Source. The caller establishes trust in that source.
func VerifyTransactionProofPackageAgainstSource(ctx context.Context, pkg *TransactionProofPackage, source HeaderSource) error {
	if err := VerifyTransactionProofPackageAgainstEmbeddedRoots(pkg); err != nil {
		return err
	}
	return verifyBlockContextAgainstSource(ctx, pkg.Block, source)
}

// VerifyReceiptProofPackageWithExpectationsAgainstRPC verifies the proof locally, then compares its block context
// with a header from the caller-configured RPC. The caller establishes trust in that source.
func VerifyReceiptProofPackageWithExpectationsAgainstRPC(ctx context.Context, pkg *ReceiptProofPackage, expect *ReceiptExpectations, rpcURL string) error {
	if err := VerifyReceiptProofPackageWithExpectationsAgainstEmbeddedRoots(pkg, expect); err != nil {
		return err
	}
	source, err := openRPCSource(ctx, rpcURL)
	if err != nil {
		return err
	}
	defer source.Close()
	return verifyBlockContextAgainstSource(ctx, pkg.Block, source)
}

// VerifyReceiptProofPackageWithExpectationsAgainstSource verifies the proof locally, then compares its block context
// with a header from the caller-configured Source. The caller establishes trust in that source.
func VerifyReceiptProofPackageWithExpectationsAgainstSource(ctx context.Context, pkg *ReceiptProofPackage, expect *ReceiptExpectations, source HeaderSource) error {
	if err := VerifyReceiptProofPackageWithExpectationsAgainstEmbeddedRoots(pkg, expect); err != nil {
		return err
	}
	return verifyBlockContextAgainstSource(ctx, pkg.Block, source)
}

func verifyBlockContextAgainstSource(ctx context.Context, block BlockContext, source HeaderSource) error {
	if err := validateSource(ctx, source); err != nil {
		return err
	}
	chainID, err := source.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("chain id: %w", err)
	}
	header, err := source.HeaderByHash(ctx, block.BlockHash)
	if err != nil {
		return fmt.Errorf("fetch header: %w", err)
	}
	anchor, err := BlockAnchorFromHeader(chainID, header)
	if err != nil {
		return err
	}
	return verifyBlockContextAgainstAnchor(block, anchor)
}
