package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/islishude/ethproof/proof"
)

type multiStringFlag []string

type generateStateConfig struct {
	Request proof.StateProofRequest
	Out     string
}

type generateReceiptConfig struct {
	Request proof.ReceiptProofRequest
	Out     string
}

type generateTransactionConfig struct {
	Request proof.TransactionProofRequest
	Out     string
}

type verifyStateConfig struct {
	ProofPath     string
	VerifyRequest proof.VerifyRPCRequest
}

type verifyReceiptConfig struct {
	ProofPath     string
	Expectations  *proof.ReceiptExpectations
	VerifyRequest proof.VerifyRPCRequest
}

type verifyTransactionConfig struct {
	ProofPath     string
	VerifyRequest proof.VerifyRPCRequest
}

func (m *multiStringFlag) String() string {
	return fmt.Sprintf("%v", []string(*m))
}

func (m *multiStringFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func parseGenerateStateArgs(args []string) (generateStateConfig, error) {
	fs := newFlagSet("generate state")
	configPath := fs.String("config", "", "config json file")
	var rpcURLs multiStringFlag
	fs.Var(&rpcURLs, "rpc", "Ethereum RPC URL")
	minRPCs := fs.Int("min-rpcs", proofMinRPCsDefault(), "minimum distinct RPC sources required")
	blockNumber := fs.Uint64("block", 0, "block number")
	accountHex := fs.String("account", "", "account address")
	slotHex := fs.String("slot", "", "32-byte storage slot key")
	out := fs.String("out", "state.json", "output proof json")
	if err := parseFlagSet(fs, args); err != nil {
		if _, ok := asUsageError(err); ok {
			return generateStateConfig{}, err
		}
		return generateStateConfig{}, newUsageError("parse generate state args: %v", err)
	}
	if err := ensureNoPositionalArgs(fs); err != nil {
		return generateStateConfig{}, err
	}

	seen := visitedFlags(fs)
	fileCfg, err := loadCLIConfig(*configPath)
	if err != nil {
		return generateStateConfig{}, newUsageError("%v", err)
	}

	var section *generateStateConfigFile
	if fileCfg != nil {
		section = fileCfg.Generate.State
	}
	cfg := generateStateConfig{
		Request: proof.StateProofRequest{
			RPCURLs:       mergeStringSlice(seen, "rpc", rpcURLs, nil),
			MinRPCSources: mergeInt(seen, "min-rpcs", *minRPCs, nil, proofMinRPCsDefault()),
			BlockNumber:   mergeUint64(seen, "block", *blockNumber, nil, 0),
		},
		Out: mergeString(seen, "out", *out, "", "state.json"),
	}
	rawAccount := mergeString(seen, "account", *accountHex, "", "")
	rawSlot := mergeString(seen, "slot", *slotHex, "", "")
	if section != nil {
		cfg.Request.RPCURLs = mergeStringSlice(seen, "rpc", rpcURLs, section.RPCs)
		cfg.Request.MinRPCSources = mergeInt(seen, "min-rpcs", *minRPCs, section.MinRPCs, proofMinRPCsDefault())
		cfg.Request.BlockNumber = mergeUint64(seen, "block", *blockNumber, section.Block, 0)
		rawAccount = mergeString(seen, "account", *accountHex, section.Account, "")
		rawSlot = mergeString(seen, "slot", *slotHex, section.Slot, "")
		cfg.Out = mergeString(seen, "out", *out, section.Out, "state.json")
	}
	if err := validateRPCInputs(cfg.Request.RPCURLs, cfg.Request.MinRPCSources, "generate state requires at least one RPC via --rpc or generate.state.rpcs in --config"); err != nil {
		return generateStateConfig{}, err
	}
	if rawAccount == "" {
		return generateStateConfig{}, newUsageError("generate state requires --account or generate.state.account in --config")
	}
	if rawSlot == "" {
		return generateStateConfig{}, newUsageError("generate state requires --slot or generate.state.slot in --config")
	}
	cfg.Request.Account = common.HexToAddress(rawAccount)
	cfg.Request.Slot = common.HexToHash(rawSlot)
	return cfg, nil
}

func parseGenerateReceiptArgs(args []string) (generateReceiptConfig, error) {
	fs := newFlagSet("generate receipt")
	configPath := fs.String("config", "", "config json file")
	var rpcURLs multiStringFlag
	fs.Var(&rpcURLs, "rpc", "Ethereum RPC URL")
	minRPCs := fs.Int("min-rpcs", proofMinRPCsDefault(), "minimum distinct RPC sources required")
	txHashHex := fs.String("tx", "", "transaction hash")
	logIndex := fs.Uint("log-index", 0, "log index within receipt")
	out := fs.String("out", "receipt.json", "output proof json")
	if err := parseFlagSet(fs, args); err != nil {
		if _, ok := asUsageError(err); ok {
			return generateReceiptConfig{}, err
		}
		return generateReceiptConfig{}, newUsageError("parse generate receipt args: %v", err)
	}
	if err := ensureNoPositionalArgs(fs); err != nil {
		return generateReceiptConfig{}, err
	}

	seen := visitedFlags(fs)
	fileCfg, err := loadCLIConfig(*configPath)
	if err != nil {
		return generateReceiptConfig{}, newUsageError("%v", err)
	}

	var section *generateReceiptConfigFile
	if fileCfg != nil {
		section = fileCfg.Generate.Receipt
	}
	cfg := generateReceiptConfig{
		Request: proof.ReceiptProofRequest{
			RPCURLs:       mergeStringSlice(seen, "rpc", rpcURLs, nil),
			MinRPCSources: mergeInt(seen, "min-rpcs", *minRPCs, nil, proofMinRPCsDefault()),
			LogIndex:      mergeUint(seen, "log-index", *logIndex, nil, 0),
		},
		Out: mergeString(seen, "out", *out, "", "receipt.json"),
	}
	rawTxHash := mergeString(seen, "tx", *txHashHex, "", "")
	if section != nil {
		cfg.Request.RPCURLs = mergeStringSlice(seen, "rpc", rpcURLs, section.RPCs)
		cfg.Request.MinRPCSources = mergeInt(seen, "min-rpcs", *minRPCs, section.MinRPCs, proofMinRPCsDefault())
		cfg.Request.LogIndex = mergeUint(seen, "log-index", *logIndex, section.LogIndex, 0)
		rawTxHash = mergeString(seen, "tx", *txHashHex, section.Tx, "")
		cfg.Out = mergeString(seen, "out", *out, section.Out, "receipt.json")
	}
	if err := validateRPCInputs(cfg.Request.RPCURLs, cfg.Request.MinRPCSources, "generate receipt requires at least one RPC via --rpc or generate.receipt.rpcs in --config"); err != nil {
		return generateReceiptConfig{}, err
	}
	if rawTxHash == "" {
		return generateReceiptConfig{}, newUsageError("generate receipt requires --tx or generate.receipt.tx in --config")
	}
	cfg.Request.TxHash = common.HexToHash(rawTxHash)
	return cfg, nil
}

func parseGenerateTransactionArgs(args []string) (generateTransactionConfig, error) {
	fs := newFlagSet("generate tx")
	configPath := fs.String("config", "", "config json file")
	var rpcURLs multiStringFlag
	fs.Var(&rpcURLs, "rpc", "Ethereum RPC URL")
	minRPCs := fs.Int("min-rpcs", proofMinRPCsDefault(), "minimum distinct RPC sources required")
	txHashHex := fs.String("tx", "", "transaction hash")
	out := fs.String("out", "tx.json", "output proof json")
	if err := parseFlagSet(fs, args); err != nil {
		if _, ok := asUsageError(err); ok {
			return generateTransactionConfig{}, err
		}
		return generateTransactionConfig{}, newUsageError("parse generate tx args: %v", err)
	}
	if err := ensureNoPositionalArgs(fs); err != nil {
		return generateTransactionConfig{}, err
	}

	seen := visitedFlags(fs)
	fileCfg, err := loadCLIConfig(*configPath)
	if err != nil {
		return generateTransactionConfig{}, newUsageError("%v", err)
	}

	var section *generateTransactionConfigFile
	if fileCfg != nil {
		section = fileCfg.Generate.Tx
	}
	cfg := generateTransactionConfig{
		Request: proof.TransactionProofRequest{
			RPCURLs:       mergeStringSlice(seen, "rpc", rpcURLs, nil),
			MinRPCSources: mergeInt(seen, "min-rpcs", *minRPCs, nil, proofMinRPCsDefault()),
		},
		Out: mergeString(seen, "out", *out, "", "tx.json"),
	}
	rawTxHash := mergeString(seen, "tx", *txHashHex, "", "")
	if section != nil {
		cfg.Request.RPCURLs = mergeStringSlice(seen, "rpc", rpcURLs, section.RPCs)
		cfg.Request.MinRPCSources = mergeInt(seen, "min-rpcs", *minRPCs, section.MinRPCs, proofMinRPCsDefault())
		rawTxHash = mergeString(seen, "tx", *txHashHex, section.Tx, "")
		cfg.Out = mergeString(seen, "out", *out, section.Out, "tx.json")
	}
	if err := validateRPCInputs(cfg.Request.RPCURLs, cfg.Request.MinRPCSources, "generate tx requires at least one RPC via --rpc or generate.tx.rpcs in --config"); err != nil {
		return generateTransactionConfig{}, err
	}
	if rawTxHash == "" {
		return generateTransactionConfig{}, newUsageError("generate tx requires --tx or generate.tx.tx in --config")
	}
	cfg.Request.TxHash = common.HexToHash(rawTxHash)
	return cfg, nil
}

func parseVerifyStateArgs(args []string) (verifyStateConfig, error) {
	fs := newFlagSet("verify state")
	configPath := fs.String("config", "", "config json file")
	var rpcURLs multiStringFlag
	fs.Var(&rpcURLs, "rpc", "Ethereum RPC URL")
	minRPCs := fs.Int("min-rpcs", proofMinRPCsDefault(), "minimum distinct RPC sources required")
	proofPath := fs.String("proof", "state.json", "proof json file")
	if err := parseFlagSet(fs, args); err != nil {
		if _, ok := asUsageError(err); ok {
			return verifyStateConfig{}, err
		}
		return verifyStateConfig{}, newUsageError("parse verify state args: %v", err)
	}
	if err := ensureNoPositionalArgs(fs); err != nil {
		return verifyStateConfig{}, err
	}

	seen := visitedFlags(fs)
	fileCfg, err := loadCLIConfig(*configPath)
	if err != nil {
		return verifyStateConfig{}, newUsageError("%v", err)
	}

	var section *verifyStateConfigFile
	if fileCfg != nil {
		section = fileCfg.Verify.State
	}
	cfg := verifyStateConfig{
		ProofPath: mergeString(seen, "proof", *proofPath, "", "state.json"),
		VerifyRequest: proof.VerifyRPCRequest{
			RPCURLs:       mergeStringSlice(seen, "rpc", rpcURLs, nil),
			MinRPCSources: mergeInt(seen, "min-rpcs", *minRPCs, nil, proofMinRPCsDefault()),
		},
	}
	if section != nil {
		cfg.ProofPath = mergeString(seen, "proof", *proofPath, section.Proof, "state.json")
		cfg.VerifyRequest.RPCURLs = mergeStringSlice(seen, "rpc", rpcURLs, section.RPCs)
		cfg.VerifyRequest.MinRPCSources = mergeInt(seen, "min-rpcs", *minRPCs, section.MinRPCs, proofMinRPCsDefault())
	}
	if err := validateRPCInputs(cfg.VerifyRequest.RPCURLs, cfg.VerifyRequest.MinRPCSources, "verify state requires independent RPCs via --rpc or verify.state.rpcs in --config"); err != nil {
		return verifyStateConfig{}, err
	}
	return cfg, nil
}

func parseVerifyReceiptArgs(args []string) (verifyReceiptConfig, error) {
	fs := newFlagSet("verify receipt")
	configPath := fs.String("config", "", "config json file")
	var rpcURLs multiStringFlag
	fs.Var(&rpcURLs, "rpc", "Ethereum RPC URL")
	minRPCs := fs.Int("min-rpcs", proofMinRPCsDefault(), "minimum distinct RPC sources required")
	proofPath := fs.String("proof", "receipt.json", "proof json file")
	expectEmitterHex := fs.String("expect-emitter", "", "optional expected emitter address")
	expectDataHex := fs.String("expect-data", "", "optional expected event data hex")
	var topics multiStringFlag
	fs.Var(&topics, "expect-topic", "optional expected topic (repeatable)")
	if err := parseFlagSet(fs, args); err != nil {
		if _, ok := asUsageError(err); ok {
			return verifyReceiptConfig{}, err
		}
		return verifyReceiptConfig{}, newUsageError("parse verify receipt args: %v", err)
	}
	if err := ensureNoPositionalArgs(fs); err != nil {
		return verifyReceiptConfig{}, err
	}

	seen := visitedFlags(fs)
	fileCfg, err := loadCLIConfig(*configPath)
	if err != nil {
		return verifyReceiptConfig{}, newUsageError("%v", err)
	}

	var section *verifyReceiptConfigFile
	if fileCfg != nil {
		section = fileCfg.Verify.Receipt
	}
	cfg := verifyReceiptConfig{
		ProofPath: mergeString(seen, "proof", *proofPath, "", "receipt.json"),
		VerifyRequest: proof.VerifyRPCRequest{
			RPCURLs:       mergeStringSlice(seen, "rpc", rpcURLs, nil),
			MinRPCSources: mergeInt(seen, "min-rpcs", *minRPCs, nil, proofMinRPCsDefault()),
		},
	}
	rawEmitter := mergeString(seen, "expect-emitter", *expectEmitterHex, "", "")
	rawData := mergeString(seen, "expect-data", *expectDataHex, "", "")
	rawTopics := mergeStringSlice(seen, "expect-topic", topics, nil)
	if section != nil {
		cfg.ProofPath = mergeString(seen, "proof", *proofPath, section.Proof, "receipt.json")
		cfg.VerifyRequest.RPCURLs = mergeStringSlice(seen, "rpc", rpcURLs, section.RPCs)
		cfg.VerifyRequest.MinRPCSources = mergeInt(seen, "min-rpcs", *minRPCs, section.MinRPCs, proofMinRPCsDefault())
		rawEmitter = mergeString(seen, "expect-emitter", *expectEmitterHex, section.ExpectEmitter, "")
		rawData = mergeString(seen, "expect-data", *expectDataHex, section.ExpectData, "")
		rawTopics = mergeStringSlice(seen, "expect-topic", topics, section.ExpectTopics)
	}
	if err := validateRPCInputs(cfg.VerifyRequest.RPCURLs, cfg.VerifyRequest.MinRPCSources, "verify receipt requires independent RPCs via --rpc or verify.receipt.rpcs in --config"); err != nil {
		return verifyReceiptConfig{}, err
	}
	expect, err := buildReceiptExpectations(rawEmitter, rawData, rawTopics)
	if err != nil {
		return verifyReceiptConfig{}, err
	}
	cfg.Expectations = expect
	return cfg, nil
}

func parseVerifyTransactionArgs(args []string) (verifyTransactionConfig, error) {
	fs := newFlagSet("verify tx")
	configPath := fs.String("config", "", "config json file")
	var rpcURLs multiStringFlag
	fs.Var(&rpcURLs, "rpc", "Ethereum RPC URL")
	minRPCs := fs.Int("min-rpcs", proofMinRPCsDefault(), "minimum distinct RPC sources required")
	proofPath := fs.String("proof", "tx.json", "proof json file")
	if err := parseFlagSet(fs, args); err != nil {
		if _, ok := asUsageError(err); ok {
			return verifyTransactionConfig{}, err
		}
		return verifyTransactionConfig{}, newUsageError("parse verify tx args: %v", err)
	}
	if err := ensureNoPositionalArgs(fs); err != nil {
		return verifyTransactionConfig{}, err
	}

	seen := visitedFlags(fs)
	fileCfg, err := loadCLIConfig(*configPath)
	if err != nil {
		return verifyTransactionConfig{}, newUsageError("%v", err)
	}

	var section *verifyTransactionConfigFile
	if fileCfg != nil {
		section = fileCfg.Verify.Tx
	}
	cfg := verifyTransactionConfig{
		ProofPath: mergeString(seen, "proof", *proofPath, "", "tx.json"),
		VerifyRequest: proof.VerifyRPCRequest{
			RPCURLs:       mergeStringSlice(seen, "rpc", rpcURLs, nil),
			MinRPCSources: mergeInt(seen, "min-rpcs", *minRPCs, nil, proofMinRPCsDefault()),
		},
	}
	if section != nil {
		cfg.ProofPath = mergeString(seen, "proof", *proofPath, section.Proof, "tx.json")
		cfg.VerifyRequest.RPCURLs = mergeStringSlice(seen, "rpc", rpcURLs, section.RPCs)
		cfg.VerifyRequest.MinRPCSources = mergeInt(seen, "min-rpcs", *minRPCs, section.MinRPCs, proofMinRPCsDefault())
	}
	if err := validateRPCInputs(cfg.VerifyRequest.RPCURLs, cfg.VerifyRequest.MinRPCSources, "verify tx requires independent RPCs via --rpc or verify.tx.rpcs in --config"); err != nil {
		return verifyTransactionConfig{}, err
	}
	return cfg, nil
}

func buildReceiptExpectations(expectEmitterHex string, expectDataHex string, topics []string) (*proof.ReceiptExpectations, error) {
	var expect proof.ReceiptExpectations
	if expectEmitterHex != "" {
		addr := common.HexToAddress(expectEmitterHex)
		expect.Emitter = &addr
	}
	if expectDataHex != "" {
		expect.Data = common.FromHex(expectDataHex)
	}
	for _, topic := range topics {
		expect.Topics = append(expect.Topics, common.HexToHash(topic))
	}
	if expect.Emitter == nil && expect.Data == nil && len(expect.Topics) == 0 {
		return nil, nil
	}
	return &expect, nil
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func parseFlagSet(fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return newHelpError()
		}
		return err
	}
	return nil
}

func ensureNoPositionalArgs(fs *flag.FlagSet) error {
	if fs.NArg() == 0 {
		return nil
	}
	return newUsageError("%s does not accept positional arguments: %s", fs.Name(), strings.Join(fs.Args(), " "))
}

func visitedFlags(fs *flag.FlagSet) map[string]bool {
	out := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		out[f.Name] = true
	})
	return out
}

func mergeString(seen map[string]bool, flagName string, flagValue string, configValue string, defaultValue string) string {
	if seen[flagName] {
		return flagValue
	}
	if configValue != "" {
		return configValue
	}
	return defaultValue
}

func mergeStringSlice(seen map[string]bool, flagName string, flagValue []string, configValue []string) []string {
	if seen[flagName] {
		return append([]string(nil), flagValue...)
	}
	return append([]string(nil), configValue...)
}

func mergeInt(seen map[string]bool, flagName string, flagValue int, configValue *int, defaultValue int) int {
	if seen[flagName] {
		return flagValue
	}
	if configValue != nil {
		return *configValue
	}
	return defaultValue
}

func mergeUint(seen map[string]bool, flagName string, flagValue uint, configValue *uint, defaultValue uint) uint {
	if seen[flagName] {
		return flagValue
	}
	if configValue != nil {
		return *configValue
	}
	return defaultValue
}

func mergeUint64(seen map[string]bool, flagName string, flagValue uint64, configValue *uint64, defaultValue uint64) uint64 {
	if seen[flagName] {
		return flagValue
	}
	if configValue != nil {
		return *configValue
	}
	return defaultValue
}

func validateRPCInputs(rpcURLs []string, minRPCs int, missingMessage string) error {
	if len(rpcURLs) == 0 {
		return newUsageError("%s", missingMessage)
	}
	if minRPCs < 1 {
		return newUsageError("--min-rpcs must be at least 1")
	}
	if len(rpcURLs) < minRPCs {
		return newUsageError("--min-rpcs=%d requires at least %d rpc values, got %d", minRPCs, minRPCs, len(rpcURLs))
	}
	return nil
}

func proofMinRPCsDefault() int {
	return 3
}
