package proof

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestOfflineFixturesMatchGoldenFiles(t *testing.T) {
	fixtures, err := BuildOfflineFixtures()
	if err != nil {
		t.Fatalf("BuildOfflineFixtures: %v", err)
	}
	cases := []struct {
		name string
		path string
		got  any
	}{
		{name: "state", path: fixturePath("state_fixture.json"), got: fixtures.State},
		{name: "receipt", path: fixturePath("receipt_fixture.json"), got: fixtures.Receipt},
		{name: "transaction", path: fixturePath("transaction_fixture.json"), got: fixtures.Transaction},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want, err := json.MarshalIndent(tc.got, "", "  ")
			if err != nil {
				t.Fatalf("marshal %s fixture: %v", tc.name, err)
			}
			var loaded any
			switch tc.name {
			case "state":
				var pkg StateProofPackage
				loaded = &pkg
			case "receipt":
				var pkg ReceiptProofPackage
				loaded = &pkg
			default:
				var pkg TransactionProofPackage
				loaded = &pkg
			}
			if err := LoadJSON(tc.path, loaded); err != nil {
				t.Fatalf("LoadJSON(%s): %v", tc.path, err)
			}
			have, err := json.MarshalIndent(loaded, "", "  ")
			if err != nil {
				t.Fatalf("marshal loaded %s fixture: %v", tc.name, err)
			}
			if !bytes.Equal(have, want) {
				t.Fatalf("%s fixture file is out of date", tc.name)
			}
		})
	}
}

func TestVerifyOfflineFixtures(t *testing.T) {
	statePkg := mustLoadStateFixture(t)
	if err := VerifyStateProofPackage(&statePkg); err != nil {
		t.Fatalf("VerifyStateProofPackage: %v", err)
	}

	receiptPkg := mustLoadReceiptFixture(t)
	if err := VerifyReceiptProofPackage(&receiptPkg); err != nil {
		t.Fatalf("VerifyReceiptProofPackage: %v", err)
	}

	txPkg := mustLoadTransactionFixture(t)
	if err := VerifyTransactionProofPackage(&txPkg); err != nil {
		t.Fatalf("VerifyTransactionProofPackage: %v", err)
	}
}

func TestTamperedTransactionProofFails(t *testing.T) {
	pkg := mustLoadTransactionFixture(t)

	rootTampered := cloneTransactionPackage(pkg)
	rootTampered.Block.TransactionsRoot = common.HexToHash("0x1234")
	if err := VerifyTransactionProofPackage(&rootTampered); err == nil {
		t.Fatal("expected modified transactions root to fail verification")
	}

	nodeTampered := cloneTransactionPackage(pkg)
	nodeTampered.ProofNodes[0] = mutateHexNode(t, nodeTampered.ProofNodes[0])
	if err := VerifyTransactionProofPackage(&nodeTampered); err == nil {
		t.Fatal("expected modified proof node to fail verification")
	}

	indexTampered := cloneTransactionPackage(pkg)
	indexTampered.TxIndex++
	if err := VerifyTransactionProofPackage(&indexTampered); err == nil {
		t.Fatal("expected modified tx index to fail verification")
	}
}

func TestTamperedReceiptProofFails(t *testing.T) {
	pkg := mustLoadReceiptFixture(t)

	rootTampered := cloneReceiptPackage(pkg)
	rootTampered.Block.ReceiptsRoot = common.HexToHash("0x1234")
	if err := VerifyReceiptProofPackage(&rootTampered); err == nil {
		t.Fatal("expected modified receipts root to fail verification")
	}

	nodeTampered := cloneReceiptPackage(pkg)
	nodeTampered.ProofNodes[0] = mutateHexNode(t, nodeTampered.ProofNodes[0])
	if err := VerifyReceiptProofPackage(&nodeTampered); err == nil {
		t.Fatal("expected modified proof node to fail verification")
	}

	logIndexTampered := cloneReceiptPackage(pkg)
	logIndexTampered.LogIndex++
	if err := VerifyReceiptProofPackage(&logIndexTampered); err == nil {
		t.Fatal("expected modified log index to fail verification")
	}

	dataTampered := cloneReceiptPackage(pkg)
	dataTampered.Event.Data = canonicalHex([]byte{0xaa})
	if err := VerifyReceiptProofPackage(&dataTampered); err == nil {
		t.Fatal("expected modified event data to fail verification")
	}

	topicTampered := cloneReceiptPackage(pkg)
	topicTampered.Event.Topics[0] = common.HexToHash("0xbeef")
	if err := VerifyReceiptProofPackage(&topicTampered); err == nil {
		t.Fatal("expected modified event topic to fail verification")
	}

	addressTampered := cloneReceiptPackage(pkg)
	addressTampered.Event.Address = common.HexToAddress("0x9999999999999999999999999999999999999999")
	if err := VerifyReceiptProofPackage(&addressTampered); err == nil {
		t.Fatal("expected modified event address to fail verification")
	}
}

func TestTamperedStateProofFails(t *testing.T) {
	pkg := mustLoadStateFixture(t)

	rootTampered := cloneStatePackage(pkg)
	rootTampered.Block.StateRoot = common.HexToHash("0x1234")
	if err := VerifyStateProofPackage(&rootTampered); err == nil {
		t.Fatal("expected modified state root to fail verification")
	}

	nodeTampered := cloneStatePackage(pkg)
	nodeTampered.AccountProofNodes[0] = mutateHexNode(t, nodeTampered.AccountProofNodes[0])
	if err := VerifyStateProofPackage(&nodeTampered); err == nil {
		t.Fatal("expected modified account proof node to fail verification")
	}

	storageNodeTampered := cloneStatePackage(pkg)
	storageNodeTampered.StorageProofNodes[0] = mutateHexNode(t, storageNodeTampered.StorageProofNodes[0])
	if err := VerifyStateProofPackage(&storageNodeTampered); err == nil {
		t.Fatal("expected modified storage proof node to fail verification")
	}

	valueTampered := cloneStatePackage(pkg)
	valueTampered.StorageValue = common.HexToHash("0x7777")
	if err := VerifyStateProofPackage(&valueTampered); err == nil {
		t.Fatal("expected modified storage value to fail verification")
	}

	nonceTampered := cloneStatePackage(pkg)
	nonceTampered.AccountClaim.Nonce++
	if err := VerifyStateProofPackage(&nonceTampered); err == nil {
		t.Fatal("expected modified account nonce to fail verification")
	}

	balanceTampered := cloneStatePackage(pkg)
	balanceTampered.AccountClaim.Balance = "0x1"
	if err := VerifyStateProofPackage(&balanceTampered); err == nil {
		t.Fatal("expected modified account balance to fail verification")
	}

	storageRootTampered := cloneStatePackage(pkg)
	storageRootTampered.AccountClaim.StorageRoot = common.HexToHash("0x8888")
	if err := VerifyStateProofPackage(&storageRootTampered); err == nil {
		t.Fatal("expected modified account storage root to fail verification")
	}

	codeHashTampered := cloneStatePackage(pkg)
	codeHashTampered.AccountClaim.CodeHash = common.HexToHash("0x9999")
	if err := VerifyStateProofPackage(&codeHashTampered); err == nil {
		t.Fatal("expected modified account code hash to fail verification")
	}
}

func fixturePath(name string) string {
	return filepath.Join("testdata", name)
}

func mustLoadStateFixture(t *testing.T) StateProofPackage {
	t.Helper()
	var pkg StateProofPackage
	if err := LoadJSON(fixturePath("state_fixture.json"), &pkg); err != nil {
		t.Fatalf("load state fixture: %v", err)
	}
	return pkg
}

func mustLoadReceiptFixture(t *testing.T) ReceiptProofPackage {
	t.Helper()
	var pkg ReceiptProofPackage
	if err := LoadJSON(fixturePath("receipt_fixture.json"), &pkg); err != nil {
		t.Fatalf("load receipt fixture: %v", err)
	}
	return pkg
}

func mustLoadTransactionFixture(t *testing.T) TransactionProofPackage {
	t.Helper()
	var pkg TransactionProofPackage
	if err := LoadJSON(fixturePath("transaction_fixture.json"), &pkg); err != nil {
		t.Fatalf("load transaction fixture: %v", err)
	}
	return pkg
}

func cloneTransactionPackage(in TransactionProofPackage) TransactionProofPackage {
	out := in
	out.ProofNodes = append([]string(nil), in.ProofNodes...)
	return out
}

func cloneReceiptPackage(in ReceiptProofPackage) ReceiptProofPackage {
	out := in
	out.ProofNodes = append([]string(nil), in.ProofNodes...)
	out.Event.Topics = append([]common.Hash(nil), in.Event.Topics...)
	return out
}

func cloneStatePackage(in StateProofPackage) StateProofPackage {
	out := in
	out.AccountProofNodes = append([]string(nil), in.AccountProofNodes...)
	out.StorageProofNodes = append([]string(nil), in.StorageProofNodes...)
	return out
}

func mutateHexNode(t *testing.T, value string) string {
	t.Helper()
	raw, err := decodeHexBytes(value)
	if err != nil {
		t.Fatalf("decode node: %v", err)
	}
	raw[0] ^= 0x01
	return canonicalHex(raw)
}
