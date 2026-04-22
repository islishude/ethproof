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
	anvilDefaultRPCURL             = "http://127.0.0.1:8545"
	anvilDefaultPrivateKey         = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	anvilExpectedChainID           = 1337
	anvilReadyTimeout              = 5 * time.Second
	anvilPollInterval              = 500 * time.Millisecond
	anvilE2ETimeout                = 5 * time.Minute
	proofDemoEventSignature        = "ValueUpdated(address,bytes32,uint256)"
	proofComplexEventSignature     = "ComplexStateUpdated(address,uint256,bytes32,uint256,uint256,uint256,uint256)"
	proofComplexContractName       = "ProofComplexDemo"
	proofComplexNoteWord0          = "abcdefghijklmnopqrstuvwxyz123456"
	proofComplexHistoryTargetIndex = 2
)

type simpleAnvilScenario struct {
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

type complexStateTarget struct {
	Query         string
	ExpectedValue common.Hash
}

type complexAnvilScenario struct {
	rpcURL          string
	blockNumber     uint64
	contractAddress common.Address
	txHash          common.Hash
	logIndex        uint
	caller          common.Address
	marker          common.Hash
	positionID      *big.Int
	balanceValue    *big.Int
	historyValue    *big.Int
	quantity        *big.Int
	lastPrice       *big.Int
	note            string
	payload         []byte
	eventLog        types.Log
	eventData       []byte
	eventTopics     []common.Hash
	stateTargets    []complexStateTarget
}

func TestAnvilE2E(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), anvilE2ETimeout)
	defer cancel()

	client, rpcURL := requireAnvilClient(t, ctx)
	defer client.Close()

	t.Run("simple", func(t *testing.T) {
		scenario := deployProofDemoScenario(t, ctx, client, rpcURL)

		t.Run("api", func(t *testing.T) {
			testSimpleAnvilAPIFlow(t, ctx, scenario)
		})
		t.Run("cli", func(t *testing.T) {
			testSimpleAnvilCLIFlow(t, ctx, scenario)
		})
	})

	t.Run("complex", func(t *testing.T) {
		scenario := deployProofComplexScenario(t, ctx, client, rpcURL)

		t.Run("behavior_and_resolver", func(t *testing.T) {
			testComplexAnvilBehaviorAndResolver(t, ctx, client, scenario)
		})
		t.Run("proof_api", func(t *testing.T) {
			testComplexAnvilProofAPIFlow(t, ctx, scenario)
		})
		t.Run("proof_cli", func(t *testing.T) {
			testComplexAnvilProofCLIFlow(t, ctx, scenario)
		})
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

func testSimpleAnvilCLIFlow(t *testing.T, ctx context.Context, scenario simpleAnvilScenario) {
	t.Helper()

	root := repoRoot(t)
	tmp := t.TempDir()
	txProof := filepath.Join(tmp, "tx.json")
	receiptProof := filepath.Join(tmp, "receipt.json")
	stateProof := filepath.Join(tmp, "state.json")
	configPath := writeAnvilCLIConfig(
		t,
		tmp,
		scenario.rpcURL,
		scenario.blockNumber,
		scenario.contractAddress,
		scenario.txHash,
		scenario.logIndex,
		[]common.Hash{scenario.slot},
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
}

func testComplexAnvilBehaviorAndResolver(t *testing.T, ctx context.Context, client *ethclient.Client, scenario complexAnvilScenario) {
	t.Helper()

	contract, err := bindings.NewProofComplexDemo(scenario.contractAddress, client)
	if err != nil {
		t.Fatalf("NewProofComplexDemo: %v", err)
	}

	balance, err := contract.Balances(&bind.CallOpts{Context: ctx}, scenario.caller)
	if err != nil {
		t.Fatalf("Balances: %v", err)
	}
	if balance.Cmp(scenario.balanceValue) != 0 {
		t.Fatalf("unexpected balance: got %s want %s", balance, scenario.balanceValue)
	}

	historyLength, err := contract.HistoryLength(&bind.CallOpts{Context: ctx}, scenario.caller)
	if err != nil {
		t.Fatalf("HistoryLength: %v", err)
	}
	if got, want := historyLength.Uint64(), uint64(proofComplexHistoryTargetIndex+1); got != want {
		t.Fatalf("unexpected history length: got %d want %d", got, want)
	}

	historyValue, err := contract.HistoryAt(&bind.CallOpts{Context: ctx}, scenario.caller, big.NewInt(proofComplexHistoryTargetIndex))
	if err != nil {
		t.Fatalf("HistoryAt: %v", err)
	}
	if historyValue.Cmp(scenario.historyValue) != 0 {
		t.Fatalf("unexpected history value: got %s want %s", historyValue, scenario.historyValue)
	}

	position, err := contract.PositionOf(&bind.CallOpts{Context: ctx}, scenario.caller, scenario.positionID)
	if err != nil {
		t.Fatalf("PositionOf: %v", err)
	}
	if position.Quantity.Cmp(scenario.quantity) != 0 {
		t.Fatalf("unexpected quantity: got %s want %s", position.Quantity, scenario.quantity)
	}
	if position.LastPrice.Cmp(scenario.lastPrice) != 0 {
		t.Fatalf("unexpected last price: got %s want %s", position.LastPrice, scenario.lastPrice)
	}

	note, err := contract.NoteText(&bind.CallOpts{Context: ctx})
	if err != nil {
		t.Fatalf("NoteText: %v", err)
	}
	if note != scenario.note {
		t.Fatalf("unexpected note text: got %q want %q", note, scenario.note)
	}

	payload, err := contract.PayloadData(&bind.CallOpts{Context: ctx})
	if err != nil {
		t.Fatalf("PayloadData: %v", err)
	}
	if string(payload) != string(scenario.payload) {
		t.Fatalf("unexpected payload: got %x want %x", payload, scenario.payload)
	}

	if len(scenario.eventLog.Topics) != len(scenario.eventTopics) {
		t.Fatalf("unexpected event topic count: got %d want %d", len(scenario.eventLog.Topics), len(scenario.eventTopics))
	}
	for i := range scenario.eventTopics {
		if scenario.eventLog.Topics[i] != scenario.eventTopics[i] {
			t.Fatalf("unexpected event topic[%d]: got %s want %s", i, scenario.eventLog.Topics[i], scenario.eventTopics[i])
		}
	}
	if string(scenario.eventLog.Data) != string(scenario.eventData) {
		t.Fatalf("unexpected event data: got %x want %x", scenario.eventLog.Data, scenario.eventData)
	}

	event, err := contract.ParseComplexStateUpdated(scenario.eventLog)
	if err != nil {
		t.Fatalf("ParseComplexStateUpdated: %v", err)
	}
	if event.Caller != scenario.caller {
		t.Fatalf("unexpected event caller: got %s want %s", event.Caller, scenario.caller)
	}
	if event.PositionId.Cmp(scenario.positionID) != 0 {
		t.Fatalf("unexpected event positionId: got %s want %s", event.PositionId, scenario.positionID)
	}
	if event.Marker != hashToBytes32(scenario.marker) {
		t.Fatalf("unexpected event marker: got %x want %x", event.Marker, hashToBytes32(scenario.marker))
	}
	if event.Balance.Cmp(scenario.balanceValue) != 0 {
		t.Fatalf("unexpected event balance: got %s want %s", event.Balance, scenario.balanceValue)
	}
	if event.HistoryValue.Cmp(scenario.historyValue) != 0 {
		t.Fatalf("unexpected event historyValue: got %s want %s", event.HistoryValue, scenario.historyValue)
	}
	if event.Quantity.Cmp(scenario.quantity) != 0 {
		t.Fatalf("unexpected event quantity: got %s want %s", event.Quantity, scenario.quantity)
	}
	if event.LastPrice.Cmp(scenario.lastPrice) != 0 {
		t.Fatalf("unexpected event lastPrice: got %s want %s", event.LastPrice, scenario.lastPrice)
	}

	layout := mustLoadComplexDemoStorageLayout(t, repoRoot(t))
	for _, query := range []string{"note", "payload"} {
		resolution, err := ResolveStorageSlots(layout, query)
		if err != nil {
			t.Fatalf("ResolveStorageSlots(%q): %v", query, err)
		}
		if resolution.Encoding != "bytes" {
			t.Fatalf("unexpected encoding for %s: got %s want bytes", query, resolution.Encoding)
		}
		if got, want := len(resolution.Slots), 1; got != want {
			t.Fatalf("unexpected slot count for %s: got %d want %d", query, got, want)
		}
	}
	for _, target := range scenario.stateTargets {
		resolution, err := ResolveStorageSlots(layout, target.Query)
		if err != nil {
			t.Fatalf("ResolveStorageSlots(%q): %v", target.Query, err)
		}
		if got, want := len(resolution.Slots), 1; got != want {
			t.Fatalf("unexpected slot count for %s: got %d want %d", target.Query, got, want)
		}
	}
}

func testComplexAnvilProofAPIFlow(t *testing.T, ctx context.Context, scenario complexAnvilScenario) {
	t.Helper()

	verifyReq := VerifyRPCRequest{
		RPCURLs:       []string{scenario.rpcURL},
		MinRPCSources: 1,
	}
	slots := resolveComplexStateSlotsViaAPI(t, repoRoot(t), scenario.stateTargets)

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
		Slots:         slots,
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
	assertStateProofMatchesTargets(t, statePkg, slots, scenario.stateTargets)
}

func testComplexAnvilProofCLIFlow(t *testing.T, ctx context.Context, scenario complexAnvilScenario) {
	t.Helper()

	root := repoRoot(t)
	compilerOutput := mustComplexDemoArtifactPath(t, root)
	slots := resolveComplexStateSlotsViaCLI(t, ctx, root, compilerOutput, scenario.stateTargets)

	tmp := t.TempDir()
	txProof := filepath.Join(tmp, "tx.json")
	receiptProof := filepath.Join(tmp, "receipt.json")
	stateProof := filepath.Join(tmp, "state.json")
	configPath := writeAnvilCLIConfig(
		t,
		tmp,
		scenario.rpcURL,
		scenario.blockNumber,
		scenario.contractAddress,
		scenario.txHash,
		scenario.logIndex,
		slots,
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

	var statePkg StateProofPackage
	if err := LoadJSON(stateProof, &statePkg); err != nil {
		t.Fatalf("LoadJSON(state proof): %v", err)
	}
	assertStateProofMatchesTargets(t, &statePkg, slots, scenario.stateTargets)
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
		marker:          marker,
		caller:          auth.From,
		newValue:        newValue,
		eventData:       expectedData,
		eventTopics:     []common.Hash{eventSigTopic, callerTopic, marker},
	}
}

func deployProofComplexScenario(t *testing.T, ctx context.Context, client *ethclient.Client, rpcURL string) complexAnvilScenario {
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
	note := proofComplexNoteWord0
	noteBytes := []byte(note)
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
		note,
		payload,
		hashToBytes32(marker),
	)
	if err != nil {
		t.Fatalf("ApplyUpdate: %v", err)
	}
	receipt, err := bind.WaitMined(ctx, client, updateTx)
	if err != nil {
		t.Fatalf("WaitMined(applyUpdate): %v", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		t.Fatalf("applyUpdate failed with status %d", receipt.Status)
	}
	if len(receipt.Logs) == 0 {
		t.Fatal("applyUpdate receipt did not contain logs")
	}

	callerTopic := common.BytesToHash(auth.From.Bytes())
	positionTopic := common.BigToHash(positionID)
	eventSigTopic := crypto.Keccak256Hash([]byte(proofComplexEventSignature))
	expectedData := encodeUint256Data(balanceValue, historyValue, quantity, lastPrice)

	return complexAnvilScenario{
		rpcURL:          rpcURL,
		blockNumber:     receipt.BlockNumber.Uint64(),
		contractAddress: address,
		txHash:          updateTx.Hash(),
		logIndex:        0,
		caller:          auth.From,
		marker:          marker,
		positionID:      new(big.Int).Set(positionID),
		balanceValue:    new(big.Int).Set(balanceValue),
		historyValue:    new(big.Int).Set(historyValue),
		quantity:        new(big.Int).Set(quantity),
		lastPrice:       new(big.Int).Set(lastPrice),
		note:            note,
		payload:         common.CopyBytes(payload),
		eventLog:        *receipt.Logs[0],
		eventData:       expectedData,
		eventTopics:     []common.Hash{eventSigTopic, callerTopic, positionTopic, marker},
		stateTargets: []complexStateTarget{
			{
				Query:         "balances[" + auth.From.Hex() + "]",
				ExpectedValue: common.BigToHash(balanceValue),
			},
			{
				Query:         "history[" + auth.From.Hex() + "][2]",
				ExpectedValue: common.BigToHash(historyValue),
			},
			{
				Query:         "positions[" + auth.From.Hex() + "][" + positionID.String() + "].quantity",
				ExpectedValue: common.BigToHash(quantity),
			},
			{
				Query:         "positions[" + auth.From.Hex() + "][" + positionID.String() + "].lastPrice",
				ExpectedValue: common.BigToHash(lastPrice),
			},
			{
				Query:         "note@word(0)",
				ExpectedValue: common.BytesToHash(noteBytes),
			},
			{
				Query:         "payload@word(0)",
				ExpectedValue: common.BytesToHash(payload),
			},
		},
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

func encodeUint256Data(values ...*big.Int) []byte {
	out := make([]byte, 0, len(values)*32)
	for _, value := range values {
		out = append(out, common.LeftPadBytes(value.Bytes(), 32)...)
	}
	return out
}

func mustLoadComplexDemoStorageLayout(t *testing.T, root string) *StorageLayout {
	t.Helper()

	layout, err := LoadStorageLayout(mustComplexDemoArtifactPath(t, root), proofComplexContractName, StorageLayoutFormatArtifact)
	if err != nil {
		t.Fatalf("LoadStorageLayout: %v", err)
	}
	return layout
}

func mustComplexDemoArtifactPath(t *testing.T, root string) string {
	t.Helper()

	path := filepath.Join(root, "out", "ProofComplexDemo.sol", "ProofComplexDemo.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("missing Foundry artifact %s: run make bindings or forge build first", path)
	}
	return path
}

func resolveComplexStateSlotsViaAPI(t *testing.T, root string, targets []complexStateTarget) []common.Hash {
	t.Helper()

	layout := mustLoadComplexDemoStorageLayout(t, root)
	slots := make([]common.Hash, len(targets))
	for i, target := range targets {
		resolution, err := ResolveStorageSlots(layout, target.Query)
		if err != nil {
			t.Fatalf("ResolveStorageSlots(%q): %v", target.Query, err)
		}
		if len(resolution.Slots) != 1 {
			t.Fatalf("ResolveStorageSlots(%q) returned %d slots, want 1", target.Query, len(resolution.Slots))
		}
		slots[i] = resolution.Slots[0].Slot
	}
	return slots
}

func resolveComplexStateSlotsViaCLI(t *testing.T, ctx context.Context, root string, compilerOutput string, targets []complexStateTarget) []common.Hash {
	t.Helper()

	slots := make([]common.Hash, len(targets))
	for i, target := range targets {
		output := runEventproof(
			t,
			ctx,
			root,
			"resolve",
			"slot",
			"--compiler-output",
			compilerOutput,
			"--contract",
			proofComplexContractName,
			"--format",
			"artifact",
			"--var",
			target.Query,
		)

		var resolution StorageSlotResolution
		if err := json.Unmarshal([]byte(output), &resolution); err != nil {
			t.Fatalf("unmarshal resolve slot output for %s: %v\noutput=%s", target.Query, err, output)
		}
		if len(resolution.Slots) != 1 {
			t.Fatalf("resolve slot for %s returned %d slots, want 1", target.Query, len(resolution.Slots))
		}
		slots[i] = resolution.Slots[0].Slot
	}
	return slots
}

func assertStateProofMatchesTargets(t *testing.T, pkg *StateProofPackage, slots []common.Hash, targets []complexStateTarget) {
	t.Helper()

	if got, want := len(pkg.StorageProofs), len(targets); got != want {
		t.Fatalf("unexpected storage proof count: got %d want %d", got, want)
	}
	for i, target := range targets {
		if pkg.StorageProofs[i].Slot != slots[i] {
			t.Fatalf("unexpected storage proof slot[%d] for %s: got %s want %s", i, target.Query, pkg.StorageProofs[i].Slot, slots[i])
		}
		if pkg.StorageProofs[i].Value != target.ExpectedValue {
			t.Fatalf("unexpected storage proof value[%d] for %s: got %s want %s", i, target.Query, pkg.StorageProofs[i].Value, target.ExpectedValue)
		}
	}
}
