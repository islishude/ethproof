package main

import (
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
	ProofPath string
}

type verifyReceiptConfig struct {
	ProofPath    string
	Expectations *proof.ReceiptExpectations
}

type verifyTransactionConfig struct {
	ProofPath string
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
	var rpcURLs multiStringFlag
	fs.Var(&rpcURLs, "rpc", "Ethereum RPC URL")
	minRPCs := fs.Int("min-rpcs", proofMinRPCsDefault(), "minimum distinct RPC sources required")
	blockNumber := fs.Uint64("block", 0, "block number")
	accountHex := fs.String("account", "", "account address")
	slotHex := fs.String("slot", "", "32-byte storage slot key")
	out := fs.String("out", "state.json", "output proof json")
	if err := fs.Parse(args); err != nil {
		return generateStateConfig{}, newUsageError("parse generate state args: %v", err)
	}
	if err := ensureNoPositionalArgs(fs); err != nil {
		return generateStateConfig{}, err
	}
	if err := validateRPCInputs(rpcURLs, *minRPCs); err != nil {
		return generateStateConfig{}, err
	}
	if *accountHex == "" {
		return generateStateConfig{}, newUsageError("generate state requires --account")
	}
	if *slotHex == "" {
		return generateStateConfig{}, newUsageError("generate state requires --slot")
	}

	return generateStateConfig{
		Request: proof.StateProofRequest{
			RPCURLs:       rpcURLs,
			MinRPCSources: *minRPCs,
			BlockNumber:   *blockNumber,
			Account:       common.HexToAddress(*accountHex),
			Slot:          common.HexToHash(*slotHex),
		},
		Out: *out,
	}, nil
}

func parseGenerateReceiptArgs(args []string) (generateReceiptConfig, error) {
	fs := newFlagSet("generate receipt")
	var rpcURLs multiStringFlag
	fs.Var(&rpcURLs, "rpc", "Ethereum RPC URL")
	minRPCs := fs.Int("min-rpcs", proofMinRPCsDefault(), "minimum distinct RPC sources required")
	txHashHex := fs.String("tx", "", "transaction hash")
	logIndex := fs.Uint("log-index", 0, "log index within receipt")
	out := fs.String("out", "receipt.json", "output proof json")
	if err := fs.Parse(args); err != nil {
		return generateReceiptConfig{}, newUsageError("parse generate receipt args: %v", err)
	}
	if err := ensureNoPositionalArgs(fs); err != nil {
		return generateReceiptConfig{}, err
	}
	if err := validateRPCInputs(rpcURLs, *minRPCs); err != nil {
		return generateReceiptConfig{}, err
	}
	if *txHashHex == "" {
		return generateReceiptConfig{}, newUsageError("generate receipt requires --tx")
	}

	return generateReceiptConfig{
		Request: proof.ReceiptProofRequest{
			RPCURLs:       rpcURLs,
			MinRPCSources: *minRPCs,
			TxHash:        common.HexToHash(*txHashHex),
			LogIndex:      *logIndex,
		},
		Out: *out,
	}, nil
}

func parseGenerateTransactionArgs(args []string) (generateTransactionConfig, error) {
	fs := newFlagSet("generate tx")
	var rpcURLs multiStringFlag
	fs.Var(&rpcURLs, "rpc", "Ethereum RPC URL")
	minRPCs := fs.Int("min-rpcs", proofMinRPCsDefault(), "minimum distinct RPC sources required")
	txHashHex := fs.String("tx", "", "transaction hash")
	out := fs.String("out", "tx.json", "output proof json")
	if err := fs.Parse(args); err != nil {
		return generateTransactionConfig{}, newUsageError("parse generate tx args: %v", err)
	}
	if err := ensureNoPositionalArgs(fs); err != nil {
		return generateTransactionConfig{}, err
	}
	if err := validateRPCInputs(rpcURLs, *minRPCs); err != nil {
		return generateTransactionConfig{}, err
	}
	if *txHashHex == "" {
		return generateTransactionConfig{}, newUsageError("generate tx requires --tx")
	}

	return generateTransactionConfig{
		Request: proof.TransactionProofRequest{
			RPCURLs:       rpcURLs,
			MinRPCSources: *minRPCs,
			TxHash:        common.HexToHash(*txHashHex),
		},
		Out: *out,
	}, nil
}

func parseVerifyStateArgs(args []string) (verifyStateConfig, error) {
	fs := newFlagSet("verify state")
	proofPath := fs.String("proof", "state.json", "proof json file")
	if err := fs.Parse(args); err != nil {
		return verifyStateConfig{}, newUsageError("parse verify state args: %v", err)
	}
	if err := ensureNoPositionalArgs(fs); err != nil {
		return verifyStateConfig{}, err
	}

	return verifyStateConfig{ProofPath: *proofPath}, nil
}

func parseVerifyReceiptArgs(args []string) (verifyReceiptConfig, error) {
	fs := newFlagSet("verify receipt")
	proofPath := fs.String("proof", "receipt.json", "proof json file")
	expectEmitterHex := fs.String("expect-emitter", "", "optional expected emitter address")
	expectDataHex := fs.String("expect-data", "", "optional expected event data hex")
	var topics multiStringFlag
	fs.Var(&topics, "expect-topic", "optional expected topic (repeatable)")
	if err := fs.Parse(args); err != nil {
		return verifyReceiptConfig{}, newUsageError("parse verify receipt args: %v", err)
	}
	if err := ensureNoPositionalArgs(fs); err != nil {
		return verifyReceiptConfig{}, err
	}

	expect, err := buildReceiptExpectations(*expectEmitterHex, *expectDataHex, topics)
	if err != nil {
		return verifyReceiptConfig{}, err
	}

	return verifyReceiptConfig{
		ProofPath:    *proofPath,
		Expectations: expect,
	}, nil
}

func parseVerifyTransactionArgs(args []string) (verifyTransactionConfig, error) {
	fs := newFlagSet("verify tx")
	proofPath := fs.String("proof", "tx.json", "proof json file")
	if err := fs.Parse(args); err != nil {
		return verifyTransactionConfig{}, newUsageError("parse verify tx args: %v", err)
	}
	if err := ensureNoPositionalArgs(fs); err != nil {
		return verifyTransactionConfig{}, err
	}

	return verifyTransactionConfig{ProofPath: *proofPath}, nil
}

func buildReceiptExpectations(expectEmitterHex string, expectDataHex string, topics multiStringFlag) (*proof.ReceiptExpectations, error) {
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

func ensureNoPositionalArgs(fs *flag.FlagSet) error {
	if fs.NArg() == 0 {
		return nil
	}
	return newUsageError("%s does not accept positional arguments: %s", fs.Name(), strings.Join(fs.Args(), " "))
}

func validateRPCInputs(rpcURLs []string, minRPCs int) error {
	if len(rpcURLs) == 0 {
		return newUsageError("at least one --rpc is required")
	}
	if minRPCs < 1 {
		return newUsageError("--min-rpcs must be at least 1")
	}
	if len(rpcURLs) < minRPCs {
		return newUsageError("--min-rpcs=%d requires at least %d --rpc values, got %d", minRPCs, minRPCs, len(rpcURLs))
	}
	return nil
}

func proofMinRPCsDefault() int {
	return 3
}
