package proof

import (
	"context"
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
)

// HeaderSource exposes the header RPCs needed for online proof verification.
// Implementations must honor context cancellation. The caller owns the source's lifecycle.
type HeaderSource interface {
	ChainID(context.Context) (*big.Int, error)
	HeaderByHash(context.Context, common.Hash) (*types.Header, error)
}

// StateSource exposes the RPCs needed to build a state proof.
// Implementations must honor context cancellation; the caller owns the source.
type StateSource interface {
	ChainID(context.Context) (*big.Int, error)
	HeaderByNumber(context.Context, *big.Int) (*types.Header, error)
	GetProof(context.Context, common.Address, []string, *big.Int) (*gethclient.AccountResult, error)
}

// TransactionSource exposes the RPCs needed to build a transaction proof.
// Implementations must honor context cancellation; the caller owns the source.
type TransactionSource interface {
	ChainID(context.Context) (*big.Int, error)
	TransactionByHash(context.Context, common.Hash) (*types.Transaction, bool, error)
	TransactionReceipt(context.Context, common.Hash) (*types.Receipt, error)
	BlockByHash(context.Context, common.Hash) (*types.Block, error)
}

// ReceiptSource exposes the RPCs needed to build a receipt proof. Implementations
// must support concurrent TransactionReceipt calls during block-receipt fallback.
type ReceiptSource interface {
	TransactionSource
	BlockReceiptsByHash(context.Context, common.Hash) ([]*types.Receipt, error)
}

func validateSource(ctx context.Context, source any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if isNilSource(source) {
		return fmt.Errorf("source is nil")
	}
	return nil
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
