package main

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/islishude/ethproof/proof"
)

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

func parseGenerateStateArgs(args []string) (generateStateConfig, error) {
	fs := newFlagSet("generate state")
	configPath := fs.String("config", "", "config json file")
	var rpcURLs multiStringFlag
	fs.Var(&rpcURLs, "rpc", "Ethereum RPC URL")
	var slotHexes multiStringFlag
	minRPCs := fs.Int("min-rpcs", proof.DefaultMinRPCSources, "minimum distinct RPC sources required")
	blockNumber := fs.Uint64("block", 0, "block number")
	accountHex := fs.String("account", "", "account address")
	fs.Var(&slotHexes, "slot", "32-byte storage slot key (repeatable)")
	out := fs.String("out", "state.json", "output proof json")

	parseCtx, err := prepareParse(fs, args, configPath, "parse generate state args")
	if err != nil {
		return generateStateConfig{}, err
	}

	var section *generateStateConfigFile
	if parseCtx.fileCfg != nil {
		section = parseCtx.fileCfg.Generate.State
	}

	cfg := generateStateConfig{
		Request: proof.StateProofRequest{
			BlockNumber: mergeUint64(parseCtx.seen, "block", *blockNumber, nil, 0),
		},
		Out: mergeString(parseCtx.seen, "out", *out, "", "state.json"),
	}
	cfg.Request.RPCURLs, cfg.Request.MinRPCSources = mergeRPCInputs(parseCtx.seen, rpcURLs, *minRPCs, nil, nil)
	rawAccount := mergeString(parseCtx.seen, "account", *accountHex, "", "")
	rawSlots := mergeStringSlice(parseCtx.seen, "slot", slotHexes, nil)
	if section != nil {
		cfg.Request.RPCURLs, cfg.Request.MinRPCSources = mergeRPCInputs(parseCtx.seen, rpcURLs, *minRPCs, section.RPCs, section.MinRPCs)
		cfg.Request.BlockNumber = mergeUint64(parseCtx.seen, "block", *blockNumber, section.Block, 0)
		rawAccount = mergeString(parseCtx.seen, "account", *accountHex, section.Account, "")
		rawSlots = mergeStringSlice(parseCtx.seen, "slot", slotHexes, section.Slots)
		cfg.Out = mergeString(parseCtx.seen, "out", *out, section.Out, "state.json")
	}
	if err := validateRPCInputs(cfg.Request.RPCURLs, cfg.Request.MinRPCSources, "generate state requires at least one RPC via --rpc or generate.state.rpcs in --config"); err != nil {
		return generateStateConfig{}, err
	}
	if rawAccount == "" {
		return generateStateConfig{}, newUsageError("generate state requires --account or generate.state.account in --config")
	}
	if len(rawSlots) == 0 {
		return generateStateConfig{}, newUsageError("generate state requires at least one --slot or generate.state.slots in --config")
	}
	cfg.Request.Account = common.HexToAddress(rawAccount)
	cfg.Request.Slots = make([]common.Hash, len(rawSlots))
	for i, slot := range rawSlots {
		cfg.Request.Slots[i] = common.HexToHash(slot)
	}
	return cfg, nil
}

func parseGenerateReceiptArgs(args []string) (generateReceiptConfig, error) {
	fs := newFlagSet("generate receipt")
	configPath := fs.String("config", "", "config json file")
	var rpcURLs multiStringFlag
	fs.Var(&rpcURLs, "rpc", "Ethereum RPC URL")
	minRPCs := fs.Int("min-rpcs", proof.DefaultMinRPCSources, "minimum distinct RPC sources required")
	txHashHex := fs.String("tx", "", "transaction hash")
	logIndex := fs.Uint("log-index", 0, "log index within receipt")
	out := fs.String("out", "receipt.json", "output proof json")

	parseCtx, err := prepareParse(fs, args, configPath, "parse generate receipt args")
	if err != nil {
		return generateReceiptConfig{}, err
	}

	var section *generateReceiptConfigFile
	if parseCtx.fileCfg != nil {
		section = parseCtx.fileCfg.Generate.Receipt
	}

	cfg := generateReceiptConfig{
		Request: proof.ReceiptProofRequest{
			LogIndex: mergeUint(parseCtx.seen, "log-index", *logIndex, nil, 0),
		},
		Out: mergeString(parseCtx.seen, "out", *out, "", "receipt.json"),
	}
	cfg.Request.RPCURLs, cfg.Request.MinRPCSources = mergeRPCInputs(parseCtx.seen, rpcURLs, *minRPCs, nil, nil)
	rawTxHash := mergeString(parseCtx.seen, "tx", *txHashHex, "", "")
	if section != nil {
		cfg.Request.RPCURLs, cfg.Request.MinRPCSources = mergeRPCInputs(parseCtx.seen, rpcURLs, *minRPCs, section.RPCs, section.MinRPCs)
		cfg.Request.LogIndex = mergeUint(parseCtx.seen, "log-index", *logIndex, section.LogIndex, 0)
		rawTxHash = mergeString(parseCtx.seen, "tx", *txHashHex, section.Tx, "")
		cfg.Out = mergeString(parseCtx.seen, "out", *out, section.Out, "receipt.json")
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
	minRPCs := fs.Int("min-rpcs", proof.DefaultMinRPCSources, "minimum distinct RPC sources required")
	txHashHex := fs.String("tx", "", "transaction hash")
	out := fs.String("out", "tx.json", "output proof json")

	parseCtx, err := prepareParse(fs, args, configPath, "parse generate tx args")
	if err != nil {
		return generateTransactionConfig{}, err
	}

	var section *generateTransactionConfigFile
	if parseCtx.fileCfg != nil {
		section = parseCtx.fileCfg.Generate.Tx
	}

	cfg := generateTransactionConfig{
		Out: mergeString(parseCtx.seen, "out", *out, "", "tx.json"),
	}
	cfg.Request.RPCURLs, cfg.Request.MinRPCSources = mergeRPCInputs(parseCtx.seen, rpcURLs, *minRPCs, nil, nil)
	rawTxHash := mergeString(parseCtx.seen, "tx", *txHashHex, "", "")
	if section != nil {
		cfg.Request.RPCURLs, cfg.Request.MinRPCSources = mergeRPCInputs(parseCtx.seen, rpcURLs, *minRPCs, section.RPCs, section.MinRPCs)
		rawTxHash = mergeString(parseCtx.seen, "tx", *txHashHex, section.Tx, "")
		cfg.Out = mergeString(parseCtx.seen, "out", *out, section.Out, "tx.json")
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
