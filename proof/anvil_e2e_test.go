package proof

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/islishude/ethproof/internal/e2e/bindings"
)

const (
	anvilDefaultRPCURL      = "http://127.0.0.1:8545"
	anvilDefaultPrivateKey  = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	anvilExpectedChainID    = 1337
	anvilReadyTimeout       = 5 * time.Second
	anvilPollInterval       = 500 * time.Millisecond
	proofDemoEventSignature = "ValueUpdated(address,bytes32,uint256)"
)

type anvilScenario struct {
	rpcURL          string
	blockNumber     uint64
	contractAddress common.Address
	txHash          common.Hash
	logIndex        uint
	slot            common.Hash
	marker          common.Hash
	caller          common.Address
	newValue        *big.Int
	eventData       []byte
	eventTopics     []common.Hash
}

func TestAnvilE2E(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client, rpcURL := requireAnvilClient(t, ctx)
	defer client.Close()

	scenario := deployProofDemoScenario(t, ctx, client, rpcURL)

	t.Run("api", func(t *testing.T) {
		testAnvilAPIFlow(t, ctx, scenario)
	})
	t.Run("cli", func(t *testing.T) {
		testAnvilCLIFlow(t, ctx, scenario)
	})
}

func testAnvilAPIFlow(t *testing.T, ctx context.Context, scenario anvilScenario) {
	t.Helper()

	verifyReq := VerifyRPCRequest{
		RPCURLs:       []string{scenario.rpcURL},
		MinRPCSources: 1,
	}

	txPkg, err := GenerateTransactionProof(ctx, TransactionProofRequest{
		RPCURLs:       []string{scenario.rpcURL},
		MinRPCSources: 1,
		TxHash:        scenario.txHash,
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
	txPkgWithTamperedConsensus := cloneTransactionPackage(*txPkg)
	txPkgWithTamperedConsensus.Block.SourceConsensus.RPCs = []string{"http://generate-rpc.invalid"}
	if err := VerifyTransactionProofPackageAgainstRPCs(ctx, &txPkgWithTamperedConsensus, verifyReq); err != nil {
		t.Fatalf("VerifyTransactionProofPackageAgainstRPCs with tampered generation rpc metadata: %v", err)
	}
	txPkgWithTamperedBlockHash := cloneTransactionPackage(*txPkg)
	txPkgWithTamperedBlockHash.Block.BlockHash = common.HexToHash("0x1234")
	if err := VerifyTransactionProofPackageAgainstRPCs(ctx, &txPkgWithTamperedBlockHash, verifyReq); err == nil {
		t.Fatal("expected tampered tx block hash to fail rpc-aware verification")
	}

	receiptPkg, err := GenerateReceiptProof(ctx, ReceiptProofRequest{
		RPCURLs:       []string{scenario.rpcURL},
		MinRPCSources: 1,
		TxHash:        scenario.txHash,
		LogIndex:      scenario.logIndex,
	})
	if err != nil {
		t.Fatalf("GenerateReceiptProof: %v", err)
	}
	receiptExpect := &ReceiptExpectations{
		Emitter: &scenario.contractAddress,
		Topics:  scenario.eventTopics,
		Data:    scenario.eventData,
	}
	if err := VerifyReceiptProofPackageWithExpectations(receiptPkg, receiptExpect); err != nil {
		t.Fatalf("VerifyReceiptProofPackageWithExpectations: %v", err)
	}
	if err := VerifyReceiptProofPackageWithExpectationsAgainstRPCs(ctx, receiptPkg, receiptExpect, verifyReq); err != nil {
		t.Fatalf("VerifyReceiptProofPackageWithExpectationsAgainstRPCs: %v", err)
	}

	statePkg, err := GenerateStateProof(ctx, StateProofRequest{
		RPCURLs:       []string{scenario.rpcURL},
		MinRPCSources: 1,
		BlockNumber:   scenario.blockNumber,
		Account:       scenario.contractAddress,
		Slot:          scenario.slot,
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
	if statePkg.StorageValue != common.BigToHash(scenario.newValue) {
		t.Fatalf("unexpected storage value: got %s want %s", statePkg.StorageValue, common.BigToHash(scenario.newValue))
	}
}

func testAnvilCLIFlow(t *testing.T, ctx context.Context, scenario anvilScenario) {
	t.Helper()

	root := repoRoot(t)
	tmp := t.TempDir()
	txProof := filepath.Join(tmp, "tx.json")
	receiptProof := filepath.Join(tmp, "receipt.json")
	stateProof := filepath.Join(tmp, "state.json")
	configPath := writeAnvilCLIConfig(t, tmp, scenario, txProof, receiptProof, stateProof)

	runEventproof(t, ctx, root, "generate", "tx", "--config", configPath)
	runEventproof(t, ctx, root, "verify", "tx", "--config", configPath)

	runEventproof(t, ctx, root, "generate", "receipt", "--config", configPath)
	runEventproof(t, ctx, root, "verify", "receipt", "--config", configPath)

	runEventproof(t, ctx, root, "generate", "state", "--config", configPath)
	runEventproof(t, ctx, root, "verify", "state", "--config", configPath)
}

func deployProofDemoScenario(t *testing.T, ctx context.Context, client *ethclient.Client, rpcURL string) anvilScenario {
	t.Helper()

	key, err := crypto.HexToECDSA(anvilDefaultPrivateKey)
	if err != nil {
		t.Fatalf("HexToECDSA: %v", err)
	}
	auth := mustTransactor(t, ctx, key)
	address, deployTx, contract, err := bindings.DeployProofDemo(auth, client)
	if err != nil {
		t.Fatalf("DeployProofDemo: %v", err)
	}
	deployReceipt, err := bind.WaitMined(ctx, client, deployTx)
	if err != nil {
		t.Fatalf("WaitMined(deploy): %v", err)
	}
	if deployReceipt.Status != types.ReceiptStatusSuccessful {
		t.Fatalf("deployment failed with status %d", deployReceipt.Status)
	}

	marker := common.HexToHash("0x0102030405060708090a0b0c0d0e0f100102030405060708090a0b0c0d0e0f10")
	newValue := big.NewInt(424242)
	setTx, err := contract.SetValue(mustTransactor(t, ctx, key), newValue, hashToBytes32(marker))
	if err != nil {
		t.Fatalf("SetValue: %v", err)
	}
	receipt, err := bind.WaitMined(ctx, client, setTx)
	if err != nil {
		t.Fatalf("WaitMined(setValue): %v", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		t.Fatalf("setValue failed with status %d", receipt.Status)
	}
	if len(receipt.Logs) == 0 {
		t.Fatal("setValue receipt did not contain logs")
	}

	event, err := contract.ParseValueUpdated(*receipt.Logs[0])
	if err != nil {
		t.Fatalf("ParseValueUpdated: %v", err)
	}
	storedValue, err := contract.Value(&bind.CallOpts{Context: ctx})
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	if storedValue.Cmp(newValue) != 0 {
		t.Fatalf("unexpected stored value: got %s want %s", storedValue, newValue)
	}

	callerTopic := common.BytesToHash(auth.From.Bytes())
	eventSigTopic := crypto.Keccak256Hash([]byte(proofDemoEventSignature))
	expectedData := common.LeftPadBytes(newValue.Bytes(), 32)
	if event.Caller != auth.From {
		t.Fatalf("unexpected event caller: got %s want %s", event.Caller, auth.From)
	}
	if event.Marker != hashToBytes32(marker) {
		t.Fatalf("unexpected event marker: got %x want %x", event.Marker, hashToBytes32(marker))
	}
	if event.Value.Cmp(newValue) != 0 {
		t.Fatalf("unexpected event value: got %s want %s", event.Value, newValue)
	}

	return anvilScenario{
		rpcURL:          rpcURL,
		blockNumber:     receipt.BlockNumber.Uint64(),
		contractAddress: address,
		txHash:          setTx.Hash(),
		logIndex:        0,
		slot:            common.Hash{},
		marker:          marker,
		caller:          auth.From,
		newValue:        newValue,
		eventData:       expectedData,
		eventTopics:     []common.Hash{eventSigTopic, callerTopic, marker},
	}
}

func requireAnvilClient(t *testing.T, ctx context.Context) (*ethclient.Client, string) {
	t.Helper()

	if os.Getenv("ETH_PROOF_REQUIRE_E2E") != "1" {
		t.Skipf("skipping anvil e2e")
		return nil, ""
	}

	rpcURL := strings.TrimSpace(os.Getenv("ETH_PROOF_E2E_RPC"))
	if rpcURL == "" {
		rpcURL = anvilDefaultRPCURL
	}

	newctx, cancel := context.WithTimeout(ctx, anvilReadyTimeout)
	defer cancel()

	ticker := time.NewTimer(0)
	defer ticker.Stop()

	for {
		select {
		case <-newctx.Done():
			t.Fatalf("timed out waiting for Anvil RPC at %s", rpcURL)
		case <-ticker.C:
			client, err := ethclient.DialContext(ctx, rpcURL)
			if err == nil {
				chainID, chainErr := client.ChainID(ctx)
				if chainErr == nil && chainID.Uint64() == anvilExpectedChainID {
					return client, rpcURL
				}
			}
			ticker.Reset(anvilPollInterval)
		}
	}
}

func mustTransactor(t *testing.T, ctx context.Context, key *ecdsa.PrivateKey) *bind.TransactOpts {
	t.Helper()

	auth, err := bind.NewKeyedTransactorWithChainID(key, big.NewInt(anvilExpectedChainID))
	if err != nil {
		t.Fatalf("NewKeyedTransactorWithChainID: %v", err)
	}
	auth.Context = ctx
	return auth
}

func runEventproof(t *testing.T, ctx context.Context, root string, args ...string) string {
	t.Helper()

	cmdArgs := append([]string{"run", "./cmd/ethproof"}, args...)
	cmd := exec.CommandContext(ctx, "go", cmdArgs...)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: go %s\n%s", strings.Join(cmdArgs, " "), string(output))
	}
	return string(output)
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to determine caller path")
	}
	return filepath.Dir(filepath.Dir(file))
}

func hashToBytes32(hash common.Hash) [32]byte {
	var out [32]byte
	copy(out[:], hash[:])
	return out
}

func writeAnvilCLIConfig(t *testing.T, dir string, scenario anvilScenario, txProof string, receiptProof string, stateProof string) string {
	t.Helper()

	topics := make([]string, 0, len(scenario.eventTopics))
	for _, topic := range scenario.eventTopics {
		topics = append(topics, topic.Hex())
	}
	config := map[string]any{
		"generate": map[string]any{
			"tx": map[string]any{
				"rpcs":    []string{scenario.rpcURL},
				"minRpcs": 1,
				"tx":      scenario.txHash.Hex(),
				"out":     txProof,
			},
			"receipt": map[string]any{
				"rpcs":     []string{scenario.rpcURL},
				"minRpcs":  1,
				"tx":       scenario.txHash.Hex(),
				"logIndex": scenario.logIndex,
				"out":      receiptProof,
			},
			"state": map[string]any{
				"rpcs":    []string{scenario.rpcURL},
				"minRpcs": 1,
				"block":   scenario.blockNumber,
				"account": scenario.contractAddress.Hex(),
				"slot":    scenario.slot.Hex(),
				"out":     stateProof,
			},
		},
		"verify": map[string]any{
			"tx": map[string]any{
				"rpcs":    []string{scenario.rpcURL},
				"minRpcs": 1,
				"proof":   txProof,
			},
			"receipt": map[string]any{
				"rpcs":          []string{scenario.rpcURL},
				"minRpcs":       1,
				"proof":         receiptProof,
				"expectEmitter": scenario.contractAddress.Hex(),
				"expectTopics":  topics,
				"expectData":    hexutil.Encode(scenario.eventData),
			},
			"state": map[string]any{
				"rpcs":    []string{scenario.rpcURL},
				"minRpcs": 1,
				"proof":   stateProof,
			},
		},
	}
	b, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("marshal cli config: %v", err)
	}
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatalf("write cli config: %v", err)
	}
	return path
}
