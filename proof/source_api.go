package proof

import (
	"context"
	"fmt"
	"math/big"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
)

// HeaderSource exposes the header RPCs needed for proof verification and shared block context loading.
type HeaderSource interface {
	SourceName() string
	ChainID(context.Context) (*big.Int, error)
	HeaderByHash(context.Context, common.Hash) (*types.Header, error)
}

// StateSource exposes the RPCs needed to build a state proof.
type StateSource interface {
	HeaderSource
	HeaderByNumber(context.Context, *big.Int) (*types.Header, error)
	GetProof(context.Context, common.Address, []string, *big.Int) (*gethclient.AccountResult, error)
}

// TransactionSource exposes the RPCs needed to build a transaction proof.
type TransactionSource interface {
	HeaderSource
	TransactionByHash(context.Context, common.Hash) (*types.Transaction, bool, error)
	TransactionReceipt(context.Context, common.Hash) (*types.Receipt, error)
	BlockByHash(context.Context, common.Hash) (*types.Block, error)
}

// ReceiptSource exposes the RPCs needed to build a receipt proof.
type ReceiptSource interface {
	TransactionSource
	BlockReceiptsByHash(context.Context, common.Hash) ([]*types.Receipt, error)
}

// StateProofSourcesRequest describes the inputs required to generate a state proof from injected sources.
type StateProofSourcesRequest struct {
	Sources       []StateSource
	MinRPCSources int
	BlockNumber   uint64
	Account       common.Address
	Slots         []common.Hash
}

// ReceiptProofSourcesRequest describes the inputs required to generate a receipt proof from injected sources.
type ReceiptProofSourcesRequest struct {
	Sources       []ReceiptSource
	MinRPCSources int
	TxHash        common.Hash
	LogIndex      uint
}

// TransactionProofSourcesRequest describes the inputs required to generate a transaction proof from injected sources.
type TransactionProofSourcesRequest struct {
	Sources       []TransactionSource
	MinRPCSources int
	TxHash        common.Hash
}

// VerifySourcesRequest describes the independent source set used for source-aware verification.
type VerifySourcesRequest struct {
	Sources       []HeaderSource
	MinRPCSources int
}

type namedSource interface {
	SourceName() string
}

func normalizeSourceNames[S namedSource](sources []S, minSources int) ([]string, error) {
	if minSources == 0 {
		minSources = DefaultMinRPCSources
	}
	if minSources < 1 {
		return nil, fmt.Errorf("min rpc sources must be at least 1, got %d", minSources)
	}
	if len(sources) < minSources {
		return nil, fmt.Errorf("need at least %d distinct rpc sources, got %d", minSources, len(sources))
	}

	seen := make(map[string]struct{}, len(sources))
	names := make([]string, 0, len(sources))
	for i, source := range sources {
		if isNilSource(source) {
			return nil, fmt.Errorf("source %d is nil", i)
		}
		name := strings.TrimSpace(source.SourceName())
		if name == "" {
			return nil, fmt.Errorf("source %d has empty name", i)
		}
		if _, ok := seen[name]; ok {
			return nil, fmt.Errorf("duplicate source name %q", name)
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	return names, nil
}

func collectFromSources[S namedSource, T any](ctx context.Context, sources []S, fetch func(context.Context, S) (T, error)) ([]T, error) {
	type sourceResult struct {
		index int
		value T
		err   error
	}

	out := make([]T, len(sources))
	if len(sources) == 0 {
		return out, nil
	}

	fetchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make(chan sourceResult, len(sources))
	for i, source := range sources {
		go func(index int, source S) {
			value, err := fetch(fetchCtx, source)
			if err != nil {
				err = fmt.Errorf("%s: %w", strings.TrimSpace(source.SourceName()), err)
			}
			results <- sourceResult{
				index: index,
				value: value,
				err:   err,
			}
		}(i, source)
	}

	for range sources {
		select {
		case <-ctx.Done():
			cancel()
			return nil, ctx.Err()
		case result := <-results:
			if result.err != nil {
				cancel()
				return nil, result.err
			}
			out[result.index] = result.value
		}
	}

	return out, nil
}

func isNilSource(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}
