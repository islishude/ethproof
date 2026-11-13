package main

import (
	"github.com/islishude/ethproof/proof"
)

type verifyStateConfig struct {
	Finality  *proof.FinalityConfig
	ProofPath string
	RPCURL    string
}

type verifyReceiptConfig struct {
	Finality     *proof.FinalityConfig
	ProofPath    string
	Expectations *proof.ReceiptExpectations
	RPCURL       string
}

type verifyTransactionConfig struct {
	Finality  *proof.FinalityConfig
	ProofPath string
	RPCURL    string
}

func parseVerifyStateArgs(args []string) (verifyStateConfig, error) {
	cfg, err := parseVerifyProofArgs(args, "state", func(cfg *cliConfig) *verifyProofConfigFile {
		return cfg.Verify.State
	})
	return verifyStateConfig(cfg), err
}

type verifyProofConfig struct {
	Finality  *proof.FinalityConfig
	ProofPath string
	RPCURL    string
}

func parseVerifyProofArgs(args []string, kind string, selectSection func(*cliConfig) *verifyProofConfigFile) (verifyProofConfig, error) {
	fs := newFlagSet("verify " + kind)
	configPath := fs.String("config", "", "config json file")
	finality := fs.Bool("finality", false, "require consensus finality using finality configuration")
	var rpcURL singleRPCFlag
	fs.Var(&rpcURL, "rpc", "Ethereum RPC URL")
	proofPath := fs.String("proof", kind+".json", "proof json file")

	parseCtx, err := prepareParse(fs, args, configPath, "parse verify "+kind+" args")
	if err != nil {
		return verifyProofConfig{}, err
	}

	var section *verifyProofConfigFile
	if parseCtx.fileCfg != nil {
		section = selectSection(parseCtx.fileCfg)
	}

	cfg := verifyProofConfig{
		ProofPath: mergeString(parseCtx.seen, "proof", *proofPath, "", kind+".json"),
	}
	cfg.RPCURL = mergeString(parseCtx.seen, "rpc", rpcURL.value, "", "")
	if section != nil {
		cfg.ProofPath = mergeString(parseCtx.seen, "proof", *proofPath, section.Proof, kind+".json")
		cfg.RPCURL = mergeString(parseCtx.seen, "rpc", rpcURL.value, section.RPC, "")
	}
	enabled := *finality
	if !parseCtx.seen["finality"] && section != nil {
		enabled = section.Finality
	}
	cfg.Finality, err = parseFinalityConfig(enabled, parseCtx.fileCfg)
	if err != nil {
		return verifyProofConfig{}, err
	}
	if err := validateRPCInput(cfg.RPCURL, "verify "+kind+" requires a separately configured RPC via --rpc or verify."+kind+".rpc in --config"); err != nil {
		return verifyProofConfig{}, err
	}
	return cfg, nil
}

func parseVerifyReceiptArgs(args []string) (verifyReceiptConfig, error) {
	fs := newFlagSet("verify receipt")
	configPath := fs.String("config", "", "config json file")
	finality := fs.Bool("finality", false, "require consensus finality using finality configuration")
	var rpcURL singleRPCFlag
	fs.Var(&rpcURL, "rpc", "Ethereum RPC URL")
	proofPath := fs.String("proof", "receipt.json", "proof json file")
	expectEmitterHex := fs.String("expect-emitter", "", "optional expected 20-byte 0x-prefixed emitter address")
	expectDataHex := fs.String("expect-data", "", "optional expected even-length 0x-prefixed event data")
	var topics multiStringFlag
	fs.Var(&topics, "expect-topic", "optional expected 32-byte 0x-prefixed topic (repeatable)")

	parseCtx, err := prepareParse(fs, args, configPath, "parse verify receipt args")
	if err != nil {
		return verifyReceiptConfig{}, err
	}

	var section *verifyReceiptConfigFile
	if parseCtx.fileCfg != nil {
		section = parseCtx.fileCfg.Verify.Receipt
	}

	cfg := verifyReceiptConfig{
		ProofPath: mergeString(parseCtx.seen, "proof", *proofPath, "", "receipt.json"),
	}
	cfg.RPCURL = mergeString(parseCtx.seen, "rpc", rpcURL.value, "", "")
	rawEmitter := mergeString(parseCtx.seen, "expect-emitter", *expectEmitterHex, "", "")
	rawData := mergeString(parseCtx.seen, "expect-data", *expectDataHex, "", "")
	rawTopics := mergeStringSlice(parseCtx.seen, "expect-topic", topics, nil)
	if section != nil {
		cfg.ProofPath = mergeString(parseCtx.seen, "proof", *proofPath, section.Proof, "receipt.json")
		cfg.RPCURL = mergeString(parseCtx.seen, "rpc", rpcURL.value, section.RPC, "")
		rawEmitter = mergeString(parseCtx.seen, "expect-emitter", *expectEmitterHex, section.ExpectEmitter, "")
		rawData = mergeString(parseCtx.seen, "expect-data", *expectDataHex, section.ExpectData, "")
		rawTopics = mergeStringSlice(parseCtx.seen, "expect-topic", topics, section.ExpectTopics)
	}
	enabled := *finality
	if !parseCtx.seen["finality"] && section != nil {
		enabled = section.Finality
	}
	cfg.Finality, err = parseFinalityConfig(enabled, parseCtx.fileCfg)
	if err != nil {
		return verifyReceiptConfig{}, err
	}
	if err := validateRPCInput(cfg.RPCURL, "verify receipt requires a separately configured RPC via --rpc or verify.receipt.rpc in --config"); err != nil {
		return verifyReceiptConfig{}, err
	}
	expect, err := buildReceiptExpectations(rawEmitter, rawData, rawTopics)
	if err != nil {
		return verifyReceiptConfig{}, newUsageError("%v", err)
	}
	cfg.Expectations = expect
	return cfg, nil
}

func parseVerifyTransactionArgs(args []string) (verifyTransactionConfig, error) {
	cfg, err := parseVerifyProofArgs(args, "tx", func(cfg *cliConfig) *verifyProofConfigFile {
		return cfg.Verify.Tx
	})
	return verifyTransactionConfig(cfg), err
}

func parseFinalityConfig(enabled bool, file *cliConfig) (*proof.FinalityConfig, error) {
	if !enabled {
		return nil, nil
	}
	if file == nil || file.Finality == nil {
		return nil, newUsageError("--finality requires a top-level finality configuration")
	}
	cfg := *file.Finality
	if err := proof.ValidateFinalityConfig(cfg); err != nil {
		return nil, newUsageError("finality configuration: %v", err)
	}
	return &cfg, nil
}
