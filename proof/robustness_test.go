package proof

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPublicVerifiersRejectNilPackages(t *testing.T) {
	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{name: "state", run: func() error { return VerifyStateProofPackage(nil) }, want: "state proof package is nil"},
		{name: "receipt", run: func() error { return VerifyReceiptProofPackage(nil) }, want: "receipt proof package is nil"},
		{name: "transaction", run: func() error { return VerifyTransactionProofPackage(nil) }, want: "transaction proof package is nil"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func FuzzVerifyStateProofPackageNeverPanics(f *testing.F) {
	seed, err := json.Marshal(mustLoadStateFixture(f))
	if err != nil {
		f.Fatal(err)
	}
	f.Add(seed)
	f.Add([]byte(`{}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		var pkg StateProofPackage
		if json.Unmarshal(raw, &pkg) == nil {
			_ = VerifyStateProofPackageAgainstEmbeddedRoots(&pkg)
		}
	})
}

func FuzzVerifyReceiptProofPackageNeverPanics(f *testing.F) {
	seed, err := json.Marshal(mustLoadReceiptFixture(f))
	if err != nil {
		f.Fatal(err)
	}
	f.Add(seed)
	f.Add([]byte(`{"logIndex":18446744073709551615}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		var pkg ReceiptProofPackage
		if json.Unmarshal(raw, &pkg) == nil {
			_ = VerifyReceiptProofPackageAgainstEmbeddedRoots(&pkg)
		}
	})
}

func FuzzVerifyTransactionProofPackageNeverPanics(f *testing.F) {
	seed, err := json.Marshal(mustLoadTransactionFixture(f))
	if err != nil {
		f.Fatal(err)
	}
	f.Add(seed)
	f.Add([]byte(`{}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		var pkg TransactionProofPackage
		if json.Unmarshal(raw, &pkg) == nil {
			_ = VerifyTransactionProofPackageAgainstEmbeddedRoots(&pkg)
		}
	})
}
