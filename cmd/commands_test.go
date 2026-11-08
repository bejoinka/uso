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
	claude := config.KnownTool{
		ID:          "claude",
		EnvVar:      "CLAUDE_CONFIG_DIR",
		Symlink:     filepath.Join(root, ".claude"),
		AccountsDir: filepath.Join(root, ".claude-accounts"),
		Label:       "Claude Code",
	}
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
