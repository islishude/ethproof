package proof

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/islishude/ethproof/internal/proofutil"
)

// BlockAnchorFromHeader builds a trusted block anchor from a caller-trusted chain ID and header.
// The caller remains responsible for establishing header authenticity and finality.
func BlockAnchorFromHeader(chainID *big.Int, header *types.Header) (BlockAnchor, error) {
	snapshot, err := blockSnapshotHeaderFromHeader(chainID, header)
	if err != nil {
		return BlockAnchor{}, err
	}
	return blockAnchorFromSnapshot(snapshot), nil
}

func blockAnchorFromSnapshot(header blockSnapshotHeader) BlockAnchor {
	return BlockAnchor{
		ChainID:          proofutil.CloneChainID(header.ChainID),
		BlockNumber:      header.BlockNumber,
		BlockHash:        header.BlockHash,
		ParentHash:       header.ParentHash,
		StateRoot:        header.StateRoot,
		TransactionsRoot: header.TransactionsRoot,
		ReceiptsRoot:     header.ReceiptsRoot,
	}
}

func blockSnapshotHeaderFromAnchor(anchor BlockAnchor) (blockSnapshotHeader, error) {
	if anchor.ChainID == nil {
		return blockSnapshotHeader{}, fmt.Errorf("block anchor chain id is required")
	}
	return blockSnapshotHeader{
		ChainID:          proofutil.CloneChainID(anchor.ChainID),
		BlockNumber:      anchor.BlockNumber,
		BlockHash:        anchor.BlockHash,
		ParentHash:       anchor.ParentHash,
		StateRoot:        anchor.StateRoot,
		TransactionsRoot: anchor.TransactionsRoot,
		ReceiptsRoot:     anchor.ReceiptsRoot,
	}, nil
}

func blockSnapshotHeaderFromContext(block BlockContext) (blockSnapshotHeader, error) {
	if block.ChainID == nil {
		return blockSnapshotHeader{}, fmt.Errorf("proof package block chain id is required")
	}
	return blockSnapshotHeader{
		ChainID:          proofutil.CloneChainID(block.ChainID),
		BlockNumber:      block.BlockNumber,
		BlockHash:        block.BlockHash,
		ParentHash:       block.ParentHash,
		StateRoot:        block.StateRoot,
		TransactionsRoot: block.TransactionsRoot,
		ReceiptsRoot:     block.ReceiptsRoot,
	}, nil
}

func verifyBlockContextAgainstAnchor(block BlockContext, anchor BlockAnchor) error {
	embedded, err := blockSnapshotHeaderFromContext(block)
	if err != nil {
		return err
	}
	expected, err := blockSnapshotHeaderFromAnchor(anchor)
	if err != nil {
		return err
	}
	return combineMismatch("proof package", "trusted block anchor", compareHeader(embedded, expected))
}
