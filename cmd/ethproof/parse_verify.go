package main

import "github.com/islishude/ethproof/proof"

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
	fs := newFlagSet("verify state")
	configPath := fs.String("config", "", "config json file")
	var rpcURLs multiStringFlag
	fs.Var(&rpcURLs, "rpc", "Ethereum RPC URL")
	minRPCs := fs.Int("min-rpcs", proofMinRPCsDefault(), "minimum distinct RPC sources required")
	proofPath := fs.String("proof", "state.json", "proof json file")

	parseCtx, err := prepareParse(fs, args, configPath, "parse verify state args")
	if err != nil {
		return verifyStateConfig{}, err
	}

	var section *verifyStateConfigFile
	if parseCtx.fileCfg != nil {
		section = parseCtx.fileCfg.Verify.State
	}

	cfg := verifyStateConfig{
		ProofPath: mergeString(parseCtx.seen, "proof", *proofPath, "", "state.json"),
	}
	cfg.VerifyRequest.RPCURLs, cfg.VerifyRequest.MinRPCSources = mergeRPCInputs(parseCtx.seen, rpcURLs, *minRPCs, nil, nil)
	if section != nil {
		cfg.ProofPath = mergeString(parseCtx.seen, "proof", *proofPath, section.Proof, "state.json")
		cfg.VerifyRequest.RPCURLs, cfg.VerifyRequest.MinRPCSources = mergeRPCInputs(parseCtx.seen, rpcURLs, *minRPCs, section.RPCs, section.MinRPCs)
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

	parseCtx, err := prepareParse(fs, args, configPath, "parse verify tx args")
	if err != nil {
		return verifyTransactionConfig{}, err
	}

	var section *verifyTransactionConfigFile
	if parseCtx.fileCfg != nil {
		section = parseCtx.fileCfg.Verify.Tx
	}

	cfg := verifyTransactionConfig{
		ProofPath: mergeString(parseCtx.seen, "proof", *proofPath, "", "tx.json"),
	}
	cfg.VerifyRequest.RPCURLs, cfg.VerifyRequest.MinRPCSources = mergeRPCInputs(parseCtx.seen, rpcURLs, *minRPCs, nil, nil)
	if section != nil {
		cfg.ProofPath = mergeString(parseCtx.seen, "proof", *proofPath, section.Proof, "tx.json")
		cfg.VerifyRequest.RPCURLs, cfg.VerifyRequest.MinRPCSources = mergeRPCInputs(parseCtx.seen, rpcURLs, *minRPCs, section.RPCs, section.MinRPCs)
	}
	if err := validateRPCInputs(cfg.VerifyRequest.RPCURLs, cfg.VerifyRequest.MinRPCSources, "verify tx requires independent RPCs via --rpc or verify.tx.rpcs in --config"); err != nil {
		return verifyTransactionConfig{}, err
	}
	return cfg, nil
}
