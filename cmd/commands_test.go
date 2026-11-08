package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bejoinka/uso/internal/config"
	"github.com/bejoinka/uso/internal/hook"
)

// useTempConfig points uso at an empty temp config and returns the temp root.
func useTempConfig(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("USO_DIR", filepath.Join(root, "uso"))
	t.Setenv("USO_PROFILE", "")
	if err := config.EnsureInit(); err != nil {
		t.Fatal(err)
	}
	return root
}

// claudeTool returns the built-in Claude Code definition with its paths under root.
func claudeTool(root string) config.KnownTool {
	return config.KnownTool{
		ID:          "claude",
		EnvVar:      "CLAUDE_CONFIG_DIR",
		Symlink:     filepath.Join(root, ".claude"),
		AccountsDir: filepath.Join(root, ".claude-accounts"),
		Label:       "Claude Code",
		StateFile:   filepath.Join(root, ".claude.json"),
	}
}

func writeFile(t *testing.T, path, content string, perm os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), perm); err != nil {
		t.Fatal(err)
	}
}

func listOutput(t *testing.T) string {
	t.Helper()
	var out bytes.Buffer
	listCmd.SetOut(&out)
	t.Cleanup(func() { listCmd.SetOut(nil) })
	if err := runList(listCmd, nil); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestInitWithACustomDefaultProfileName(t *testing.T) {
	root := useTempConfig(t)
	claude := claudeTool(root)
	if err := os.MkdirAll(claude.Symlink, 0755); err != nil { // existing config to migrate
		t.Fatal(err)
	}

	if err := applyInit(io.Discard, []config.KnownTool{claude}, "personal"); err != nil {
		t.Fatal(err)
	}

	if got, want := listOutput(t), "* personal\n"; got != want {
		t.Errorf("uso list = %q, want %q", got, want)
	}
	if !config.ProfileExists("personal") {
		t.Error("the default profile 'personal' does not exist after init")
	}

	out, err := hook.EvalSwitch("personal")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "export CLAUDE_CONFIG_DIR=") {
		t.Errorf("uso personal did not switch; output:\n%s", out)
	}

	out, err = hook.EvalSwitch("home")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "unknown profile 'home'") {
		t.Errorf("uso home should fail when no home profile exists; output:\n%s", out)
	}
}

func TestInitWithNoToolsStillCreatesTheDefaultProfile(t *testing.T) {
	useTempConfig(t)

	if err := applyInit(io.Discard, nil, "personal"); err != nil {
		t.Fatal(err)
	}

	if !config.ProfileExists("personal") {
		t.Error("the default profile 'personal' does not exist after init")
	}
}

func TestInitCopiesClaudeStateIntoTheDefaultProfile(t *testing.T) {
	root := useTempConfig(t)
	claude := claudeTool(root)
	writeFile(t, filepath.Join(claude.Symlink, "settings.json"), "{}", 0644)
	writeFile(t, claude.StateFile, `{"mcpServers":{"linear":{}}}`, 0600)

	if err := applyInit(io.Discard, []config.KnownTool{claude}, "home"); err != nil {
		t.Fatal(err)
	}

	copied := filepath.Join(claude.AccountsDir, "home", ".claude.json")
	data, err := os.ReadFile(copied)
	if err != nil {
		t.Fatalf("~/.claude.json was not copied into the default profile: %v", err)
	}
	if string(data) != `{"mcpServers":{"linear":{}}}` {
		t.Errorf("copied ~/.claude.json = %q", data)
	}
	if info, err := os.Stat(copied); err != nil || info.Mode().Perm() != 0600 {
		t.Errorf("copied ~/.claude.json should keep mode 0600; got %v, %v", info.Mode().Perm(), err)
	}
	if _, err := os.Stat(claude.StateFile); err != nil {
		t.Errorf("the original ~/.claude.json should stay in place: %v", err)
	}
}

func TestInitKeepsAStateFileTheProfileAlreadyHas(t *testing.T) {
	root := useTempConfig(t)
	claude := claudeTool(root)
	writeFile(t, claude.StateFile, "top-level", 0600)
	inProfile := filepath.Join(claude.AccountsDir, "home", ".claude.json")
	writeFile(t, inProfile, "the profile's own", 0600)

	if err := applyInit(io.Discard, []config.KnownTool{claude}, "home"); err != nil {
		t.Fatal(err)
	}

	if data, _ := os.ReadFile(inProfile); string(data) != "the profile's own" {
		t.Errorf("init overwrote the profile's .claude.json with %q", data)
	}
}

func TestAddToolCreatesADirectoryForEveryProfile(t *testing.T) {
	root := useTempConfig(t)
	for _, p := range []string{"personal", "work"} {
		if err := config.EnsureProfileDir(p, nil); err != nil {
			t.Fatal(err)
		}
	}
	accounts := filepath.Join(root, ".gemini-accounts")

	addToolCmd.SetOut(io.Discard)
	t.Cleanup(func() { addToolCmd.SetOut(nil) })
	err := runAddTool(addToolCmd, []string{"gemini", "GEMINI_CONFIG_DIR", filepath.Join(root, ".gemini"), accounts})
	if err != nil {
		t.Fatal(err)
	}

	for _, p := range []string{"personal", "work"} {
		if info, err := os.Stat(filepath.Join(accounts, p)); err != nil || !info.IsDir() {
			t.Errorf("expected a gemini directory for profile %q", p)
		}
	}
	if _, err := os.Stat(filepath.Join(accounts, "home")); err == nil {
		t.Error("add-tool created a 'home' directory although no home profile exists")
	}
}
