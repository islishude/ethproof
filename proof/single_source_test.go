package proof

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

func TestSingleSourceRejectsNilAndTypedNil(t *testing.T) {
	state := mustLoadStateFixture(t)
	receipt := mustLoadReceiptFixture(t)
	tx := mustLoadTransactionFixture(t)
	var typedState *fakeStateSource
	var typedReceipt *fakeReceiptSource
	var typedHeader *fakeHeaderSource
	for _, typed := range []bool{false, true} {
		var stateSource StateSource
		var receiptSource ReceiptSource
		var headerSource HeaderSource
		if typed {
			stateSource, receiptSource, headerSource = typedState, typedReceipt, typedHeader
		}
		checks := map[string]func() error{
			"generate state": func() error {
				_, err := GenerateStateProof(t.Context(), stateSource, StateProofRequest{Slots: []common.Hash{{}}})
				return err
			},
			"generate receipt": func() error {
				_, err := GenerateReceiptProof(t.Context(), receiptSource, ReceiptProofRequest{})
				return err
			},
			"generate transaction": func() error {
				_, err := GenerateTransactionProof(t.Context(), receiptSource, TransactionProofRequest{})
				return err
			},
			"verify state": func() error {
				return VerifyStateProofPackageAgainstSource(t.Context(), &state, headerSource)
			},
			"verify receipt": func() error {
				return VerifyReceiptProofPackageWithExpectationsAgainstSource(t.Context(), &receipt, nil, headerSource)
			},
			"verify transaction": func() error {
				return VerifyTransactionProofPackageAgainstSource(t.Context(), &tx, headerSource)
			},
		}
		for name, check := range checks {
			t.Run(name, func(t *testing.T) {
				if err := check(); err == nil || !strings.Contains(err.Error(), "source is nil") {
					t.Fatalf("unexpected error: %v", err)
				}
			})
		}
	}
}

type headerRPCService struct {
	chainID *big.Int
	header  *types.Header
	calls   atomic.Int32
}

func (s *headerRPCService) ChainId() *hexutil.Big {
	s.calls.Add(1)
	return (*hexutil.Big)(s.chainID)
}
func (s *headerRPCService) GetBlockByHash(hash common.Hash, _ bool) (*types.Header, error) {
	s.calls.Add(1)
	if hash != s.header.Hash() {
		return nil, errors.New("unknown block")
	}
	return s.header, nil
}
func serveHeaderRPC(t *testing.T, source *fakeHeaderSource) (*httptest.Server, *headerRPCService) {
	t.Helper()
	service := &headerRPCService{chainID: source.chainID, header: source.header}
	server := rpc.NewServer()
	if err := server.RegisterName("eth", service); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(server.Stop)
	httpServer := httptest.NewServer(server)
	t.Cleanup(httpServer.Close)
	return httpServer, service
}

func TestVerifyAgainstSingleRPC(t *testing.T) {
	stateReq, stateSource := testStateRequest(t)
	state, err := GenerateStateProof(t.Context(), stateSource, stateReq)
	if err != nil {
		t.Fatal(err)
	}
	receiptReq, receiptSource := testReceiptRequest(t)
	receipt, err := GenerateReceiptProof(t.Context(), receiptSource, receiptReq)
	if err != nil {
		t.Fatal(err)
	}
	txReq, txSource := testTransactionRequest(t)
	tx, err := GenerateTransactionProof(t.Context(), txSource, txReq)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		source *fakeHeaderSource
		verify func(string) error
	}{
		{"state", stateSource.fakeHeaderSource, func(req string) error { return VerifyStateProofPackageAgainstRPC(t.Context(), state, req) }},
		{"receipt", receiptSource.fakeHeaderSource, func(req string) error {
			return VerifyReceiptProofPackageWithExpectationsAgainstRPC(t.Context(), receipt, &ReceiptExpectations{Emitter: &receipt.Event.Address}, req)
		}},
		{"transaction", txSource.fakeHeaderSource, func(req string) error { return VerifyTransactionProofPackageAgainstRPC(t.Context(), tx, req) }},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			server, service := serveHeaderRPC(t, tt.source)
			if err := tt.verify(" \t" + server.URL + "\n"); err != nil {
				t.Fatal(err)
			}
			if got := service.calls.Load(); got != 2 {
				t.Fatalf("expected chain ID and one header request, got %d calls", got)
			}
			// A separately configured endpoint on another chain must fail even with identical header data.
			wrong := *tt.source
			wrong.chainID = new(big.Int).Add(tt.source.chainID, big.NewInt(1))
			wrongServer, _ := serveHeaderRPC(t, &wrong)
			if err := tt.verify(wrongServer.URL); err == nil || !strings.Contains(err.Error(), "chainId mismatch") {
				t.Fatalf("unexpected wrong-chain result: %v", err)
			}
		})
	}
}

func TestOnlineVerifyChecksLocalProofBeforeRPC(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("invalid proof must not contact RPC") }))
	defer server.Close()
	req := server.URL
	state := mustLoadStateFixture(t)
	state.AccountRLP[0] ^= 1
	receipt := mustLoadReceiptFixture(t)
	tx := mustLoadTransactionFixture(t)
	tx.TransactionRLP[0] ^= 1
	for name, check := range map[string]func() error{
		"state": func() error { return VerifyStateProofPackageAgainstRPC(t.Context(), &state, req) },
		"receipt expectation": func() error {
			return VerifyReceiptProofPackageWithExpectationsAgainstRPC(t.Context(), &receipt, &ReceiptExpectations{Data: []byte{0}}, req)
		},
		"transaction": func() error { return VerifyTransactionProofPackageAgainstRPC(t.Context(), &tx, req) },
	} {
		t.Run(name, func(t *testing.T) {
			if err := check(); err == nil {
				t.Fatal("accepted invalid proof or expectation")
			}
		})
	}
}

func TestVerifySourceRejectsBlockContextMismatch(t *testing.T) {
	req, source := testTransactionRequest(t)
	pkg, err := GenerateTransactionProof(t.Context(), source, req)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name string
		edit func(*BlockContext)
	}{
		{"chainId", func(b *BlockContext) { b.ChainID.SetUint64(99) }},
		{"blockNumber", func(b *BlockContext) { b.BlockNumber++ }},
		{"blockHash", func(b *BlockContext) { b.BlockHash[0] ^= 1 }},
		{"parentHash", func(b *BlockContext) { b.ParentHash[0] ^= 1 }},
		{"stateRoot", func(b *BlockContext) { b.StateRoot[0] ^= 1 }},
		{"receiptsRoot", func(b *BlockContext) { b.ReceiptsRoot[0] ^= 1 }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			altered := cloneTransactionPackage(*pkg)
			altered.Block.ChainID = pkg.Block.ChainID.Clone()
			tt.edit(&altered.Block)
			if err := VerifyTransactionProofPackageAgainstSource(t.Context(), &altered, source); err == nil {
				t.Fatal("accepted mismatched block context")
			}
		})
	}
}

func TestSingleRPCValidationAndLifecycle(t *testing.T) {
	for _, value := range []string{"", " \n\t"} {
		if _, err := openRPCSource(t.Context(), value); err == nil {
			t.Fatal("accepted empty URL")
		}
	}
	source, err := openRPCSource(t.Context(), " http://127.0.0.1:1 ")
	if err != nil {
		t.Fatal(err)
	}
	if source.url != "http://127.0.0.1:1" {
		t.Fatalf("URL not trimmed: %q", source.url)
	}
	source.Close()
	source.Close()
	server := rpc.NewServer()
	defer server.Stop()
	client := rpc.DialInProc(server)
	connected := &rpcSource{raw: client}
	connected.Close()
	connected.Close()
	if err := client.CallContext(t.Context(), nil, "eth_chainId"); !errors.Is(err, rpc.ErrClientQuit) {
		t.Fatalf("expected closed client, got %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := openRPCSource(ctx, "http://127.0.0.1:1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

type waitingHeaderSource struct{ started chan struct{} }

func (s *waitingHeaderSource) ChainID(ctx context.Context) (*big.Int, error) {
	close(s.started)
	<-ctx.Done()
	return nil, ctx.Err()
}
func (*waitingHeaderSource) HeaderByHash(context.Context, common.Hash) (*types.Header, error) {
	return nil, errors.New("unexpected header fetch")
}

func TestVerifySourceCancellationAndFailure(t *testing.T) {
	pkg := mustLoadTransactionFixture(t)
	cause := errors.New("RPC unavailable")
	source := &fakeHeaderSource{chainIDErr: cause}
	if err := VerifyTransactionProofPackageAgainstSource(t.Context(), &pkg, source); !errors.Is(err, cause) {
		t.Fatalf("lost source error: %v", err)
	}
	waiting := &waitingHeaderSource{started: make(chan struct{})}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		result <- VerifyTransactionProofPackageAgainstSource(ctx, &pkg, waiting)
	}()
	select {
	case <-waiting.started:
	case <-time.After(5 * time.Second):
		t.Fatal("source did not start")
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("lost cancellation: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("source did not cancel")
	}
}

func TestProofJSONIgnoresLegacyConsensus(t *testing.T) {
	state := mustLoadStateFixture(t)
	receipt := mustLoadReceiptFixture(t)
	tx := mustLoadTransactionFixture(t)
	for _, tt := range []struct {
		name   string
		pkg    any
		verify func() error
	}{
		{"state", &state, func() error { return VerifyStateProofPackageAgainstEmbeddedRoots(&state) }},
		{"receipt", &receipt, func() error { return VerifyReceiptProofPackageAgainstEmbeddedRoots(&receipt) }},
		{"transaction", &tx, func() error { return VerifyTransactionProofPackageAgainstEmbeddedRoots(&tx) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := json.Marshal(tt.pkg)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(raw), "sourceConsensus") {
				t.Fatal("new proof contains consensus metadata")
			}
			var value map[string]any
			if err := json.Unmarshal(raw, &value); err != nil {
				t.Fatal(err)
			}
			value["block"].(map[string]any)["sourceConsensus"] = map[string]any{"rpcs": []string{"https://legacy.invalid"}, "fields": []any{map[string]any{"consistent": false}}}
			legacy, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "legacy.json")
			if err := os.WriteFile(path, legacy, 0600); err != nil {
				t.Fatal(err)
			}
			if err := LoadJSON(path, tt.pkg); err != nil {
				t.Fatal(err)
			}
			if err := tt.verify(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
