package proof

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SaveJSON marshals v as indented JSON and writes it to path.
func SaveJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary json file: %w", err)
	}
	tempPath := temp.Name()
	keepTemp := true
	defer func() {
		_ = temp.Close()
		if keepTemp {
			_ = os.Remove(tempPath)
		}
	}()

	if err := temp.Chmod(0o600); err != nil {
		return fmt.Errorf("set temporary json permissions: %w", err)
	}
	if _, err := temp.Write(b); err != nil {
		return fmt.Errorf("write temporary json file: %w", err)
	}
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("sync temporary json file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary json file: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace json file: %w", err)
	}
	keepTemp = false
	return nil
}

// LoadJSON reads JSON from path into v.
func LoadJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}
