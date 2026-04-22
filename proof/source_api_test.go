package proof

import (
	"context"
	"fmt"
	"math/big"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/islishude/ethproof/internal/proofutil"
)

func TestGenerateStateProofFromSources(t *testing.T) {
	req, verifyReq, wantNames := testStateProofSourcesRequest(t)

	pkg, err := GenerateStateProofFromSources(context.Background(), req)
	if err != nil {
		t.Fatalf("GenerateStateProofFromSources: %v", err)
	}
	if !slices.Equal(pkg.Block.SourceConsensus.RPCs, wantNames) {
		t.Fatalf("unexpected source names: got %v want %v", pkg.Block.SourceConsensus.RPCs, wantNames)
	}
	if err := VerifyStateProofPackage(pkg); err != nil {
		t.Fatalf("VerifyStateProofPackage: %v", err)
	}
	if err := VerifyStateProofPackageAgainstSources(context.Background(), pkg, verifyReq); err != nil {
		t.Fatalf("VerifyStateProofPackageAgainstSources: %v", err)
	}
}

func TestGenerateReceiptProofFromSources(t *testing.T) {
	req, verifyReq, wantNames := testReceiptProofSourcesRequest(t)

	pkg, err := GenerateReceiptProofFromSources(context.Background(), req)
	if err != nil {
		t.Fatalf("GenerateReceiptProofFromSources: %v", err)
	}
	if !slices.Equal(pkg.Block.SourceConsensus.RPCs, wantNames) {
		t.Fatalf("unexpected source names: got %v want %v", pkg.Block.SourceConsensus.RPCs, wantNames)
	}
	if err := VerifyReceiptProofPackage(pkg); err != nil {
		t.Fatalf("VerifyReceiptProofPackage: %v", err)
	}
	if err := VerifyReceiptProofPackageWithExpectationsAgainstSources(context.Background(), pkg, &ReceiptExpectations{
		Emitter: &pkg.Event.Address,
		Topics:  append([]common.Hash(nil), pkg.Event.Topics...),
		Data:    append([]byte(nil), pkg.Event.Data...),
	}, verifyReq); err != nil {
		t.Fatalf("VerifyReceiptProofPackageWithExpectationsAgainstSources: %v", err)
	}
}

func TestGenerateTransactionProofFromSources(t *testing.T) {
	req, verifyReq, wantNames := testTransactionProofSourcesRequest(t)

	pkg, err := GenerateTransactionProofFromSources(context.Background(), req)
	if err != nil {
		t.Fatalf("GenerateTransactionProofFromSources: %v", err)
	}
	if !slices.Equal(pkg.Block.SourceConsensus.RPCs, wantNames) {
		t.Fatalf("unexpected source names: got %v want %v", pkg.Block.SourceConsensus.RPCs, wantNames)
	}
	if err := VerifyTransactionProofPackage(pkg); err != nil {
		t.Fatalf("VerifyTransactionProofPackage: %v", err)
	}
	if err := VerifyTransactionProofPackageAgainstSources(context.Background(), pkg, verifyReq); err != nil {
		t.Fatalf("VerifyTransactionProofPackageAgainstSources: %v", err)
	}
}

type fakeHeaderSource struct {
	name       string
	chainID    *big.Int
	header     *types.Header
	chainIDErr error
	headerErr  error
}

func (s *fakeHeaderSource) SourceName() string {
	return s.name
}

func (s *fakeHeaderSource) ChainID(context.Context) (*big.Int, error) {
	if s.chainIDErr != nil {
		return nil, s.chainIDErr
	}
	if s.chainID == nil {
		return nil, nil
	}
	return new(big.Int).Set(s.chainID), nil
}

func (s *fakeHeaderSource) HeaderByHash(_ context.Context, blockHash common.Hash) (*types.Header, error) {
	if s.headerErr != nil {
		return nil, s.headerErr
	}
	if s.header == nil {
		return nil, fmt.Errorf("header unavailable")
	}
	if got := s.header.Hash(); got != blockHash {
		return nil, fmt.Errorf("unknown block %s", blockHash)
	}
	return cloneHeader(s.header), nil
}

type fakeStateSource struct {
	*fakeHeaderSource
	expectedBlockNumber uint64
	expectedAccount     common.Address
	expectedSlot        common.Hash
	proof               *gethclient.AccountResult
	proofErr            error
}

func (s *fakeStateSource) HeaderByNumber(_ context.Context, blockNumber *big.Int) (*types.Header, error) {
	if blockNumber == nil || blockNumber.Uint64() != s.expectedBlockNumber {
		return nil, fmt.Errorf("unexpected block number %v", blockNumber)
	}
	return cloneHeader(s.header), nil
}

func (s *fakeStateSource) GetProof(_ context.Context, account common.Address, keys []string, blockNumber *big.Int) (*gethclient.AccountResult, error) {
	if s.proofErr != nil {
		return nil, s.proofErr
	}
	if blockNumber == nil || blockNumber.Uint64() != s.expectedBlockNumber {
		return nil, fmt.Errorf("unexpected block number %v", blockNumber)
	}
	if account != s.expectedAccount {
		return nil, fmt.Errorf("unexpected account %s", account)
	}
	if len(keys) != 1 || keys[0] != s.expectedSlot.Hex() {
		return nil, fmt.Errorf("unexpected proof keys %v", keys)
	}
	return cloneAccountResult(s.proof), nil
}

type fakeReceiptSource struct {
	*fakeHeaderSource
	block            *types.Block
	blockErr         error
	blockReceipts    []*types.Receipt
	blockReceiptsErr error
	receiptsByTxHash map[common.Hash]*types.Receipt
	txsByHash        map[common.Hash]*types.Transaction
	pending          bool
	txErr            error
	receiptErr       error
}

func (s *fakeReceiptSource) TransactionByHash(_ context.Context, txHash common.Hash) (*types.Transaction, bool, error) {
	if s.txErr != nil {
		return nil, false, s.txErr
	}
	tx, ok := s.txsByHash[txHash]
	if !ok {
		return nil, false, fmt.Errorf("unknown tx %s", txHash)
	}
	return tx, s.pending, nil
}

func (s *fakeReceiptSource) TransactionReceipt(_ context.Context, txHash common.Hash) (*types.Receipt, error) {
	if s.receiptErr != nil {
		return nil, s.receiptErr
	}
	receipt, ok := s.receiptsByTxHash[txHash]
	if !ok {
		return nil, fmt.Errorf("unknown receipt %s", txHash)
	}
	return cloneReceipt(receipt), nil
}

func (s *fakeReceiptSource) BlockByHash(_ context.Context, blockHash common.Hash) (*types.Block, error) {
	if s.blockErr != nil {
		return nil, s.blockErr
	}
	if s.block == nil {
		return nil, fmt.Errorf("block unavailable")
	}
	if got := s.block.Hash(); got != blockHash {
		return nil, fmt.Errorf("unknown block %s", blockHash)
	}
	return s.block, nil
}

func (s *fakeReceiptSource) BlockReceiptsByHash(_ context.Context, blockHash common.Hash) ([]*types.Receipt, error) {
	if s.blockReceiptsErr != nil {
		return nil, s.blockReceiptsErr
	}
	if s.block == nil || s.block.Hash() != blockHash {
		return nil, fmt.Errorf("unknown block %s", blockHash)
	}
	return cloneReceiptList(s.blockReceipts), nil
}

func testStateProofSourcesRequest(t *testing.T) (StateProofSourcesRequest, VerifySourcesRequest, []string) {
	t.Helper()

	fixture := mustLoadStateFixture(t)
	header := &types.Header{
		Number:      new(big.Int).SetUint64(701),
		ParentHash:  common.HexToHash("0x1111"),
		Root:        fixture.Block.StateRoot,
		TxHash:      common.HexToHash("0x2222"),
		ReceiptHash: common.HexToHash("0x3333"),
		GasLimit:    30_000_000,
		Time:        1,
	}
	balance, err := proofutil.ParseHexBig(fixture.AccountClaim.Balance)
	if err != nil {
		t.Fatalf("ParseHexBig: %v", err)
	}
	proof := &gethclient.AccountResult{
		Address:      fixture.Account,
		AccountProof: encodeProofNodes(fixture.AccountProofNodes),
		Balance:      balance,
		CodeHash:     fixture.AccountClaim.CodeHash,
		Nonce:        fixture.AccountClaim.Nonce,
		StorageHash:  fixture.AccountClaim.StorageRoot,
		StorageProof: []gethclient.StorageResult{{
			Key:   fixture.Slot.Hex(),
			Value: new(big.Int).SetBytes(fixture.StorageValue[:]),
			Proof: encodeProofNodes(fixture.StorageProofNodes),
		}},
	}
	names := []string{"source-a", "source-b", "source-c"}
	sources := make([]StateSource, len(names))
	headerSources := make([]HeaderSource, len(names))
	for i, name := range names {
		source := &fakeStateSource{
			fakeHeaderSource: &fakeHeaderSource{
				name:    name,
				chainID: fixture.Block.ChainID.ToBig(),
				header:  header,
			},
			expectedBlockNumber: header.Number.Uint64(),
			expectedAccount:     fixture.Account,
			expectedSlot:        fixture.Slot,
			proof:               proof,
		}
		sources[i] = source
		headerSources[i] = source
	}
	return StateProofSourcesRequest{
			Sources:       sources,
			MinRPCSources: len(sources),
			BlockNumber:   header.Number.Uint64(),
			Account:       fixture.Account,
			Slot:          fixture.Slot,
		},
		VerifySourcesRequest{
			Sources:       headerSources,
			MinRPCSources: len(headerSources),
		},
		names
}

func testReceiptProofSourcesRequest(t *testing.T) (ReceiptProofSourcesRequest, VerifySourcesRequest, []string) {
	t.Helper()

	sources, txHash, logIndex, _, names := testReceiptSourceSet(t)
	headerSources := make([]HeaderSource, len(sources))
	for i, source := range sources {
		headerSources[i] = source
	}
	return ReceiptProofSourcesRequest{
			Sources:       sources,
			MinRPCSources: len(sources),
			TxHash:        txHash,
			LogIndex:      logIndex,
		},
		VerifySourcesRequest{
			Sources:       headerSources,
			MinRPCSources: len(headerSources),
		},
		names
}

func testTransactionProofSourcesRequest(t *testing.T) (TransactionProofSourcesRequest, VerifySourcesRequest, []string) {
	t.Helper()

	sources, txHash, _, _, names := testReceiptSourceSet(t)
	txSources := make([]TransactionSource, len(sources))
	headerSources := make([]HeaderSource, len(sources))
	for i, source := range sources {
		txSources[i] = source
		headerSources[i] = source
	}
	return TransactionProofSourcesRequest{
			Sources:       txSources,
			MinRPCSources: len(txSources),
			TxHash:        txHash,
		},
		VerifySourcesRequest{
			Sources:       headerSources,
			MinRPCSources: len(headerSources),
		},
		names
}

func testReceiptSourceSet(t *testing.T) ([]ReceiptSource, common.Hash, uint, uint64, []string) {
	t.Helper()

	to0 := common.HexToAddress("0x1000000000000000000000000000000000000001")
	to1 := common.HexToAddress("0x2000000000000000000000000000000000000002")
	tx0 := types.NewTx(&types.LegacyTx{
		Nonce:    1,
		To:       &to0,
		Value:    big.NewInt(7),
		Gas:      21_000,
		GasPrice: big.NewInt(1),
	})
	tx1 := types.NewTx(&types.LegacyTx{
		Nonce:    2,
		To:       &to1,
		Value:    big.NewInt(9),
		Gas:      50_000,
		GasPrice: big.NewInt(2),
		Data:     []byte{0xca, 0xfe},
	})

	targetLog := &types.Log{
		Address: common.HexToAddress("0x3000000000000000000000000000000000000003"),
		Topics: []common.Hash{
			common.HexToHash("0xaaaa"),
			common.HexToHash("0xbbbb"),
		},
		Data: []byte{0x01, 0x02, 0x03},
	}
	receipts := []*types.Receipt{
		{
			Type:              tx0.Type(),
			Status:            types.ReceiptStatusSuccessful,
			CumulativeGasUsed: 21_000,
			TxHash:            tx0.Hash(),
			TransactionIndex:  0,
		},
		{
			Type:              tx1.Type(),
			Status:            types.ReceiptStatusSuccessful,
			CumulativeGasUsed: 42_000,
			TxHash:            tx1.Hash(),
			TransactionIndex:  1,
			Logs:              []*types.Log{targetLog},
		},
	}
	for _, receipt := range receipts {
		receipt.Bloom = types.CreateBloom(receipt)
	}

	header := &types.Header{
		Number:     big.NewInt(99),
		ParentHash: common.HexToHash("0x4444"),
		Root:       common.HexToHash("0x5555"),
		GasLimit:   30_000_000,
		Time:       2,
	}
	block := types.NewBlock(header, &types.Body{Transactions: []*types.Transaction{tx0, tx1}}, receipts, trie.NewStackTrie(nil))
	blockHash := block.Hash()
	blockReceipts := cloneReceiptList(receipts)
	for i := range blockReceipts {
		blockReceipts[i].BlockHash = blockHash
		blockReceipts[i].TxHash = []*types.Transaction{tx0, tx1}[i].Hash()
		blockReceipts[i].TransactionIndex = uint(i)
	}

	txsByHash := map[common.Hash]*types.Transaction{
		tx0.Hash(): tx0,
		tx1.Hash(): tx1,
	}
	receiptsByHash := map[common.Hash]*types.Receipt{
		tx0.Hash(): blockReceipts[0],
		tx1.Hash(): blockReceipts[1],
	}
	names := []string{"source-a", "source-b", "source-c"}
	sources := make([]ReceiptSource, len(names))
	for i, name := range names {
		sources[i] = &fakeReceiptSource{
			fakeHeaderSource: &fakeHeaderSource{
				name:    name,
				chainID: big.NewInt(1337),
				header:  block.Header(),
			},
			block:         block,
			blockReceipts: blockReceipts,
			receiptsByTxHash: map[common.Hash]*types.Receipt{
				tx0.Hash(): cloneReceipt(receiptsByHash[tx0.Hash()]),
				tx1.Hash(): cloneReceipt(receiptsByHash[tx1.Hash()]),
			},
			txsByHash: txsByHash,
		}
	}
	return sources, tx1.Hash(), 0, 1, names
}

func encodeProofNodes(nodes []hexutil.Bytes) []string {
	out := make([]string, len(nodes))
	for i, node := range nodes {
		out[i] = hexutil.Encode(node)
	}
	return out
}

func cloneAccountResult(in *gethclient.AccountResult) *gethclient.AccountResult {
	if in == nil {
		return nil
	}
	out := *in
	out.AccountProof = append([]string(nil), in.AccountProof...)
	if in.Balance != nil {
		out.Balance = new(big.Int).Set(in.Balance)
	}
	out.StorageProof = make([]gethclient.StorageResult, len(in.StorageProof))
	for i, proof := range in.StorageProof {
		out.StorageProof[i] = gethclient.StorageResult{
			Key:   proof.Key,
			Proof: append([]string(nil), proof.Proof...),
		}
		if proof.Value != nil {
			out.StorageProof[i].Value = new(big.Int).Set(proof.Value)
		}
	}
	return &out
}

func cloneHeader(in *types.Header) *types.Header {
	if in == nil {
		return nil
	}
	out := *in
	if in.Number != nil {
		out.Number = new(big.Int).Set(in.Number)
	}
	if in.BaseFee != nil {
		out.BaseFee = new(big.Int).Set(in.BaseFee)
	}
	if in.Difficulty != nil {
		out.Difficulty = new(big.Int).Set(in.Difficulty)
	}
	out.Extra = append([]byte(nil), in.Extra...)
	return &out
}

func cloneReceipt(in *types.Receipt) *types.Receipt {
	if in == nil {
		return nil
	}
	out := *in
	out.Logs = make([]*types.Log, len(in.Logs))
	for i, log := range in.Logs {
		if log == nil {
			continue
		}
		logCopy := *log
		logCopy.Topics = append([]common.Hash(nil), log.Topics...)
		logCopy.Data = append([]byte(nil), log.Data...)
		out.Logs[i] = &logCopy
	}
	return &out
}

func cloneReceiptList(in []*types.Receipt) []*types.Receipt {
	out := make([]*types.Receipt, len(in))
	for i, receipt := range in {
		out[i] = cloneReceipt(receipt)
	}
	return out
}
