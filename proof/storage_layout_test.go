package proof

import (
	"fmt"
	"math/big"
	"slices"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestLoadStorageLayoutFormats(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		contract string
		format   StorageLayoutFormat
	}{
		{
			name:     "raw layout auto",
			path:     fixturePath("storage_layout_fixture.json"),
			contract: "",
			format:   StorageLayoutFormatAuto,
		},
		{
			name:     "artifact auto",
			path:     fixturePath("storage_layout_artifact_fixture.json"),
			contract: "Fixture",
			format:   StorageLayoutFormatAuto,
		},
		{
			name:     "build info bare contract",
			path:     fixturePath("storage_layout_buildinfo_fixture.json"),
			contract: "Fixture",
			format:   StorageLayoutFormatAuto,
		},
		{
			name:     "build info exact selector",
			path:     fixturePath("storage_layout_buildinfo_fixture.json"),
			contract: "contracts/Fixture.sol:Fixture",
			format:   StorageLayoutFormatBuildInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layout, err := LoadStorageLayout(tt.path, tt.contract, tt.format)
			if err != nil {
				t.Fatalf("LoadStorageLayout: %v", err)
			}
			if len(layout.Storage) == 0 {
				t.Fatal("expected non-empty storage layout")
			}
			if _, ok := findStorageLayoutEntry(layout.Storage, "data"); !ok {
				t.Fatal("expected data variable in fixture")
			}
		})
	}
}

func TestParseStorageLayoutJSONRejectsUnsupportedShape(t *testing.T) {
	_, err := ParseStorageLayoutJSON([]byte(`{"abi":[]}`), "Fixture", StorageLayoutFormatAuto)
	if err == nil {
		t.Fatal("expected unsupported shape to fail")
	}
	if !strings.Contains(err.Error(), "unsupported compiler output shape") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveStorageSlots(t *testing.T) {
	layout := mustLoadStorageLayoutFixture(t, "storage_layout_fixture.json", "", StorageLayoutFormatLayout)

	tests := []struct {
		name      string
		query     string
		wantType  string
		wantHead  common.Hash
		wantSlots []ResolvedStorageSlot
	}{
		{
			name:     "simple variable",
			query:    "value",
			wantType: "uint256",
			wantHead: common.BigToHash(big.NewInt(0)),
			wantSlots: []ResolvedStorageSlot{{
				Slot:   common.BigToHash(big.NewInt(0)),
				Offset: 0,
				Bytes:  32,
				Label:  "value",
				Type:   "uint256",
			}},
		},
		{
			name:     "packed top-level variable",
			query:    "packedB",
			wantType: "uint128",
			wantHead: common.BigToHash(big.NewInt(1)),
			wantSlots: []ResolvedStorageSlot{{
				Slot:   common.BigToHash(big.NewInt(1)),
				Offset: 16,
				Bytes:  16,
				Label:  "packedB",
				Type:   "uint128",
			}},
		},
		{
			name:     "struct member",
			query:    "config.owner",
			wantType: "address",
			wantHead: common.BigToHash(big.NewInt(2)),
			wantSlots: []ResolvedStorageSlot{{
				Slot:   common.BigToHash(big.NewInt(2)),
				Offset: 0,
				Bytes:  20,
				Label:  "config.owner",
				Type:   "address",
			}},
		},
		{
			name:     "struct expansion",
			query:    "config",
			wantType: "struct Fixture.Config",
			wantHead: common.BigToHash(big.NewInt(2)),
			wantSlots: []ResolvedStorageSlot{
				{
					Slot:   common.BigToHash(big.NewInt(2)),
					Offset: 0,
					Bytes:  20,
					Label:  "config.owner",
					Type:   "address",
				},
				{
					Slot:   common.BigToHash(big.NewInt(3)),
					Offset: 0,
					Bytes:  8,
					Label:  "config.limit",
					Type:   "uint64",
				},
			},
		},
		{
			name:     "static array element",
			query:    "fixeds[1]",
			wantType: "uint256",
			wantHead: common.BigToHash(big.NewInt(5)),
			wantSlots: []ResolvedStorageSlot{{
				Slot:   common.BigToHash(big.NewInt(5)),
				Offset: 0,
				Bytes:  32,
				Label:  "fixeds[1]",
				Type:   "uint256",
			}},
		},
		{
			name:     "packed static array expansion",
			query:    "smalls",
			wantType: "uint128[3]",
			wantHead: common.BigToHash(big.NewInt(6)),
			wantSlots: []ResolvedStorageSlot{
				{
					Slot:   common.BigToHash(big.NewInt(6)),
					Offset: 0,
					Bytes:  16,
					Label:  "smalls[0]",
					Type:   "uint128",
				},
				{
					Slot:   common.BigToHash(big.NewInt(6)),
					Offset: 16,
					Bytes:  16,
					Label:  "smalls[1]",
					Type:   "uint128",
				},
				{
					Slot:   common.BigToHash(big.NewInt(7)),
					Offset: 0,
					Bytes:  16,
					Label:  "smalls[2]",
					Type:   "uint128",
				},
			},
		},
		{
			name:     "nested mapping struct member",
			query:    "data[4][9].b",
			wantType: "uint256",
			wantHead: bigToHash(t, new(big.Int).Add(mappingSlot(t, 9, mappingSlot(t, 4, big.NewInt(9))), big.NewInt(1))),
			wantSlots: []ResolvedStorageSlot{{
				Slot:   bigToHash(t, new(big.Int).Add(mappingSlot(t, 9, mappingSlot(t, 4, big.NewInt(9))), big.NewInt(1))),
				Offset: 0,
				Bytes:  32,
				Label:  "data[4][9].b",
				Type:   "uint256",
			}},
		},
		{
			name:     "nested dynamic array element",
			query:    "grid[1][2]",
			wantType: "uint24",
			wantHead: bigToHash(t, nestedDynamicArraySlot(t, big.NewInt(13), 1, 2)),
			wantSlots: []ResolvedStorageSlot{{
				Slot:   bigToHash(t, nestedDynamicArraySlot(t, big.NewInt(13), 1, 2)),
				Offset: 6,
				Bytes:  3,
				Label:  "grid[1][2]",
				Type:   "uint24",
			}},
		},
		{
			name:     "dynamic array struct member",
			query:    "users[2].score",
			wantType: "uint128",
			wantHead: bigToHash(t, new(big.Int).Add(hashSlotBigInt(t, big.NewInt(11)), big.NewInt(2))),
			wantSlots: []ResolvedStorageSlot{{
				Slot:   bigToHash(t, new(big.Int).Add(hashSlotBigInt(t, big.NewInt(11)), big.NewInt(2))),
				Offset: 16,
				Bytes:  16,
				Label:  "users[2].score",
				Type:   "uint128",
			}},
		},
		{
			name:     "bytes head slot",
			query:    "blob",
			wantType: "string",
			wantHead: common.BigToHash(big.NewInt(12)),
			wantSlots: []ResolvedStorageSlot{{
				Slot:   common.BigToHash(big.NewInt(12)),
				Offset: 0,
				Bytes:  32,
				Label:  "blob",
				Type:   "string",
			}},
		},
		{
			name:     "bytes word slot",
			query:    "blob@word(1)",
			wantType: "string",
			wantHead: common.BigToHash(big.NewInt(12)),
			wantSlots: []ResolvedStorageSlot{{
				Slot:   bigToHash(t, new(big.Int).Add(hashSlotBigInt(t, big.NewInt(12)), big.NewInt(1))),
				Offset: 0,
				Bytes:  32,
				Label:  "blob@word(1)",
				Type:   "string",
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveStorageSlots(layout, tt.query)
			if err != nil {
				t.Fatalf("ResolveStorageSlots(%q): %v", tt.query, err)
			}
			if got.TypeLabel != tt.wantType {
				t.Fatalf("unexpected type label: got %s want %s", got.TypeLabel, tt.wantType)
			}
			if got.HeadSlot != tt.wantHead {
				t.Fatalf("unexpected head slot: got %s want %s", got.HeadSlot, tt.wantHead)
			}
			if len(got.Slots) != len(tt.wantSlots) {
				t.Fatalf("unexpected slot count: got %d want %d", len(got.Slots), len(tt.wantSlots))
			}
			for i := range tt.wantSlots {
				if got.Slots[i] != tt.wantSlots[i] {
					t.Fatalf("unexpected slot[%d]: got %+v want %+v", i, got.Slots[i], tt.wantSlots[i])
				}
			}
			wantProofSlots := uniqueProofSlots(tt.wantSlots)
			if !slices.Equal(got.ProofSlots, wantProofSlots) {
				t.Fatalf("unexpected proof slots: got %v want %v", got.ProofSlots, wantProofSlots)
			}
		})
	}
}

func TestResolveStorageSlotsERC7201CustomLayout(t *testing.T) {
	layout := mustLoadStorageLayoutFixture(t, "storage_layout_erc7201_fixture.json", "", StorageLayoutFormatLayout)

	tests := []struct {
		name     string
		query    string
		wantSlot common.Hash
	}{
		{
			name:     "first variable starts at erc7201 namespace base",
			query:    "x",
			wantSlot: common.HexToHash("0x9f96f1285fecaf7ff903f9bcb9e24bb62cf61a391840765cfc133c40cd812700"),
		},
		{
			name:     "second variable follows the namespace base",
			query:    "y",
			wantSlot: common.HexToHash("0x9f96f1285fecaf7ff903f9bcb9e24bb62cf61a391840765cfc133c40cd812701"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveStorageSlots(layout, tt.query)
			if err != nil {
				t.Fatalf("ResolveStorageSlots(%q): %v", tt.query, err)
			}
			if got.TypeLabel != "uint256" {
				t.Fatalf("unexpected type label: got %s want uint256", got.TypeLabel)
			}
			if got.HeadSlot != tt.wantSlot {
				t.Fatalf("unexpected head slot: got %s want %s", got.HeadSlot, tt.wantSlot)
			}
			if len(got.Slots) != 1 {
				t.Fatalf("unexpected slot count: got %d want 1", len(got.Slots))
			}

			want := ResolvedStorageSlot{
				Slot:   tt.wantSlot,
				Offset: 0,
				Bytes:  32,
				Label:  tt.query,
				Type:   "uint256",
			}
			if got.Slots[0] != want {
				t.Fatalf("unexpected slot: got %+v want %+v", got.Slots[0], want)
			}
			if !slices.Equal(got.ProofSlots, []common.Hash{tt.wantSlot}) {
				t.Fatalf("unexpected proof slots: %v", got.ProofSlots)
			}
		})
	}
}

func TestLoadStorageLayoutValidatesArtifactContractSelector(t *testing.T) {
	path := fixturePath("storage_layout_artifact_fixture.json")
	for _, selector := range []string{"Fixture", "contracts/Fixture.sol:Fixture"} {
		if _, err := LoadStorageLayout(path, selector, StorageLayoutFormatArtifact); err != nil {
			t.Fatalf("valid selector %q rejected: %v", selector, err)
		}
	}
	for _, selector := range []string{"", "Wrong", "contracts/Wrong.sol:Fixture"} {
		if _, err := LoadStorageLayout(path, selector, StorageLayoutFormatArtifact); err == nil {
			t.Fatalf("invalid selector %q accepted", selector)
		}
	}
	if _, err := LoadStorageLayout(fixturePath("storage_layout_fixture.json"), "Fixture", StorageLayoutFormatLayout); err == nil {
		t.Fatal("expected raw layout contract selector to fail")
	}
	withoutMetadata := []byte(`{"storageLayout":{"storage":[],"types":{}}}`)
	if _, err := ParseStorageLayoutJSON(withoutMetadata, "Fixture", StorageLayoutFormatArtifact); err == nil || !strings.Contains(err.Error(), "compilationTarget") {
		t.Fatalf("unexpected missing metadata error: %v", err)
	}
}

func TestResolveStorageSlotsRejectsMalformedBytesMappingKeys(t *testing.T) {
	layout := &StorageLayout{
		Storage: []StorageLayoutEntry{
			{Label: "dynamic", Slot: "0", Type: "dynamic-map"},
			{Label: "fixed", Slot: "1", Type: "fixed-map"},
		},
		Types: map[string]StorageLayoutType{
			"dynamic-map": {Encoding: "mapping", Label: "mapping(bytes => uint256)", Key: "bytes", Value: "uint"},
			"fixed-map":   {Encoding: "mapping", Label: "mapping(bytes1 => uint256)", Key: "bytes1", Value: "uint"},
			"bytes":       {Encoding: "bytes", Label: "bytes", NumberOfBytes: "32"},
			"bytes1":      {Encoding: "inplace", Label: "bytes1", NumberOfBytes: "1"},
			"uint":        {Encoding: "inplace", Label: "uint256", NumberOfBytes: "32"},
		},
	}
	for _, query := range []string{"dynamic[0xzz]", "fixed[0xaazz]"} {
		if _, err := ResolveStorageSlots(layout, query); err == nil {
			t.Fatalf("malformed mapping key accepted for %q", query)
		}
	}
}

func TestResolveStorageSlotsLimitsExpansionAndRejectsCycles(t *testing.T) {
	largeArray := &StorageLayout{
		Storage: []StorageLayoutEntry{{Label: "values", Slot: "0", Type: "array"}},
		Types: map[string]StorageLayoutType{
			"array":   {Encoding: "inplace", Label: "uint8[4097]", NumberOfBytes: "4097", Base: "t_uint8"},
			"t_uint8": {Encoding: "inplace", Label: "uint8", NumberOfBytes: "1"},
		},
	}
	// Static array length is encoded in the compiler type ID, so use the canonical ID here.
	largeArray.Storage[0].Type = "t_array(t_uint8)4097_storage"
	largeArray.Types["t_array(t_uint8)4097_storage"] = largeArray.Types["array"]
	if _, err := ResolveStorageSlots(largeArray, "values"); err == nil || !strings.Contains(err.Error(), "4096") {
		t.Fatalf("unexpected large expansion error: %v", err)
	}

	cyclic := &StorageLayout{
		Storage: []StorageLayoutEntry{{Label: "value", Slot: "0", Type: "self"}},
		Types: map[string]StorageLayoutType{
			"self": {
				Encoding:      "inplace",
				Label:         "struct Self",
				NumberOfBytes: "32",
				Members:       []StorageLayoutEntry{{Label: "next", Slot: "0", Type: "self"}},
			},
		},
	}
	if _, err := ResolveStorageSlots(cyclic, "value"); err == nil || !strings.Contains(err.Error(), "cyclic") {
		t.Fatalf("unexpected cyclic layout error: %v", err)
	}

	deepTypes := make(map[string]StorageLayoutType, maxStorageResolverDepth+1)
	for i := range maxStorageResolverDepth + 1 {
		typeID := fmt.Sprintf("depth-%d", i)
		if i == maxStorageResolverDepth {
			deepTypes[typeID] = StorageLayoutType{Encoding: "inplace", Label: "uint256", NumberOfBytes: "32"}
			continue
		}
		deepTypes[typeID] = StorageLayoutType{
			Encoding:      "inplace",
			Label:         typeID,
			NumberOfBytes: "32",
			Members:       []StorageLayoutEntry{{Label: "next", Slot: "0", Type: fmt.Sprintf("depth-%d", i+1)}},
		}
	}
	deep := &StorageLayout{
		Storage: []StorageLayoutEntry{{Label: "value", Slot: "0", Type: "depth-0"}},
		Types:   deepTypes,
	}
	if _, err := ResolveStorageSlots(deep, "value"); err == nil || !strings.Contains(err.Error(), "maximum depth") {
		t.Fatalf("unexpected deep layout error: %v", err)
	}
}

func TestResolveStorageSlotsRejectsInvalidQueries(t *testing.T) {
	layout := mustLoadStorageLayoutFixture(t, "storage_layout_fixture.json", "", StorageLayoutFormatLayout)

	tests := []struct {
		name  string
		query string
		want  string
	}{
		{name: "unknown variable", query: "missing", want: "unknown variable"},
		{name: "missing mapping key", query: "balances", want: "requires explicit key selector"},
		{name: "missing dynamic index", query: "dynNums", want: "requires explicit index selector"},
		{name: "wrong key literal", query: "balances[nope]", want: "invalid mapping key"},
		{name: "static array out of bounds", query: "fixeds[2]", want: "out of bounds"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveStorageSlots(layout, tt.query)
			if err == nil {
				t.Fatal("expected ResolveStorageSlots to fail")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func mustLoadStorageLayoutFixture(t *testing.T, name string, contract string, format StorageLayoutFormat) *StorageLayout {
	t.Helper()
	layout, err := LoadStorageLayout(fixturePath(name), contract, format)
	if err != nil {
		t.Fatalf("LoadStorageLayout(%s): %v", name, err)
	}
	return layout
}

func bigToHash(t *testing.T, value *big.Int) common.Hash {
	t.Helper()
	hash, err := slotBigIntToHash(value)
	if err != nil {
		t.Fatalf("slotBigIntToHash(%s): %v", value, err)
	}
	return hash
}

func hashSlotBigInt(t *testing.T, slot *big.Int) *big.Int {
	t.Helper()
	slotBytes := common.LeftPadBytes(slot.Bytes(), 32)
	return new(big.Int).SetBytes(crypto.Keccak256(slotBytes))
}

func mappingSlot(t *testing.T, key uint64, slot *big.Int) *big.Int {
	t.Helper()
	keyBytes := common.LeftPadBytes(new(big.Int).SetUint64(key).Bytes(), 32)
	slotBytes := common.LeftPadBytes(slot.Bytes(), 32)
	return new(big.Int).SetBytes(crypto.Keccak256(append(keyBytes, slotBytes...)))
}

func nestedDynamicArraySlot(t *testing.T, slot *big.Int, outerIndex uint64, innerIndex uint64) *big.Int {
	t.Helper()
	outerBase := hashSlotBigInt(t, slot)
	innerHead := new(big.Int).Add(outerBase, new(big.Int).SetUint64(outerIndex))
	innerBase := hashSlotBigInt(t, innerHead)
	slotOffset := new(big.Int).SetUint64(innerIndex / 10)
	return new(big.Int).Add(innerBase, slotOffset)
}
