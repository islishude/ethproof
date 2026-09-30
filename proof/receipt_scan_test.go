package proof

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type countingReceiptSource struct {
	*fakeReceiptSource
	mu       sync.Mutex
	blocks   int
	receipts map[common.Hash]int
}

func (s *countingReceiptSource) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	s.mu.Lock()
	s.blocks++
	s.mu.Unlock()
	return s.fakeReceiptSource.BlockByHash(ctx, hash)
}
func (s *countingReceiptSource) TransactionReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error) {
	s.mu.Lock()
	s.receipts[hash]++
	s.mu.Unlock()
	return s.fakeReceiptSource.TransactionReceipt(ctx, hash)
}
func TestReceiptGenerationReusesFetchedData(t *testing.T) {
	for _, fallback := range []bool{false, true} {
		t.Run(fmt.Sprint(fallback), func(t *testing.T) {
			base, hash, logIndex := mustReceiptSource(t)
			if fallback {
				base.blockReceiptsErr = methodNotFoundRPCError{}
			}
			source := &countingReceiptSource{fakeReceiptSource: base, receipts: make(map[common.Hash]int)}
			pkg, err := GenerateReceiptProofFromSources(t.Context(), ReceiptProofSourcesRequest{Sources: []ReceiptSource{source}, MinRPCSources: 1, TxHash: hash, LogIndex: logIndex})
			if err != nil {
				t.Fatal(err)
			}
			if err := VerifyReceiptProofPackageAgainstEmbeddedRoots(pkg); err != nil {
				t.Fatal(err)
			}
			if source.blocks != 1 || source.receipts[hash] != 1 {
				t.Fatalf("duplicate lookups: blocks=%d receipts=%v", source.blocks, source.receipts)
			}
			if fallback {
				for _, tx := range base.block.Transactions() {
					if source.receipts[tx.Hash()] != 1 {
						t.Fatalf("receipt calls: %v", source.receipts)
					}
				}
			} else if len(source.receipts) != 1 {
				t.Fatalf("unexpected receipt calls: %v", source.receipts)
			}
		})
	}
}

type receiptScanSource struct {
	ReceiptSource
	fetch func(context.Context, common.Hash) (*types.Receipt, error)
}

func (s receiptScanSource) TransactionReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error) {
	return s.fetch(ctx, hash)
}

func TestReceiptScanConcurrencyOrderingAndCancellation(t *testing.T) {
	const count = 25
	hash := common.HexToHash("0x1234")
	txs := make(types.Transactions, count)
	receipts := make(map[common.Hash]*types.Receipt, count)
	for i := range txs {
		txs[i] = types.NewTx(&types.LegacyTx{Nonce: uint64(i), Gas: 21000})
		receipts[txs[i].Hash()] = &types.Receipt{TxHash: txs[i].Hash(), BlockHash: hash, TransactionIndex: uint(i), Status: 1, CumulativeGasUsed: uint64(i+1) * 21000}
	}
	for _, mode := range []string{"success", "source error", "caller cancellation"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			entered := make(chan struct{}, count)
			release := make(chan struct{})
			var once sync.Once
			defer once.Do(func() { close(release) })
			var active, maxActive atomic.Int64
			sentinel := errors.New("receipt unavailable")
			source := receiptScanSource{fetch: func(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
				n := active.Add(1)
				defer active.Add(-1)
				for old := maxActive.Load(); n > old; old = maxActive.Load() {
					if maxActive.CompareAndSwap(old, n) {
						break
					}
				}
				entered <- struct{}{}
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-release:
				}
				if mode == "source error" {
					if txHash == txs[0].Hash() {
						return nil, sentinel
					}
					<-ctx.Done()
					return nil, ctx.Err()
				}
				return receipts[txHash], nil
			}}
			done := make(chan error, 1)
			go func() {
				got, err := fetchBlockReceiptsByTransactionScan(ctx, source, hash, txs, nil)
				if err == nil {
					if len(got) != count {
						done <- fmt.Errorf("receipt count %d", len(got))
						return
					}
					for i, raw := range got {
						var receipt types.Receipt
						if err := receipt.UnmarshalBinary(raw); err != nil {
							done <- err
							return
						}
						if receipt.CumulativeGasUsed != uint64(i+1)*21000 {
							done <- fmt.Errorf("receipt order changed at %d", i)
							return
						}
					}
				}
				done <- err
			}()
			for range receiptScanConcurrency {
				select {
				case <-entered:
				case <-ctx.Done():
					t.Fatal("workers did not start concurrently")
				}
			}
			if mode == "caller cancellation" {
				cancel()
			} else {
				once.Do(func() { close(release) })
			}
			var err error
			select {
			case err = <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("receipt scan did not terminate")
			}
			switch mode {
			case "success":
				if err != nil {
					t.Fatal(err)
				}
			case "source error":
				if !errors.Is(err, sentinel) {
					t.Fatalf("lost source error: %v", err)
				}
			case "caller cancellation":
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("lost cancellation: %v", err)
				}
			}
			if maxActive.Load() != receiptScanConcurrency || active.Load() != 0 {
				t.Fatalf("concurrency maximum=%d remaining=%d", maxActive.Load(), active.Load())
			}
		})
	}
}

func TestReceiptCollectionRejectsInvalidMetadata(t *testing.T) {
	for _, fallback := range []bool{false, true} {
		for _, field := range []string{"block", "index", "transaction", "nil log", "nil receipt"} {
			t.Run(fmt.Sprintf("%t/%s", fallback, field), func(t *testing.T) {
				source, _, _ := mustReceiptSource(t)
				hash := source.block.Transactions()[0].Hash()
				receipt := cloneReceipt(source.receiptsByTxHash[hash])
				switch field {
				case "block":
					receipt.BlockHash = common.Hash{}
				case "index":
					receipt.TransactionIndex = 10
				case "transaction":
					receipt.TxHash = common.Hash{}
				case "nil log":
					receipt.Logs = []*types.Log{nil}
				case "nil receipt":
					receipt = nil
				}
				source.receiptsByTxHash[hash] = receipt
				source.blockReceipts[0] = receipt
				if fallback {
					source.blockReceiptsErr = methodNotFoundRPCError{}
				}
				if _, err := fetchBlockReceipts(t.Context(), source, source.block.Hash(), source.block.Transactions(), nil); err == nil {
					t.Fatal("accepted invalid receipt metadata")
				}
			})
		}
	}
}
