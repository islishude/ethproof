package proof

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type rpcSource struct {
	url  string
	raw  *rpc.Client
	eth  *ethclient.Client
	geth *gethclient.Client
}

func normalizeRPCURLs(urls []string, minSources int) ([]string, error) {
	seen := make(map[string]struct{}, len(urls))
	out := make([]string, 0, len(urls))
	for _, raw := range urls {
		url := strings.TrimSpace(raw)
		if url == "" {
			continue
		}
		if _, ok := seen[url]; ok {
			continue
		}
		seen[url] = struct{}{}
		out = append(out, url)
	}
	if minSources == 0 {
		minSources = DefaultMinRPCSources
	}
	if minSources < 1 {
		return nil, fmt.Errorf("min rpc sources must be at least 1, got %d", minSources)
	}
	if len(out) < minSources {
		return nil, fmt.Errorf("need at least %d distinct rpc sources, got %d", minSources, len(out))
	}
	return out, nil
}

func (s *rpcSource) SourceName() string {
	return s.url
}

func (s *rpcSource) ChainID(ctx context.Context) (*big.Int, error) {
	return s.eth.ChainID(ctx)
}

func (s *rpcSource) HeaderByHash(ctx context.Context, blockHash common.Hash) (*types.Header, error) {
	return s.eth.HeaderByHash(ctx, blockHash)
}

func (s *rpcSource) HeaderByNumber(ctx context.Context, blockNumber *big.Int) (*types.Header, error) {
	return s.eth.HeaderByNumber(ctx, blockNumber)
}

func (s *rpcSource) GetProof(ctx context.Context, account common.Address, keys []string, blockNumber *big.Int) (*gethclient.AccountResult, error) {
	return s.geth.GetProof(ctx, account, keys, blockNumber)
}

func (s *rpcSource) TransactionByHash(ctx context.Context, txHash common.Hash) (*types.Transaction, bool, error) {
	return s.eth.TransactionByHash(ctx, txHash)
}

func (s *rpcSource) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return s.eth.TransactionReceipt(ctx, txHash)
}

func (s *rpcSource) BlockByHash(ctx context.Context, blockHash common.Hash) (*types.Block, error) {
	return s.eth.BlockByHash(ctx, blockHash)
}

func (s *rpcSource) BlockReceiptsByHash(ctx context.Context, blockHash common.Hash) ([]*types.Receipt, error) {
	return s.eth.BlockReceipts(ctx, rpc.BlockNumberOrHashWithHash(blockHash, true))
}

func openRPCSources(ctx context.Context, urls []string) ([]*rpcSource, error) {
	sources := make([]*rpcSource, 0, len(urls))
	for _, url := range urls {
		raw, err := rpc.DialContext(ctx, url)
		if err != nil {
			closeRPCSources(sources)
			return nil, fmt.Errorf("dial rpc %s: %w", url, err)
		}
		sources = append(sources, &rpcSource{
			url:  url,
			raw:  raw,
			eth:  ethclient.NewClient(raw),
			geth: gethclient.New(raw),
		})
	}
	return sources, nil
}

func closeRPCSources(sources []*rpcSource) {
	for _, source := range sources {
		if source != nil && source.raw != nil {
			source.raw.Close()
		}
	}
}

type rpcSourceSet struct {
	sources   []*rpcSource
	closer    func([]*rpcSource)
	closeOnce sync.Once
}

func openNormalizedRPCSources(ctx context.Context, urls []string, minSources int) (*rpcSourceSet, error) {
	return openNormalizedRPCSourcesUsing(ctx, urls, minSources, openRPCSources, closeRPCSources)
}

func openNormalizedRPCSourcesUsing(
	ctx context.Context,
	urls []string,
	minSources int,
	opener func(context.Context, []string) ([]*rpcSource, error),
	closer func([]*rpcSource),
) (*rpcSourceSet, error) {
	rpcs, err := normalizeRPCURLs(urls, minSources)
	if err != nil {
		return nil, err
	}
	sources, err := opener(ctx, rpcs)
	if err != nil {
		return nil, err
	}

	return &rpcSourceSet{
		sources: sources,
		closer:  closer,
	}, nil
}

func (s *rpcSourceSet) Close() {
	if s == nil {
		return
	}
	s.closeOnce.Do(func() {
		if s.closer != nil {
			s.closer(s.sources)
		}
	})
}

func (s *rpcSourceSet) StateSources() []StateSource {
	out := make([]StateSource, len(s.sources))
	for i, source := range s.sources {
		out[i] = source
	}
	return out
}

func (s *rpcSourceSet) ReceiptSources() []ReceiptSource {
	out := make([]ReceiptSource, len(s.sources))
	for i, source := range s.sources {
		out[i] = source
	}
	return out
}

func (s *rpcSourceSet) TransactionSources() []TransactionSource {
	out := make([]TransactionSource, len(s.sources))
	for i, source := range s.sources {
		out[i] = source
	}
	return out
}

func (s *rpcSourceSet) HeaderSources() []HeaderSource {
	out := make([]HeaderSource, len(s.sources))
	for i, source := range s.sources {
		out[i] = source
	}
	return out
}
