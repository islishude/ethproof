package proof

import (
	"context"
	"fmt"
	"math/big"
	"net/url"
	"strings"

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

func (s *rpcSource) ChainID(ctx context.Context) (*big.Int, error) {
	value, err := s.eth.ChainID(ctx)
	return value, s.redactError(err)
}

func (s *rpcSource) HeaderByHash(ctx context.Context, blockHash common.Hash) (*types.Header, error) {
	value, err := s.eth.HeaderByHash(ctx, blockHash)
	return value, s.redactError(err)
}

func (s *rpcSource) HeaderByNumber(ctx context.Context, blockNumber *big.Int) (*types.Header, error) {
	value, err := s.eth.HeaderByNumber(ctx, blockNumber)
	return value, s.redactError(err)
}

func (s *rpcSource) GetProof(ctx context.Context, account common.Address, keys []string, blockNumber *big.Int) (*gethclient.AccountResult, error) {
	value, err := s.geth.GetProof(ctx, account, keys, blockNumber)
	return value, s.redactError(err)
}

func (s *rpcSource) TransactionByHash(ctx context.Context, txHash common.Hash) (*types.Transaction, bool, error) {
	tx, pending, err := s.eth.TransactionByHash(ctx, txHash)
	return tx, pending, s.redactError(err)
}

func (s *rpcSource) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	value, err := s.eth.TransactionReceipt(ctx, txHash)
	return value, s.redactError(err)
}

func (s *rpcSource) BlockByHash(ctx context.Context, blockHash common.Hash) (*types.Block, error) {
	value, err := s.eth.BlockByHash(ctx, blockHash)
	return value, s.redactError(err)
}

func (s *rpcSource) BlockReceiptsByHash(ctx context.Context, blockHash common.Hash) ([]*types.Receipt, error) {
	value, err := s.eth.BlockReceipts(ctx, rpc.BlockNumberOrHashWithHash(blockHash, true))
	return value, s.redactError(err)
}

func openRPCSource(ctx context.Context, rawURL string) (*rpcSource, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("rpc URL is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	raw, err := rpc.DialContext(ctx, rawURL)
	if err != nil {
		return nil, fmt.Errorf("dial rpc: %w", redactRPCError(err, rawURL))
	}
	return &rpcSource{url: rawURL, raw: raw, eth: ethclient.NewClient(raw), geth: gethclient.New(raw)}, nil
}

func (s *rpcSource) Close() {
	if s != nil && s.raw != nil {
		s.raw.Close()
	}
}

type redactedRPCError struct {
	message string
	cause   error
}

func (e redactedRPCError) Error() string { return e.message }

func (e redactedRPCError) Unwrap() error { return e.cause }

func (s *rpcSource) redactError(err error) error {
	if s == nil {
		return err
	}
	return redactRPCError(err, s.url)
}

func redactRPCError(err error, rawURL string) error {
	if err == nil {
		return nil
	}
	message := strings.ReplaceAll(err.Error(), rawURL, "rpc")
	parsed, parseErr := url.Parse(rawURL)
	if parseErr == nil {
		redactions := []string{parsed.RawQuery, parsed.Fragment}
		if parsed.Path != "" && parsed.Path != "/" {
			redactions = append(redactions, parsed.Path)
		}
		if parsed.User != nil {
			redactions = append(redactions, parsed.User.String(), parsed.User.Username())
			if password, ok := parsed.User.Password(); ok {
				redactions = append(redactions, password)
			}
		}
		for _, secret := range redactions {
			if secret != "" {
				message = strings.ReplaceAll(message, secret, "<redacted>")
			}
		}
	}
	return redactedRPCError{message: message, cause: err}
}
