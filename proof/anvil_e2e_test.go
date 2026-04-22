package proof

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
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
	anvilDefaultRPCURL       = "http://127.0.0.1:8545"
	anvilDefaultPrivateKey   = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	anvilExpectedChainID     = 1337
	anvilReadyTimeout        = 5 * time.Second
	anvilPollInterval        = 500 * time.Millisecond
	anvilE2ETimeout          = 5 * time.Minute
	proofDemoEventSignature  = "ValueUpdated(address,bytes32,uint256)"
	proofDemoContractName    = "ProofDemo"
	proofComplexContractName = "ProofComplexDemo"
	proofComplexNoteWord0    = "abcdefghijklmnopqrstuvwxyz123456"
)

type simpleAnvilScenario struct {
	rpcURL          string
	blockNumber     uint64
	contractAddress common.Address
	txHash          common.Hash
	logIndex        uint
	slot            common.Hash
	newValue        *big.Int
	eventData       []byte
	eventTopics     []common.Hash
}

type complexResolveTarget struct {
	query        string
	expectedWord common.Hash
}

type complexResolveScenario struct {
	rpcURL          string
	blockNumber     uint64
	contractAddress common.Address
	caller          common.Address
	positionID      *big.Int
	targets         []complexResolveTarget
}

func TestAnvilE2E(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), anvilE2ETimeout)
	defer cancel()

	client, rpcURL := requireAnvilClient(t, ctx)
	defer client.Close()

	scenario := deployProofDemoScenario(t, ctx, client, rpcURL)

	t.Run("api_mainline", func(t *testing.T) {
		testSimpleAnvilAPIFlow(t, ctx, scenario)
	})
	t.Run("cli_mainline", func(t *testing.T) {
		testAnvilCLIFlow(t, ctx, client, scenario)
	})
}

func testSimpleAnvilAPIFlow(t *testing.T, ctx context.Context, scenario simpleAnvilScenario) {
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

	receiptExpect := &ReceiptExpectations{
		Emitter: &scenario.contractAddress,
		Topics:  scenario.eventTopics,
		Data:    scenario.eventData,
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
		Slots:         []common.Hash{scenario.slot},
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
	if got, want := len(statePkg.StorageProofs), 1; got != want {
		t.Fatalf("unexpected storage proof count: got %d want %d", got, want)
	}
	if statePkg.StorageProofs[0].Value != common.BigToHash(scenario.newValue) {
		t.Fatalf("unexpected storage value: got %s want %s", statePkg.StorageProofs[0].Value, common.BigToHash(scenario.newValue))
	}
}

func testAnvilCLIFlow(t *testing.T, ctx context.Context, client *ethclient.Client, scenario simpleAnvilScenario) {
	t.Helper()

	root := repoRoot(t)
	tmp := t.TempDir()
	slotPath := filepath.Join(tmp, "slot.json")
	txProof := filepath.Join(tmp, "tx.json")
	receiptProof := filepath.Join(tmp, "receipt.json")
	stateProof := filepath.Join(tmp, "state.json")

	runEventproof(t, ctx, root,
		"resolve", "slot",
		"--compiler-output", mustProofDemoArtifactPath(t, root),
		"--contract", proofDemoContractName,
		"--format", "artifact",
		"--var", "value",
		"--out", slotPath,
	)

	var resolution StorageSlotResolution
	if err := LoadJSON(slotPath, &resolution); err != nil {
		t.Fatalf("LoadJSON(slot): %v", err)
	}
	if got, want := len(resolution.Slots), 1; got != want {
		t.Fatalf("unexpected resolved slot count: got %d want %d", got, want)
	}
	if resolution.Slots[0].Slot != scenario.slot {
		t.Fatalf("unexpected resolved slot: got %s want %s", resolution.Slots[0].Slot, scenario.slot)
	}

	configPath := writeAnvilCLIConfig(
		t,
		tmp,
		scenario.rpcURL,
		scenario.blockNumber,
		scenario.contractAddress,
		scenario.txHash,
		scenario.logIndex,
		[]common.Hash{resolution.Slots[0].Slot},
		scenario.contractAddress,
		scenario.eventTopics,
		scenario.eventData,
		txProof,
		receiptProof,
		stateProof,
	)

	runEventproof(t, ctx, root, "generate", "tx", "--config", configPath)
	runEventproof(t, ctx, root, "verify", "tx", "--config", configPath)
	runEventproof(t, ctx, root, "generate", "receipt", "--config", configPath)
	runEventproof(t, ctx, root, "verify", "receipt", "--config", configPath)
	runEventproof(t, ctx, root, "generate", "state", "--config", configPath)
	runEventproof(t, ctx, root, "verify", "state", "--config", configPath)

	complexScenario := deployProofComplexResolveScenario(t, ctx, client, scenario.rpcURL)
	testComplexResolveCLIRegression(t, ctx, client, complexScenario)
}

func deployProofDemoScenario(t *testing.T, ctx context.Context, client *ethclient.Client, rpcURL string) simpleAnvilScenario {
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
	expectedData := encodeUint256Data(newValue)
	if event.Caller != auth.From {
		t.Fatalf("unexpected event caller: got %s want %s", event.Caller, auth.From)
	}
	if event.Marker != hashToBytes32(marker) {
		t.Fatalf("unexpected event marker: got %x want %x", event.Marker, hashToBytes32(marker))
	}
	if event.Value.Cmp(newValue) != 0 {
		t.Fatalf("unexpected event value: got %s want %s", event.Value, newValue)
	}

	return simpleAnvilScenario{
		rpcURL:          rpcURL,
		blockNumber:     receipt.BlockNumber.Uint64(),
		contractAddress: address,
		txHash:          setTx.Hash(),
		logIndex:        0,
		slot:            common.Hash{},
		newValue:        newValue,
		eventData:       expectedData,
		eventTopics:     []common.Hash{eventSigTopic, callerTopic, marker},
	}
}

func deployProofComplexResolveScenario(t *testing.T, ctx context.Context, client *ethclient.Client, rpcURL string) complexResolveScenario {
	t.Helper()

	key, err := crypto.HexToECDSA(anvilDefaultPrivateKey)
	if err != nil {
		t.Fatalf("HexToECDSA: %v", err)
	}
	auth := mustTransactor(t, ctx, key)
	address, deployTx, contract, err := bindings.DeployProofComplexDemo(auth, client)
	if err != nil {
		t.Fatalf("DeployProofComplexDemo: %v", err)
	}
	deployReceipt, err := bind.WaitMined(ctx, client, deployTx)
	if err != nil {
		t.Fatalf("WaitMined(deploy complex): %v", err)
	}
	if deployReceipt.Status != types.ReceiptStatusSuccessful {
		t.Fatalf("complex deployment failed with status %d", deployReceipt.Status)
	}

	historySeed := []*big.Int{
		big.NewInt(111),
		big.NewInt(222),
	}
	seedTx, err := contract.SeedHistory(mustTransactor(t, ctx, key), auth.From, historySeed)
	if err != nil {
		t.Fatalf("SeedHistory: %v", err)
	}
	seedReceipt, err := bind.WaitMined(ctx, client, seedTx)
	if err != nil {
		t.Fatalf("WaitMined(seedHistory): %v", err)
	}
	if seedReceipt.Status != types.ReceiptStatusSuccessful {
		t.Fatalf("seedHistory failed with status %d", seedReceipt.Status)
	}

	balanceValue := big.NewInt(777777)
	positionID := big.NewInt(7)
	historyValue := big.NewInt(333)
	quantity := big.NewInt(444)
	lastPrice := big.NewInt(555)
	noteBytes := []byte(proofComplexNoteWord0)
	if len(noteBytes) != 32 {
		t.Fatalf("proofComplexNoteWord0 must be 32 bytes, got %d", len(noteBytes))
	}
	payload := common.FromHex("0x000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
	if len(payload) != 32 {
		t.Fatalf("complex payload must be 32 bytes, got %d", len(payload))
	}
	marker := common.HexToHash("0xf1f2f3f4f5f6f7f8f9fafbfcfdfeff000102030405060708090a0b0c0d0e0f10")
	updateTx, err := contract.ApplyUpdate(
		mustTransactor(t, ctx, key),
		balanceValue,
		positionID,
		historyValue,
		quantity,
		lastPrice,
		proofComplexNoteWord0,
		payload,
		hashToBytes32(marker),
	)
	if err != nil {
		t.Fatalf("ApplyUpdate: %v", err)
	}
	updateReceipt, err := bind.WaitMined(ctx, client, updateTx)
	if err != nil {
		t.Fatalf("WaitMined(applyUpdate): %v", err)
	}
	if updateReceipt.Status != types.ReceiptStatusSuccessful {
		t.Fatalf("applyUpdate failed with status %d", updateReceipt.Status)
	}

	caller := auth.From
	return complexResolveScenario{
		rpcURL:          rpcURL,
		blockNumber:     updateReceipt.BlockNumber.Uint64(),
		contractAddress: address,
		caller:          caller,
		positionID:      new(big.Int).Set(positionID),
		targets: []complexResolveTarget{
			{
				query:        "balances[" + caller.Hex() + "]",
				expectedWord: common.BigToHash(balanceValue),
			},
			{
				query:        "history[" + caller.Hex() + "][2]",
				expectedWord: common.BigToHash(historyValue),
			},
			{
				query:        "positions[" + caller.Hex() + "][" + positionID.String() + "].lastPrice",
				expectedWord: common.BigToHash(lastPrice),
			},
			{
				query:        "note@word(0)",
				expectedWord: common.BytesToHash(noteBytes),
			},
			{
				query:        "payload@word(0)",
				expectedWord: common.BytesToHash(payload),
			},
		},
	}
}

func requireAnvilClient(t *testing.T, ctx context.Context) (*ethclient.Client, string) {
	t.Helper()

	if os.Getenv("ETH_PROOF_REQUIRE_E2E") != "1" {
		t.Skip("skipping anvil e2e")
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
				client.Close()
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

func hashToBytes32(hash common.Hash) [32]byte {
	var out [32]byte
	copy(out[:], hash[:])
	return out
}

func writeAnvilCLIConfig(
	t *testing.T,
	dir string,
	rpcURL string,
	blockNumber uint64,
	account common.Address,
	txHash common.Hash,
	logIndex uint,
	slots []common.Hash,
	expectEmitter common.Address,
	eventTopics []common.Hash,
	eventData []byte,
	txProof string,
	receiptProof string,
	stateProof string,
) string {
	t.Helper()

	topics := make([]string, 0, len(eventTopics))
	for _, topic := range eventTopics {
		topics = append(topics, topic.Hex())
	}
	slotHexes := make([]string, 0, len(slots))
	for _, slot := range slots {
		slotHexes = append(slotHexes, slot.Hex())
	}

	config := map[string]any{
		"generate": map[string]any{
			"tx": map[string]any{
				"rpcs":    []string{rpcURL},
				"minRpcs": 1,
				"tx":      txHash.Hex(),
				"out":     txProof,
			},
			"receipt": map[string]any{
				"rpcs":     []string{rpcURL},
				"minRpcs":  1,
				"tx":       txHash.Hex(),
				"logIndex": logIndex,
				"out":      receiptProof,
			},
			"state": map[string]any{
				"rpcs":    []string{rpcURL},
				"minRpcs": 1,
				"block":   blockNumber,
				"account": account.Hex(),
				"slots":   slotHexes,
				"out":     stateProof,
			},
		},
		"verify": map[string]any{
			"tx": map[string]any{
				"rpcs":    []string{rpcURL},
				"minRpcs": 1,
				"proof":   txProof,
			},
			"receipt": map[string]any{
				"rpcs":          []string{rpcURL},
				"minRpcs":       1,
				"proof":         receiptProof,
				"expectEmitter": expectEmitter.Hex(),
				"expectTopics":  topics,
				"expectData":    hexutil.Encode(eventData),
			},
			"state": map[string]any{
				"rpcs":    []string{rpcURL},
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

func testComplexResolveCLIRegression(t *testing.T, ctx context.Context, client *ethclient.Client, scenario complexResolveScenario) {
	t.Helper()

	root := repoRoot(t)
	artifactPath := mustProofComplexArtifactPath(t, root)
	for _, target := range scenario.targets {
		resolution := runResolveSlotCLI(t, ctx, root, artifactPath, proofComplexContractName, target.query)
		assertResolvedSlotMatchesStorageAt(t, ctx, client, scenario, resolution, target)
	}
}

func runResolveSlotCLI(t *testing.T, ctx context.Context, root string, compilerOutput string, contract string, query string) StorageSlotResolution {
	t.Helper()

	output := runEventproof(
		t,
		ctx,
		root,
		"resolve", "slot",
		"--compiler-output", compilerOutput,
		"--contract", contract,
		"--format", "artifact",
		"--var", query,
	)

	var resolution StorageSlotResolution
	if err := json.Unmarshal([]byte(output), &resolution); err != nil {
		t.Fatalf("unmarshal resolve output for %q: %v\noutput=%s", query, err, output)
	}
	return resolution
}

func assertResolvedSlotMatchesStorageAt(
	t *testing.T,
	ctx context.Context,
	client *ethclient.Client,
	scenario complexResolveScenario,
	resolution StorageSlotResolution,
	target complexResolveTarget,
) {
	t.Helper()

	if got, want := len(resolution.Slots), 1; got != want {
		t.Fatalf("%s: unexpected resolved slot count: got %d want %d", target.query, got, want)
	}
	if resolution.Slots[0].Label != target.query {
		t.Fatalf("%s: unexpected resolved slot label: got %s", target.query, resolution.Slots[0].Label)
	}

	blockNumber := new(big.Int).SetUint64(scenario.blockNumber)
	storageWord, err := client.StorageAt(ctx, scenario.contractAddress, resolution.Slots[0].Slot, blockNumber)
	if err != nil {
		t.Fatalf("%s: StorageAt(%s): %v", target.query, resolution.Slots[0].Slot, err)
	}
	if got := common.BytesToHash(storageWord); got != target.expectedWord {
		t.Fatalf("%s: unexpected storage word: got %s want %s", target.query, got, target.expectedWord)
	}
}

func encodeUint256Data(values ...*big.Int) []byte {
	out := make([]byte, 0, len(values)*32)
	for _, value := range values {
		out = append(out, common.LeftPadBytes(value.Bytes(), 32)...)
	}
	return out
}

func mustProofDemoArtifactPath(t *testing.T, root string) string {
	t.Helper()

	path := filepath.Join(root, "out", "ProofDemo.sol", "ProofDemo.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("missing Foundry artifact %s: run make bindings or forge build first", path)
	}
	return path
}

func mustProofComplexArtifactPath(t *testing.T, root string) string {
	t.Helper()

	path := filepath.Join(root, "out", "ProofComplexDemo.sol", "ProofComplexDemo.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("missing Foundry artifact %s: run make bindings or forge build first", path)
	}
	return path
}
