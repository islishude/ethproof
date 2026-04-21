package proof

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestVerifyTransactionProofPackageAgainstRPCsLogsProgress(t *testing.T) {
	pkg := mustLoadTransactionFixture(t)
	req := VerifyRPCRequest{
		RPCURLs:       []string{"https://verify-1.example", "https://verify-2.example", "https://verify-3.example"},
		MinRPCSources: 3,
	}
	logger, buf := testLogger(slog.LevelDebug)
	ctx := WithLogger(context.Background(), logger)

	if err := verifyTransactionProofPackageAgainstRPCsWithFetcher(ctx, &pkg, req, fixedBlockHeaderFetcher(pkg.Block)); err != nil {
		t.Fatalf("verifyTransactionProofPackageAgainstRPCsWithFetcher: %v", err)
	}

	logOutput := buf.String()
	for _, want := range []string{
		"verify proof started",
		"verifying local transaction proof",
		"independent rpc consensus established",
		"verify proof completed",
	} {
		if !strings.Contains(logOutput, want) {
			t.Fatalf("expected log output to contain %q, got:\n%s", want, logOutput)
		}
	}
}

func TestFetchBlockReceiptsWithFallbackLogsWarning(t *testing.T) {
	logger, buf := testLogger(slog.LevelDebug)
	ctx := WithLogger(context.Background(), logger)
	source := &rpcSource{url: "https://rpc.example"}

	called := false
	got, err := fetchBlockReceiptsWithFallback(ctx, source, common.HexToHash("0x1234"), 2, func() ([]hexutil.Bytes, error) {
		called = true
		return []hexutil.Bytes{{0x01}}, nil
	})
	if err != nil {
		t.Fatalf("fetchBlockReceiptsWithFallback: %v", err)
	}
	if !called {
		t.Fatal("expected fallback to be called")
	}
	if len(got) != 1 {
		t.Fatalf("expected one receipt from fallback, got %d", len(got))
	}
	if !strings.Contains(buf.String(), "falling back to transaction scan") {
		t.Fatalf("expected warning log, got:\n%s", buf.String())
	}
}

func testLogger(level slog.Level) (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return attr
		},
	}))
	return logger, &buf
}
