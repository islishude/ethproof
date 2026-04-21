package proof

import (
	"context"
	"fmt"

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

func collectFromRPCs[T any](ctx context.Context, urls []string, fetch func(context.Context, *rpcSource) (T, error)) ([]T, error) {
	// Open and close the full source set in one place so callers only describe how to fetch a
	// single-source snapshot and do not repeat lifecycle/error-wrapping code.
	sources, err := openRPCSources(ctx, urls)
	if err != nil {
		return nil, err
	}
	defer closeRPCSources(sources)

	out := make([]T, 0, len(sources))
	for _, source := range sources {
		// Prefix every fetch error with the source URL so strict multi-RPC failures are actionable.
		value, fetchErr := fetch(ctx, source)
		if fetchErr != nil {
			return nil, fmt.Errorf("%s: %w", source.url, fetchErr)
		}
		out = append(out, value)
	}
	return out, nil
}
