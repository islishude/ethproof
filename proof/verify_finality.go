package proof

import (
	"context"
)

// VerifyStateProofPackageAgainstFinality verifies the local claim and authenticates
// its complete execution block context using the client's RPC and consensus state.
func VerifyStateProofPackageAgainstFinality(ctx context.Context, pkg *StateProofPackage, client *FinalityClient) error {
	if err := VerifyStateProofPackageAgainstEmbeddedRoots(pkg); err != nil {
		return err
	}
	anchor, err := client.FinalizedAnchor(ctx, pkg.Block)
	if err != nil {
		return err
	}
	return VerifyStateProofPackageAgainstBlockAnchor(pkg, anchor)
}

// VerifyTransactionProofPackageAgainstFinality authenticates transaction inclusion
// in a finalized execution block under the configured checkpoint trust policy.
func VerifyTransactionProofPackageAgainstFinality(ctx context.Context, pkg *TransactionProofPackage, client *FinalityClient) error {
	if err := VerifyTransactionProofPackageAgainstEmbeddedRoots(pkg); err != nil {
		return err
	}
	anchor, err := client.FinalizedAnchor(ctx, pkg.Block)
	if err != nil {
		return err
	}
	return VerifyTransactionProofPackageAgainstBlockAnchor(pkg, anchor)
}

// VerifyReceiptProofPackageWithExpectationsAgainstFinality verifies both inclusion
// proofs, the event, caller expectations, and consensus finality of their block.
func VerifyReceiptProofPackageWithExpectationsAgainstFinality(ctx context.Context, pkg *ReceiptProofPackage, expect *ReceiptExpectations, client *FinalityClient) error {
	if err := VerifyReceiptProofPackageWithExpectationsAgainstEmbeddedRoots(pkg, expect); err != nil {
		return err
	}
	anchor, err := client.FinalizedAnchor(ctx, pkg.Block)
	if err != nil {
		return err
	}
	return VerifyReceiptProofPackageWithExpectationsAgainstBlockAnchor(pkg, expect, anchor)
}

// VerifyBlockFinalityFromRPC opens one separately configured execution RPC and
// returns the authenticated anchor. Proof-package callers must also verify MPTs.
func VerifyBlockFinalityFromRPC(ctx context.Context, block BlockContext, rpcURL string, cfg FinalityConfig, onTrustChange func(FinalityTrustEvent)) (BlockAnchor, error) {
	source, err := openRPCSource(ctx, rpcURL)
	if err != nil {
		return BlockAnchor{}, err
	}
	defer source.Close()
	client, err := NewFinalityClientWithDependencies(cfg, source, FinalityDependencies{OnTrustChange: onTrustChange})
	if err != nil {
		return BlockAnchor{}, err
	}
	return client.FinalizedAnchor(ctx, block)
}

// VerifyReceiptProofPackageAgainstFinality verifies receipt/event inclusion and
// finality without additional caller-supplied event expectations.
func VerifyReceiptProofPackageAgainstFinality(ctx context.Context, pkg *ReceiptProofPackage, client *FinalityClient) error {
	return VerifyReceiptProofPackageWithExpectationsAgainstFinality(ctx, pkg, nil, client)
}
