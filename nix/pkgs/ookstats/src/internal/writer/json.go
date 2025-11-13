package writer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WriteJSONFile writes a value as JSON to a file atomically.
// It creates a temporary file first, then renames it to ensure atomic writes.
// The JSON is pretty-printed with 2-space indentation and HTML escaping disabled.
func WriteJSONFile(path string, v any) error {
	// Ensure parent directory exists (robust for all callers)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}

	// Create a temp file in the target directory to avoid cross-filesystem issues
	tmpFile, err := os.CreateTemp(filepath.Dir(path), ".tmp-*.json")
	if err != nil {
		return fmt.Errorf("create temp for %s: %w", path, err)
	}
	tmp := tmpFile.Name()

	// Encode JSON with pretty printing
	enc := json.NewEncoder(tmpFile)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		tmpFile.Close()
		os.Remove(tmp)
		return fmt.Errorf("encode json %s: %w", path, err)
	}

	// Flush to disk
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		os.Remove(tmp)
		return fmt.Errorf("sync temp %s: %w", tmp, err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("close temp %s: %w", tmp, err)
	}

	// Atomic replace; add a short retry in case of transient fs race
	if err := os.Rename(tmp, path); err != nil {
		// Retry once after ensuring parent dir again
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
		if err2 := os.Rename(tmp, path); err2 != nil {
			os.Remove(tmp)
			return fmt.Errorf("rename %s: %w", path, err2)
		}
	}

	return nil
}

// WriteJSONFileCompact writes JSON in compact format (no indentation)
func WriteJSONFileCompact(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(path), ".tmp-*.json")
	if err != nil {
		return fmt.Errorf("create temp for %s: %w", path, err)
	}
	tmp := tmpFile.Name()

	enc := json.NewEncoder(tmpFile)
	enc.SetEscapeHTML(false)
	// No SetIndent call = compact JSON
	if err := enc.Encode(v); err != nil {
		tmpFile.Close()
		os.Remove(tmp)
		return fmt.Errorf("encode json %s: %w", path, err)
	}

	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		os.Remove(tmp)
		return fmt.Errorf("sync temp %s: %w", tmp, err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("close temp %s: %w", tmp, err)
	}

	if err := os.Rename(tmp, path); err != nil {
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
		if err2 := os.Rename(tmp, path); err2 != nil {
			os.Remove(tmp)
			return fmt.Errorf("rename %s: %w", path, err2)
		}
	}

	return nil
}

// EnsureDir creates a directory if it doesn't exist
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}
