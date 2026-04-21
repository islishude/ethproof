package proof

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

func TestLiveGenerateAndVerify(t *testing.T) {
	rpcs := splitEnvList("ETH_PROOF_RPCS")
	txHash := strings.TrimSpace(os.Getenv("ETH_PROOF_LIVE_TX"))
	logIndex := strings.TrimSpace(os.Getenv("ETH_PROOF_LIVE_LOG_INDEX"))
	blockNumber := strings.TrimSpace(os.Getenv("ETH_PROOF_LIVE_STATE_BLOCK"))
	account := strings.TrimSpace(os.Getenv("ETH_PROOF_LIVE_ACCOUNT"))
	slot := strings.TrimSpace(os.Getenv("ETH_PROOF_LIVE_SLOT"))
	if len(rpcs) == 0 || txHash == "" || logIndex == "" || blockNumber == "" || account == "" || slot == "" {
		t.Skip("set ETH_PROOF_RPCS, ETH_PROOF_LIVE_TX, ETH_PROOF_LIVE_LOG_INDEX, ETH_PROOF_LIVE_STATE_BLOCK, ETH_PROOF_LIVE_ACCOUNT, and ETH_PROOF_LIVE_SLOT to run live tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	verifyReq := VerifyRPCRequest{
		RPCURLs:       rpcs,
		MinRPCSources: defaultMinRPCSources,
	}

	txPkg, err := GenerateTransactionProof(ctx, TransactionProofRequest{
		RPCURLs: rpcs,
		TxHash:  common.HexToHash(txHash),
	})
	if err != nil {
		t.Fatalf("GenerateTransactionProof: %v", err)
	}
	if err := VerifyTransactionProofPackage(txPkg); err != nil {
		t.Fatalf("VerifyTransactionProofPackage: %v", err)
	}
	if err := VerifyTransactionProofPackageAgainstRPCs(ctx, txPkg, verifyReq); err != nil {
		t.Fatalf("VerifyTransactionProofPackageAgainstRPCs: %v", err)
	}

	receiptPkg, err := GenerateReceiptProof(ctx, ReceiptProofRequest{
		RPCURLs:  rpcs,
		TxHash:   common.HexToHash(txHash),
		LogIndex: mustParseUint(t, logIndex),
	})
	if err != nil {
		t.Fatalf("GenerateReceiptProof: %v", err)
	}
	if err := VerifyReceiptProofPackage(receiptPkg); err != nil {
		t.Fatalf("VerifyReceiptProofPackage: %v", err)
	}
	if err := VerifyReceiptProofPackageWithExpectationsAgainstRPCs(ctx, receiptPkg, nil, verifyReq); err != nil {
		t.Fatalf("VerifyReceiptProofPackageWithExpectationsAgainstRPCs: %v", err)
	}

	statePkg, err := GenerateStateProof(ctx, StateProofRequest{
		RPCURLs:     rpcs,
		BlockNumber: mustParseUint64(t, blockNumber),
		Account:     common.HexToAddress(account),
		Slot:        common.HexToHash(slot),
	})
	if err != nil {
		t.Fatalf("GenerateStateProof: %v", err)
	}
	if err := VerifyStateProofPackage(statePkg); err != nil {
		t.Fatalf("VerifyStateProofPackage: %v", err)
	}
	if err := VerifyStateProofPackageAgainstRPCs(ctx, statePkg, verifyReq); err != nil {
		t.Fatalf("VerifyStateProofPackageAgainstRPCs: %v", err)
	}
}

func splitEnvList(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func mustParseUint(t *testing.T, value string) uint {
	t.Helper()
	n, err := mustParseUint64Value(value)
	if err != nil {
		t.Fatalf("parse uint %q: %v", value, err)
	}
	return uint(n)
}

func mustParseUint64(t *testing.T, value string) uint64 {
	t.Helper()
	n, err := mustParseUint64Value(value)
	if err != nil {
		t.Fatalf("parse uint64 %q: %v", value, err)
	}
	return n
}

func mustParseUint64Value(value string) (uint64, error) {
	return strconv.ParseUint(strings.TrimSpace(value), 10, 64)
}
