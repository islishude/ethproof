package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

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

func (m *multiStringFlag) String() string {
	return fmt.Sprintf("%v", []string(*m))
}

func (m *multiStringFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func main() {
	if len(os.Args) < 3 {
		usage()
	}
	switch os.Args[1] {
	case "generate":
		cmdGenerate(os.Args[2:])
	case "verify":
		cmdVerify(os.Args[2:])
	default:
		usage()
	}
}

func cmdGenerate(args []string) {
	if len(args) == 0 {
		usage()
	}
	switch args[0] {
	case "state":
		cmdGenerateState(args[1:])
	case "receipt":
		cmdGenerateReceipt(args[1:])
	case "tx":
		cmdGenerateTransaction(args[1:])
	default:
		usage()
	}
}

func cmdVerify(args []string) {
	if len(args) == 0 {
		usage()
	}
	switch args[0] {
	case "state":
		cmdVerifyState(args[1:])
	case "receipt":
		cmdVerifyReceipt(args[1:])
	case "tx":
		cmdVerifyTransaction(args[1:])
	default:
		usage()
	}
}

func cmdGenerateState(args []string) {
	cfg := parseGenerateStateArgs(args)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pkg, err := proof.GenerateStateProof(ctx, cfg.Request)
	if err != nil {
		log.Fatalf("generate state proof: %v", err)
	}
	if err := proof.SaveJSON(cfg.Out, pkg); err != nil {
		log.Fatalf("write state proof: %v", err)
	}
	fmt.Printf("state proof written to %s\n", cfg.Out)
	fmt.Printf("block=%d account=%s slot=%s stateRoot=%s\n", pkg.Block.BlockNumber, pkg.Account, pkg.Slot, pkg.Block.StateRoot)
}

func cmdGenerateReceipt(args []string) {
	cfg := parseGenerateReceiptArgs(args)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	pkg, err := proof.GenerateReceiptProof(ctx, cfg.Request)
	if err != nil {
		log.Fatalf("generate receipt proof: %v", err)
	}
	if err := proof.SaveJSON(cfg.Out, pkg); err != nil {
		log.Fatalf("write receipt proof: %v", err)
	}
	fmt.Printf("receipt proof written to %s\n", cfg.Out)
	fmt.Printf("block=%d txIndex=%d receiptsRoot=%s\n", pkg.Block.BlockNumber, pkg.TxIndex, pkg.Block.ReceiptsRoot)
}

func cmdGenerateTransaction(args []string) {
	cfg := parseGenerateTransactionArgs(args)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	pkg, err := proof.GenerateTransactionProof(ctx, cfg.Request)
	if err != nil {
		log.Fatalf("generate transaction proof: %v", err)
	}
	if err := proof.SaveJSON(cfg.Out, pkg); err != nil {
		log.Fatalf("write transaction proof: %v", err)
	}
	fmt.Printf("transaction proof written to %s\n", cfg.Out)
	fmt.Printf("block=%d txIndex=%d transactionsRoot=%s\n", pkg.Block.BlockNumber, pkg.TxIndex, pkg.Block.TransactionsRoot)
}

func cmdVerifyState(args []string) {
	fs := flag.NewFlagSet("verify state", flag.ExitOnError)
	proofPath := fs.String("proof", "state.json", "proof json file")
	if err := fs.Parse(args); err != nil {
		log.Fatalf("parse verify state args: %v", err)
	}

	var pkg proof.StateProofPackage
	if err := proof.LoadJSON(*proofPath, &pkg); err != nil {
		log.Fatalf("read state proof json: %v", err)
	}
	if err := proof.VerifyStateProofPackage(&pkg); err != nil {
		log.Fatalf("verify state proof: %v", err)
	}
	fmt.Println("state proof verification succeeded")
}

func cmdVerifyReceipt(args []string) {
	fs := flag.NewFlagSet("verify receipt", flag.ExitOnError)
	proofPath := fs.String("proof", "receipt.json", "proof json file")
	expectEmitterHex := fs.String("expect-emitter", "", "optional expected emitter address")
	expectDataHex := fs.String("expect-data", "", "optional expected event data hex")
	var topics multiStringFlag
	fs.Var(&topics, "expect-topic", "optional expected topic (repeatable)")
	if err := fs.Parse(args); err != nil {
		log.Fatalf("parse verify receipt args: %v", err)
	}

	var pkg proof.ReceiptProofPackage
	if err := proof.LoadJSON(*proofPath, &pkg); err != nil {
		log.Fatalf("read receipt proof json: %v", err)
	}

	var expect proof.ReceiptExpectations
	if *expectEmitterHex != "" {
		addr := common.HexToAddress(*expectEmitterHex)
		expect.Emitter = &addr
	}
	if *expectDataHex != "" {
		expect.Data = common.FromHex(*expectDataHex)
	}
	for _, topic := range topics {
		expect.Topics = append(expect.Topics, common.HexToHash(topic))
	}

	var expectPtr *proof.ReceiptExpectations
	if expect.Emitter != nil || expect.Data != nil || len(expect.Topics) > 0 {
		expectPtr = &expect
	}
	if err := proof.VerifyReceiptProofPackageWithExpectations(&pkg, expectPtr); err != nil {
		log.Fatalf("verify receipt proof: %v", err)
	}
	fmt.Println("receipt proof verification succeeded")
}

func cmdVerifyTransaction(args []string) {
	fs := flag.NewFlagSet("verify tx", flag.ExitOnError)
	proofPath := fs.String("proof", "tx.json", "proof json file")
	if err := fs.Parse(args); err != nil {
		log.Fatalf("parse verify tx args: %v", err)
	}

	var pkg proof.TransactionProofPackage
	if err := proof.LoadJSON(*proofPath, &pkg); err != nil {
		log.Fatalf("read transaction proof json: %v", err)
	}
	if err := proof.VerifyTransactionProofPackage(&pkg); err != nil {
		log.Fatalf("verify transaction proof: %v", err)
	}
	fmt.Println("transaction proof verification succeeded")
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage:
  ethproof generate state   --rpc URL [--rpc URL ...] --min-rpcs N --block N --account 0xADDR --slot 0xSLOT --out state.json
  ethproof generate receipt --rpc URL [--rpc URL ...] --min-rpcs N --tx 0xHASH --log-index N --out receipt.json
  ethproof generate tx      --rpc URL [--rpc URL ...] --min-rpcs N --tx 0xHASH --out tx.json

  ethproof verify state   --proof state.json
  ethproof verify receipt --proof receipt.json [--expect-emitter 0xADDR] [--expect-topic 0xHASH] [--expect-data 0xDATA]
  ethproof verify tx      --proof tx.json
`)
	os.Exit(2)
}

func parseGenerateStateArgs(args []string) generateStateConfig {
	fs := flag.NewFlagSet("generate state", flag.ExitOnError)
	var rpcURLs multiStringFlag
	fs.Var(&rpcURLs, "rpc", "Ethereum RPC URL")
	minRPCs := fs.Int("min-rpcs", proofMinRPCsDefault(), "minimum distinct RPC sources required")
	blockNumber := fs.Uint64("block", 0, "block number")
	accountHex := fs.String("account", "", "account address")
	slotHex := fs.String("slot", "", "32-byte storage slot key")
	out := fs.String("out", "state.json", "output proof json")
	if err := fs.Parse(args); err != nil {
		log.Fatalf("parse generate state args: %v", err)
	}

	if len(rpcURLs) == 0 || *accountHex == "" || *slotHex == "" || *minRPCs < 1 {
		fs.Usage()
		os.Exit(2)
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
	}
}

func parseGenerateReceiptArgs(args []string) generateReceiptConfig {
	fs := flag.NewFlagSet("generate receipt", flag.ExitOnError)
	var rpcURLs multiStringFlag
	fs.Var(&rpcURLs, "rpc", "Ethereum RPC URL")
	minRPCs := fs.Int("min-rpcs", proofMinRPCsDefault(), "minimum distinct RPC sources required")
	txHashHex := fs.String("tx", "", "transaction hash")
	logIndex := fs.Uint("log-index", 0, "log index within receipt")
	out := fs.String("out", "receipt.json", "output proof json")
	if err := fs.Parse(args); err != nil {
		log.Fatalf("parse generate receipt args: %v", err)
	}

	if len(rpcURLs) == 0 || *txHashHex == "" || *minRPCs < 1 {
		fs.Usage()
		os.Exit(2)
	}
	return generateReceiptConfig{
		Request: proof.ReceiptProofRequest{
			RPCURLs:       rpcURLs,
			MinRPCSources: *minRPCs,
			TxHash:        common.HexToHash(*txHashHex),
			LogIndex:      *logIndex,
		},
		Out: *out,
	}
}

func parseGenerateTransactionArgs(args []string) generateTransactionConfig {
	fs := flag.NewFlagSet("generate tx", flag.ExitOnError)
	var rpcURLs multiStringFlag
	fs.Var(&rpcURLs, "rpc", "Ethereum RPC URL")
	minRPCs := fs.Int("min-rpcs", proofMinRPCsDefault(), "minimum distinct RPC sources required")
	txHashHex := fs.String("tx", "", "transaction hash")
	out := fs.String("out", "tx.json", "output proof json")
	if err := fs.Parse(args); err != nil {
		log.Fatalf("parse generate tx args: %v", err)
	}

	if len(rpcURLs) == 0 || *txHashHex == "" || *minRPCs < 1 {
		fs.Usage()
		os.Exit(2)
	}
	return generateTransactionConfig{
		Request: proof.TransactionProofRequest{
			RPCURLs:       rpcURLs,
			MinRPCSources: *minRPCs,
			TxHash:        common.HexToHash(*txHashHex),
		},
		Out: *out,
	}
}

func proofMinRPCsDefault() int {
	return 3
}
