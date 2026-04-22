package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/islishude/ethproof/proof"
)

func TestRunResolveSlotSuccessPaths(t *testing.T) {
	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "writes json to stdout",
			run: func(t *testing.T) {
				var exit int
				stdout, stderr := captureCommandOutput(t, func() {
					exit = runMain([]string{
						"resolve", "slot",
						"--compiler-output", resolveFixturePath("storage_layout_fixture.json"),
						"--contract", "Fixture",
						"--var", "data[4][9].b",
					})
				})
				if exit != 0 || stderr != "" {
					t.Fatalf("unexpected result: exit=%d stderr=%q", exit, stderr)
				}
				var got proof.StorageSlotResolution
				if err := json.Unmarshal([]byte(stdout), &got); err != nil {
					t.Fatalf("unmarshal stdout json: %v\nstdout=%s", err, stdout)
				}
				if got.TypeLabel != "uint256" || len(got.Slots) != 1 || got.Slots[0].Label != "data[4][9].b" {
					t.Fatalf("unexpected resolution: %+v", got)
				}
			},
		},
		{
			name: "writes json to file",
			run: func(t *testing.T) {
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
				if exit != 0 || stdout != "" || stderr != "" {
					t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
				}
				b, err := os.ReadFile(outPath)
				if err != nil {
					t.Fatalf("read output file: %v", err)
				}
				var got proof.StorageSlotResolution
				if err := json.Unmarshal(b, &got); err != nil {
					t.Fatalf("unmarshal output file: %v", err)
				}
				if got.TypeLabel != "string" || len(got.Slots) != 1 || got.Slots[0].Label != "blob@word(1)" {
					t.Fatalf("unexpected resolution: %+v", got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}

func TestRunResolveSlotRuntimeErrors(t *testing.T) {
	t.Run("artifact missing storage layout", func(t *testing.T) {
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

		if exit != 1 || stdout != "" || !strings.Contains(stderr, "storageLayout not found in artifact") {
			t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
		}
	})

	t.Run("resolver error", func(t *testing.T) {
		var exit int
		stdout, stderr := captureCommandOutput(t, func() {
			exit = runMain([]string{
				"resolve", "slot",
				"--compiler-output", resolveFixturePath("storage_layout_fixture.json"),
				"--contract", "Fixture",
				"--var", "data[4][9].missing",
			})
		})

		if exit != 1 || stdout != "" {
			t.Fatalf("unexpected result: exit=%d stdout=%q", exit, stdout)
		}
		if !strings.Contains(stderr, "resolve storage slot") {
			t.Fatalf("unexpected stderr: %s", stderr)
		}
	})
}

func resolveFixturePath(name string) string {
	return filepath.Join("..", "..", "proof", "testdata", name)
}
