package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/islishude/ethproof/proof"
)

func main() {
	outDir := flag.String("out-dir", "proof/testdata", "output directory for offline fixtures")
	flag.Parse()

	fixtures, err := proof.BuildOfflineFixtures()
	if err != nil {
		log.Fatalf("build offline fixtures: %v", err)
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatalf("create output dir: %v", err)
	}
	if err := proof.SaveJSON(filepath.Join(*outDir, "state_fixture.json"), fixtures.State); err != nil {
		log.Fatalf("write state fixture: %v", err)
	}
	if err := proof.SaveJSON(filepath.Join(*outDir, "receipt_fixture.json"), fixtures.Receipt); err != nil {
		log.Fatalf("write receipt fixture: %v", err)
	}
	if err := proof.SaveJSON(filepath.Join(*outDir, "transaction_fixture.json"), fixtures.Transaction); err != nil {
		log.Fatalf("write transaction fixture: %v", err)
	}
}
