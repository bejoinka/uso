package config

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// MigrateResult describes what happened during migration.
type MigrateResult struct {
	Tool       string
	Copied     bool
	Symlinked  bool
	WasSymlink bool
	Error      error
}

// MigrateTool checks if a tool's symlink path is a real directory.
// If so, copies its contents to accountsDir/profile and replaces it with a symlink.
// If it's already a symlink or doesn't exist, does nothing harmful.
func MigrateTool(toolName, symlink, accountsDir, profile string) MigrateResult {
	result := MigrateResult{Tool: toolName}

	info, err := os.Lstat(symlink)
	if err != nil {
		// Doesn't exist — just create the symlink target and link
		profileDir := filepath.Join(accountsDir, profile)
		os.MkdirAll(profileDir, 0755)
		os.Symlink(profileDir, symlink)
		result.Symlinked = true
		return result
	}

	// Already a symlink — nothing to migrate
	if info.Mode()&os.ModeSymlink != 0 {
		result.WasSymlink = true
		return result
	}

	// Real directory — migrate it
	if !info.IsDir() {
		result.Error = fmt.Errorf("%s exists but is not a directory", symlink)
		return result
	}

	profileDir := filepath.Join(accountsDir, profile)
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		result.Error = fmt.Errorf("mkdir %s: %w", profileDir, err)
		return result
	}

	// Copy contents
	if err := copyDir(symlink, profileDir); err != nil {
		result.Error = fmt.Errorf("copy %s -> %s: %w", symlink, profileDir, err)
		return result
	}
	result.Copied = true

	// Rename original to .bak
	backup := symlink + ".bak"
	// Remove existing backup if present
	os.RemoveAll(backup)
	if err := os.Rename(symlink, backup); err != nil {
		result.Error = fmt.Errorf("backup %s: %w", symlink, err)
		return result
	}

	// Create symlink
	if err := os.Symlink(profileDir, symlink); err != nil {
		// Try to restore backup
		os.Rename(backup, symlink)
		result.Error = fmt.Errorf("symlink %s -> %s: %w", symlink, profileDir, err)
		return result
	}
	result.Symlinked = true

	return result
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		// Handle symlinks within the directory
		if d.Type()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			// Remove existing symlink at target if present
			os.Remove(target)
			return os.Symlink(link, target)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		return os.WriteFile(target, data, info.Mode())
	})
}
