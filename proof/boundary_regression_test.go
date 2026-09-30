package proof

import (
	"math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
	"github.com/islishude/ethproof/internal/proofutil"
)

func TestStateProofEmptyStorageTrie(t *testing.T) {
	account := common.HexToAddress("0x1234")
	stateAccount := types.StateAccount{Balance: uint256.NewInt(1), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()}
	encoded, err := rlp.EncodeToBytes(&stateAccount)
	if err != nil {
		t.Fatal(err)
	}
	tr := proofutil.MakeProofTrie()
	if err := tr.Update(crypto.Keccak256(account.Bytes()), encoded); err != nil {
		t.Fatal(err)
	}
	db := memorydb.New()
	if err := tr.Prove(crypto.Keccak256(account.Bytes()), db); err != nil {
		t.Fatal(err)
	}
	nodes, err := proofutil.DumpProofNodes(db)
	if err != nil {
		t.Fatal(err)
	}
	pkg := StateProofPackage{
		Block:   BlockContext{ChainID: uint256.NewInt(1), StateRoot: tr.Hash()},
		Account: account, AccountRLP: encoded, AccountProofNodes: nodes,
		AccountClaim:  StateAccountClaim{Balance: "0x1", StorageRoot: types.EmptyRootHash, CodeHash: types.EmptyCodeHash},
		StorageProofs: []StateStorageProof{{Slot: common.HexToHash("0x1")}},
	}
	if err := VerifyStateProofPackageAgainstBlockAnchor(&pkg, anchorForTest(pkg.Block)); err != nil {
		t.Fatalf("empty storage: %v", err)
	}

	source := &fakeStateSource{
		fakeHeaderSource: &fakeHeaderSource{name: "empty-storage", chainID: big.NewInt(1), header: &types.Header{Number: big.NewInt(0), Root: pkg.Block.StateRoot}},
		expectedAccount:  account, expectedSlots: []common.Hash{pkg.StorageProofs[0].Slot},
		proof: &gethclient.AccountResult{Address: account, AccountProof: encodeProofNodes(pkg.AccountProofNodes), Balance: big.NewInt(1), StorageHash: types.EmptyRootHash, CodeHash: types.EmptyCodeHash, StorageProof: storageResultsFromFixture(pkg)},
	}
	if _, err := GenerateStateProofFromSources(t.Context(), StateProofSourcesRequest{Sources: []StateSource{source}, MinRPCSources: 1, Account: account, Slots: source.expectedSlots}); err != nil {
		t.Fatalf("generate empty storage proof: %v", err)
	}
	pkg.StorageProofs[0].Value = common.HexToHash("0x1")
	if err := VerifyStateProofPackageAgainstEmbeddedRoots(&pkg); err == nil {
		t.Fatal("accepted nonzero value under empty root")
	}
	if err := verifyStorageProof(common.HexToHash("0x123"), common.Hash{}, nil, common.Hash{}); err == nil {
		t.Fatal("accepted missing proof for nonempty root")
	}
}

func TestStateSourcesRejectWrongRequestedHeight(t *testing.T) {
	req, _, _ := testStateProofSourcesRequest(t)
	for _, source := range req.Sources {
		source.(*fakeStateSource).header.Number = big.NewInt(702)
	}
	if pkg, err := GenerateStateProofFromSources(t.Context(), req); err == nil {
		t.Fatalf("requested %d, accepted %d", req.BlockNumber, pkg.Block.BlockNumber)
	}
}

func TestBlockAnchorHeaderNumberBounds(t *testing.T) {
	for _, n := range []*big.Int{nil, big.NewInt(-1), new(big.Int).Lsh(big.NewInt(1), 64)} {
		if _, err := BlockAnchorFromHeader(big.NewInt(1), &types.Header{Number: n}); err == nil {
			t.Fatalf("accepted invalid height %v", n)
		}
	}
	for _, n := range []uint64{0, ^uint64(0)} {
		anchor, err := BlockAnchorFromHeader(big.NewInt(1), &types.Header{Number: new(big.Int).SetUint64(n)})
		if err != nil || anchor.BlockNumber != n {
			t.Fatalf("height %d: %+v, %v", n, anchor, err)
		}
	}
}

func mappingLayoutForTest(key StorageLayoutType) *StorageLayout {
	return &StorageLayout{Storage: []StorageLayoutEntry{{Label: "m", Slot: "0", Type: "map"}}, Types: map[string]StorageLayoutType{
		"map": {Encoding: "mapping", Label: "mapping", Key: "key", Value: "val", NumberOfBytes: "32"},
		"key": key, "val": {Encoding: "inplace", Label: "uint256", NumberOfBytes: "32"},
	}}
}

func TestMappingIntegerKeyWidthValidation(t *testing.T) {
	for _, tt := range []struct{ label, width, key string }{
		{"uint264", "33", "0x1" + strings.Repeat("0", 64)},
		{"uint256", "33", "0"}, {"uint256", "0", "0"},
		{"uint256", "18446744073709551615", "0"},
		{"uint8", "32", "256"}, {"int8", "32", "128"},
	} {
		t.Run(tt.label+"_"+tt.width, func(t *testing.T) {
			layout := mappingLayoutForTest(StorageLayoutType{Encoding: "inplace", Label: tt.label, NumberOfBytes: tt.width})
			if _, err := ResolveStorageSlots(layout, "m["+tt.key+"]"); err == nil {
				t.Fatal("accepted invalid integer width")
			}
		})
	}
}

func TestMappingStringKeyDelimiters(t *testing.T) {
	layout := mappingLayoutForTest(StorageLayoutType{Encoding: "bytes", Label: "string", NumberOfBytes: "32"})
	for _, key := range []string{"a]b", "[", `a\"b]c`, "a\\]b", "", "a'b"} {
		t.Run(key, func(t *testing.T) {
			resolved, err := ResolveStorageSlots(layout, "m["+strconv.Quote(key)+"]")
			if err != nil {
				t.Fatal(err)
			}
			want := crypto.Keccak256Hash(append([]byte(key), make([]byte, 32)...))
			if resolved.HeadSlot != want {
				t.Fatalf("got %s want %s", resolved.HeadSlot, want)
			}
		})
	}
	resolved, err := ResolveStorageSlots(layout, `m['a]b']`)
	if err != nil || resolved.HeadSlot != crypto.Keccak256Hash(append([]byte("a]b"), make([]byte, 32)...)) {
		t.Fatalf("single quoted key: %v", err)
	}
	for _, query := range []string{`m["a]b]`, `m['a]b]`, `m["a]b"`, `m["a"]tail`, `m["a"junk]`} {
		if _, err := ResolveStorageSlots(layout, query); err == nil {
			t.Fatalf("accepted malformed query %s", query)
		}
	}
}

func TestMappingPaddingRejectsOversizedInput(t *testing.T) {
	if _, err := leftPadBytes(make([]byte, 33)); err == nil {
		t.Fatal("accepted oversized mapping key")
	}
}
