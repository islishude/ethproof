package proof

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/islishude/ethproof/internal/proofutil"
)

type blockHeaderSource struct {
	source string
	header blockSnapshotHeader
}

type blockHeaderFetcher func(ctx context.Context, sources []HeaderSource, blockHash common.Hash) ([]blockHeaderSource, error)

type blockHeaderSnapshotCollector struct {
	blockHash common.Hash
}

func fetchBlockHeadersFromSources(ctx context.Context, sources []HeaderSource, blockHash common.Hash) ([]blockHeaderSource, error) {
	collector := blockHeaderSnapshotCollector{blockHash: blockHash}
	headers, err := collectFromSources(ctx, sources, collector.fetch)
	if err != nil {
		return nil, err
	}
	for i := range headers {
		headers[i].source = sourceID(i)
	}
	return headers, nil
}

func blockSnapshotHeaderFromHeader(chainID *big.Int, header *types.Header) (blockSnapshotHeader, error) {
	if header == nil {
		return blockSnapshotHeader{}, fmt.Errorf("block header is nil")
	}
	if header.Number == nil {
		return blockSnapshotHeader{}, fmt.Errorf("block header number is nil")
	}
	chainIDValue, err := proofutil.ChainIDFromBig(chainID)
	if err != nil {
		return blockSnapshotHeader{}, err
	}
	return blockSnapshotHeader{
		ChainID:          chainIDValue,
		BlockNumber:      header.Number.Uint64(),
		BlockHash:        header.Hash(),
		ParentHash:       header.ParentHash,
		StateRoot:        header.Root,
		TransactionsRoot: header.TxHash,
		ReceiptsRoot:     header.ReceiptHash,
	}, nil
}

func fetchBlockHeader(ctx context.Context, source HeaderSource, blockHash common.Hash, chainID *big.Int) (*types.Header, blockSnapshotHeader, error) {
	header, err := source.HeaderByHash(ctx, blockHash)
	if err != nil {
		return nil, blockSnapshotHeader{}, fmt.Errorf("fetch header: %w", err)
	}
	snapshot, err := blockSnapshotHeaderFromHeader(chainID, header)
	if err != nil {
		return nil, blockSnapshotHeader{}, err
	}
	return header, snapshot, nil
}

func fetchBlockHeaderSnapshotByHash(ctx context.Context, source HeaderSource, blockHash common.Hash) (blockSnapshotHeader, error) {
	chainID, err := source.ChainID(ctx)
	if err != nil {
		return blockSnapshotHeader{}, fmt.Errorf("chain id: %w", err)
	}
	_, snapshot, err := fetchBlockHeader(ctx, source, blockHash, chainID)
	if err != nil {
		return blockSnapshotHeader{}, err
	}
	return snapshot, nil
}

func (c blockHeaderSnapshotCollector) fetch(ctx context.Context, source HeaderSource) (blockHeaderSource, error) {
	header, err := fetchBlockHeaderSnapshotByHash(ctx, source, c.blockHash)
	if err != nil {
		return blockHeaderSource{}, err
	}
	return blockHeaderSource{
		header: header,
	}, nil
}
