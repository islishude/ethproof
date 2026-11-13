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

// BeaconFinalitySource supplies untrusted light-client data; every response is
// verified locally. Implementations must honor cancellation.
type BeaconFinalitySource interface {
	Bootstrap(context.Context, common.Hash) (FinalityBootstrap, error)
	Updates(context.Context, uint64, uint64) ([]FinalityUpdate, error)
	FinalityUpdate(context.Context) (FinalityUpdate, error)
}

// CheckpointSource is explicitly trusted when acquiring or refreshing a checkpoint.
// Name is a stable opaque operator identifier, never a credential-bearing URL.
type CheckpointSource interface {
	Name() string
	Genesis(context.Context) (uint64, common.Hash, error)
	FinalizedCheckpoint(context.Context) (FinalityCheckpoint, error)
	CheckpointHeader(context.Context, common.Hash) (CheckpointHeader, error)
}

// FinalityStore holds locally trusted state. Load returns os.ErrNotExist for a
// new store. Lock must serialize the entire read/verify/write transaction.
// Implementations must not accept snapshots from untrusted proof producers.
type FinalityStore interface {
	Lock(context.Context) (func() error, error)
	Load(context.Context) ([]byte, error)
	Save(context.Context, []byte) error
}
