package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/islishude/ethproof/proof"
)

func TestOfflineFixturesMatchCheckedInGoldenFiles(t *testing.T) {
	fixtures, err := BuildOfflineFixtures()
	if err != nil {
		t.Fatalf("BuildOfflineFixtures: %v", err)
	}

	cases := []struct {
		name string
		path string
		data any
	}{
		{name: "state", path: "state_fixture.json", data: fixtures.State},
		{name: "receipt", path: "receipt_fixture.json", data: fixtures.Receipt},
		{name: "transaction", path: "transaction_fixture.json", data: fixtures.Transaction},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			outPath := filepath.Join(t.TempDir(), tc.path)
			if err := proof.SaveJSON(outPath, tc.data); err != nil {
				t.Fatalf("SaveJSON(%s): %v", tc.name, err)
			}

			want, err := os.ReadFile(outPath)
			if err != nil {
				t.Fatalf("read generated %s fixture: %v", tc.name, err)
			}
			have, err := os.ReadFile(filepath.Join("..", "..", "proof", "testdata", tc.path))
			if err != nil {
				t.Fatalf("read checked-in %s fixture: %v", tc.name, err)
			}
			if !bytes.Equal(have, want) {
				t.Fatalf("%s fixture file is out of date", tc.name)
			}
		})
	}
}
