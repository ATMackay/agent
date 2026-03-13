package documentor

import (
	"fmt"
	"os"
	"path/filepath"
)

// writeTextFile creates parent directories as needed and writes content to path.
func writeTextFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
