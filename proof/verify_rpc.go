package proof

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

type blockHeaderSource struct {
	source string
	header blockSnapshotHeader
}

type blockHeaderFetcher func(ctx context.Context, urls []string, blockHash common.Hash) ([]blockHeaderSource, error)

func VerifyStateProofPackageAgainstRPCs(ctx context.Context, pkg *StateProofPackage, req VerifyRPCRequest) error {
	return verifyStateProofPackageAgainstRPCsWithFetcher(ctx, pkg, req, fetchBlockHeadersFromRPCs)
}

func verifyStateProofPackageAgainstRPCsWithFetcher(ctx context.Context, pkg *StateProofPackage, req VerifyRPCRequest, fetcher blockHeaderFetcher) error {
	if err := VerifyStateProofPackage(pkg); err != nil {
		return err
	}
	return verifyBlockContextAgainstRPCs(ctx, pkg.Block, req, fetcher)
}

func VerifyReceiptProofPackageWithExpectationsAgainstRPCs(ctx context.Context, pkg *ReceiptProofPackage, expect *ReceiptExpectations, req VerifyRPCRequest) error {
	return verifyReceiptProofPackageWithExpectationsAgainstRPCsWithFetcher(ctx, pkg, expect, req, fetchBlockHeadersFromRPCs)
}

func verifyReceiptProofPackageWithExpectationsAgainstRPCsWithFetcher(ctx context.Context, pkg *ReceiptProofPackage, expect *ReceiptExpectations, req VerifyRPCRequest, fetcher blockHeaderFetcher) error {
	if err := VerifyReceiptProofPackageWithExpectations(pkg, expect); err != nil {
		return err
	}
	return verifyBlockContextAgainstRPCs(ctx, pkg.Block, req, fetcher)
}

func VerifyTransactionProofPackageAgainstRPCs(ctx context.Context, pkg *TransactionProofPackage, req VerifyRPCRequest) error {
	return verifyTransactionProofPackageAgainstRPCsWithFetcher(ctx, pkg, req, fetchBlockHeadersFromRPCs)
}

func verifyTransactionProofPackageAgainstRPCsWithFetcher(ctx context.Context, pkg *TransactionProofPackage, req VerifyRPCRequest, fetcher blockHeaderFetcher) error {
	if err := VerifyTransactionProofPackage(pkg); err != nil {
		return err
	}
	return verifyBlockContextAgainstRPCs(ctx, pkg.Block, req, fetcher)
}

func verifyBlockContextAgainstRPCs(ctx context.Context, block BlockContext, req VerifyRPCRequest, fetcher blockHeaderFetcher) error {
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

	base := headers[0]
	for i := 1; i < len(headers); i++ {
		if err := combineMismatch(base.source, headers[i].source, compareHeader(base.header, headers[i].header)); err != nil {
			return err
		}
	}
	if err := combineMismatch("proof package", base.source, compareHeader(blockSnapshotHeader{
		ChainID:          cloneChainID(block.ChainID),
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
