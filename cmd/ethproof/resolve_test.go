package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/islishude/ethproof/proof"
)

func TestRunResolveSlotWritesJSONToStdout(t *testing.T) {
	var exit int
	stdout, stderr := captureCommandOutput(t, func() {
		exit = runMain([]string{
			"resolve", "slot",
			"--compiler-output", resolveFixturePath("storage_layout_fixture.json"),
			"--contract", "Fixture",
			"--var", "data[4][9].b",
		})
	})

	if exit != 0 {
		t.Fatalf("expected exit code 0, got %d", exit)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var got proof.StorageSlotResolution
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("unmarshal stdout json: %v\nstdout=%s", err, stdout)
	}
	if got.TypeLabel != "uint256" {
		t.Fatalf("unexpected type label: %s", got.TypeLabel)
	}
	if len(got.Slots) != 1 {
		t.Fatalf("expected 1 resolved slot, got %d", len(got.Slots))
	}
	if got.Slots[0].Label != "data[4][9].b" {
		t.Fatalf("unexpected slot label: %s", got.Slots[0].Label)
	}
}

func TestRunResolveSlotWritesJSONToFile(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "slot.json")

	var exit int
	stdout, stderr := captureCommandOutput(t, func() {
		exit = runMain([]string{
			"resolve", "slot",
			"--compiler-output", resolveFixturePath("storage_layout_artifact_fixture.json"),
			"--contract", "Fixture",
			"--format", "artifact",
			"--var", "blob@word(1)",
			"--out", outPath,
		})
	})

	if exit != 0 {
		t.Fatalf("expected exit code 0, got %d", exit)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	b, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	var got proof.StorageSlotResolution
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal output file: %v", err)
	}
	if got.TypeLabel != "string" {
		t.Fatalf("unexpected type label: %s", got.TypeLabel)
	}
	if len(got.Slots) != 1 || got.Slots[0].Label != "blob@word(1)" {
		t.Fatalf("unexpected resolved slots: %+v", got.Slots)
	}
}

func TestRunResolveSlotArtifactMissingStorageLayout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifact.json")
	if err := os.WriteFile(path, []byte(`{"abi":[]}`), 0o644); err != nil {
		t.Fatalf("write artifact fixture: %v", err)
	}

	var exit int
	stdout, stderr := captureCommandOutput(t, func() {
		exit = runMain([]string{
			"resolve", "slot",
			"--compiler-output", path,
			"--contract", "Fixture",
			"--format", "artifact",
			"--var", "value",
		})
	})

	if exit != 1 {
		t.Fatalf("expected exit code 1, got %d", exit)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "storageLayout not found in artifact") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func resolveFixturePath(name string) string {
	return filepath.Join("..", "..", "proof", "testdata", name)
}
