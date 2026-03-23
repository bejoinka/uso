package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var validName = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// KnownTool is a built-in tool definition offered during `uso init`.
type KnownTool struct {
	ID          string
	EnvVar      string
	Symlink     string // with $HOME expanded
	AccountsDir string // with $HOME expanded
	Label       string
}

// KnownTools returns the built-in tool list.
func KnownTools() []KnownTool {
	home, _ := os.UserHomeDir()
	return []KnownTool{
		{"claude", "CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude"), filepath.Join(home, ".claude-accounts"), "Claude Code"},
		{"codex", "CODEX_HOME", filepath.Join(home, ".codex"), filepath.Join(home, ".codex-accounts"), "Codex CLI"},
	}
}

// Tool represents a registered CLI tool.
type Tool struct {
	EnvVar      string
	Symlink     string
	AccountsDir string
}

// Dir returns the uso config directory.
func Dir() string {
	if d := os.Getenv("USO_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "uso")
}

// ValidName checks if a profile/tool name is valid.
func ValidName(name string) bool {
	return len(name) > 0 && len(name) <= 32 && validName.MatchString(name)
}

// CurrentProfile reads the active profile name.
func CurrentProfile() string {
	data, err := os.ReadFile(filepath.Join(Dir(), "current"))
	if err != nil {
		return "home"
	}
	s := strings.TrimSpace(string(data))
	if s == "" {
		return "home"
	}
	return s
}

// SetCurrentProfile writes the active profile.
func SetCurrentProfile(name string) error {
	return os.WriteFile(filepath.Join(Dir(), "current"), []byte(name+"\n"), 0644)
}

// LoadTools reads all registered tools.
func LoadTools() (map[string]Tool, error) {
	dir := filepath.Join(Dir(), "tools")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	tools := make(map[string]Tool)
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".conf") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".conf")
		t, err := loadTool(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("tool %s: %w", name, err)
		}
		tools[name] = t
	}
	return tools, nil
}

func loadTool(path string) (Tool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Tool{}, err
	}
	var t Tool
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "ENV_VAR":
			t.EnvVar = strings.TrimSpace(v)
		case "SYMLINK":
			t.Symlink = strings.TrimSpace(v)
		case "ACCOUNTS_DIR":
			t.AccountsDir = strings.TrimSpace(v)
		}
	}
	return t, nil
}

// SaveTool writes a tool config.
func SaveTool(name string, t Tool) error {
	dir := filepath.Join(Dir(), "tools")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	content := fmt.Sprintf("ENV_VAR=%s\nSYMLINK=%s\nACCOUNTS_DIR=%s\n", t.EnvVar, t.Symlink, t.AccountsDir)
	return os.WriteFile(filepath.Join(dir, name+".conf"), []byte(content), 0644)
}

// LoadProfileEnv reads a profile's KEY=VALUE pairs.
func LoadProfileEnv(profile string) ([][2]string, error) {
	path := filepath.Join(Dir(), "profiles", profile+".conf")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var pairs [][2]string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		pairs = append(pairs, [2]string{strings.TrimSpace(k), strings.TrimSpace(v)})
	}
	return pairs, nil
}

// ProfileExists checks if a profile config file exists.
func ProfileExists(name string) bool {
	_, err := os.Stat(filepath.Join(Dir(), "profiles", name+".conf"))
	return err == nil
}

// ListProfiles returns all registered profile names.
func ListProfiles() ([]string, error) {
	dir := filepath.Join(Dir(), "profiles")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".conf") {
			names = append(names, strings.TrimSuffix(e.Name(), ".conf"))
		}
	}
	return names, nil
}

// SetProfileKey upserts a key in a profile conf. Empty value removes the key.
func SetProfileKey(profile, key, value string) error {
	dir := filepath.Join(Dir(), "profiles")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, profile+".conf")
	existing, _ := os.ReadFile(path)

	var lines []string
	for _, line := range strings.Split(string(existing), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		k, _, ok := strings.Cut(trimmed, "=")
		if ok && strings.TrimSpace(k) == key {
			continue // remove old value
		}
		lines = append(lines, trimmed)
	}

	if value != "" {
		lines = append(lines, key+"="+value)
	}

	content := ""
	if len(lines) > 0 {
		content = strings.Join(lines, "\n") + "\n"
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// EnsureProfileDir creates the profile conf file and tool account dirs.
func EnsureProfileDir(profile string, tools map[string]Tool) error {
	dir := filepath.Join(Dir(), "profiles")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, profile+".conf")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, nil, 0644); err != nil {
			return err
		}
	}

	for _, t := range tools {
		if err := os.MkdirAll(filepath.Join(t.AccountsDir, profile), 0755); err != nil {
			return err
		}
	}
	return nil
}

// EnsureInit creates the base directory structure.
func EnsureInit() error {
	d := Dir()
	if err := os.MkdirAll(filepath.Join(d, "tools"), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(d, "profiles"), 0755); err != nil {
		return err
	}
	cur := filepath.Join(d, "current")
	if _, err := os.Stat(cur); os.IsNotExist(err) {
		if err := os.WriteFile(cur, []byte("home\n"), 0644); err != nil {
			return err
		}
	}
	return nil
}
