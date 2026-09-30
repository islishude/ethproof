package main

import (
	"github.com/islishude/ethproof/proof"
)

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

func parseVerifyStateArgs(args []string) (verifyStateConfig, error) {
	cfg, err := parseVerifyProofArgs(args, "state", func(cfg *cliConfig) *verifyProofConfigFile {
		return cfg.Verify.State
	})
	return verifyStateConfig(cfg), err
}

type verifyProofConfig struct {
	ProofPath     string
	VerifyRequest proof.VerifyRPCRequest
}

func parseVerifyProofArgs(args []string, kind string, selectSection func(*cliConfig) *verifyProofConfigFile) (verifyProofConfig, error) {
	fs := newFlagSet("verify " + kind)
	configPath := fs.String("config", "", "config json file")
	var rpcURLs multiStringFlag
	fs.Var(&rpcURLs, "rpc", "Ethereum RPC URL")
	minRPCs := fs.Int("min-rpcs", proof.DefaultMinRPCSources, "minimum distinct RPC sources required")
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
	cfg.VerifyRequest.RPCURLs, cfg.VerifyRequest.MinRPCSources = mergeRPCInputs(parseCtx.seen, rpcURLs, *minRPCs, nil, nil)
	if section != nil {
		cfg.ProofPath = mergeString(parseCtx.seen, "proof", *proofPath, section.Proof, kind+".json")
		cfg.VerifyRequest.RPCURLs, cfg.VerifyRequest.MinRPCSources = mergeRPCInputs(parseCtx.seen, rpcURLs, *minRPCs, section.RPCs, section.MinRPCs)
	}
	if err := validateRPCInputs(cfg.VerifyRequest.RPCURLs, cfg.VerifyRequest.MinRPCSources, "verify "+kind+" requires independent RPCs via --rpc or verify."+kind+".rpcs in --config"); err != nil {
		return verifyProofConfig{}, err
	}
	return cfg, nil
}

func parseVerifyReceiptArgs(args []string) (verifyReceiptConfig, error) {
	fs := newFlagSet("verify receipt")
	configPath := fs.String("config", "", "config json file")
	var rpcURLs multiStringFlag
	fs.Var(&rpcURLs, "rpc", "Ethereum RPC URL")
	minRPCs := fs.Int("min-rpcs", proof.DefaultMinRPCSources, "minimum distinct RPC sources required")
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
	cfg.VerifyRequest.RPCURLs, cfg.VerifyRequest.MinRPCSources = mergeRPCInputs(parseCtx.seen, rpcURLs, *minRPCs, nil, nil)
	rawEmitter := mergeString(parseCtx.seen, "expect-emitter", *expectEmitterHex, "", "")
	rawData := mergeString(parseCtx.seen, "expect-data", *expectDataHex, "", "")
	rawTopics := mergeStringSlice(parseCtx.seen, "expect-topic", topics, nil)
	if section != nil {
		cfg.ProofPath = mergeString(parseCtx.seen, "proof", *proofPath, section.Proof, "receipt.json")
		cfg.VerifyRequest.RPCURLs, cfg.VerifyRequest.MinRPCSources = mergeRPCInputs(parseCtx.seen, rpcURLs, *minRPCs, section.RPCs, section.MinRPCs)
		rawEmitter = mergeString(parseCtx.seen, "expect-emitter", *expectEmitterHex, section.ExpectEmitter, "")
		rawData = mergeString(parseCtx.seen, "expect-data", *expectDataHex, section.ExpectData, "")
		rawTopics = mergeStringSlice(parseCtx.seen, "expect-topic", topics, section.ExpectTopics)
	}
	if err := validateRPCInputs(cfg.VerifyRequest.RPCURLs, cfg.VerifyRequest.MinRPCSources, "verify receipt requires independent RPCs via --rpc or verify.receipt.rpcs in --config"); err != nil {
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
