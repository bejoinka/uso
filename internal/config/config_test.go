package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// useTempConfig points uso at a temp config with one registered tool and
// returns that tool's accounts directory.
func useTempConfig(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("USO_DIR", filepath.Join(root, "uso"))
	accounts := filepath.Join(root, ".claude-accounts")
	if err := EnsureInit(); err != nil {
		t.Fatal(err)
	}
	if err := SaveTool("claude", Tool{
		EnvVar:      "CLAUDE_CONFIG_DIR",
		Symlink:     filepath.Join(root, ".claude"),
		AccountsDir: accounts,
	}); err != nil {
		t.Fatal(err)
	}
	return accounts
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
}

func TestListProfilesIncludesConfigFilesAndToolDirectories(t *testing.T) {
	accounts := useTempConfig(t)

	// home has a tool directory but no config file, like the default
	// profile on installs made before init created one.
	mkdir(t, filepath.Join(accounts, "home"))
	// work has a config file only.
	if err := SetProfileKey("work", "AWS_PROFILE", "prod"); err != nil {
		t.Fatal(err)
	}
	// personal has both.
	mkdir(t, filepath.Join(accounts, "personal"))
	if err := SetProfileKey("personal", "GH_TOKEN", "x"); err != nil {
		t.Fatal(err)
	}
	// Neither a stray file nor a directory with an invalid name is a profile.
	if err := os.WriteFile(filepath.Join(accounts, ".DS_Store"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	mkdir(t, filepath.Join(accounts, "Not A Profile"))

	got, err := ListProfiles()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"home", "personal", "work"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ListProfiles() = %v, want %v", got, want)
	}
}

func TestProfileExists(t *testing.T) {
	accounts := useTempConfig(t)
	mkdir(t, filepath.Join(accounts, "home"))
	if err := SetProfileKey("work", "AWS_PROFILE", "prod"); err != nil {
		t.Fatal(err)
	}

	tests := map[string]bool{
		"home":                     true,  // tool directory only
		"work":                     true,  // config file only
		"nope":                     false, // neither
		"../.claude-accounts/home": false, // not a valid profile name
	}
	for name, want := range tests {
		if got := ProfileExists(name); got != want {
			t.Errorf("ProfileExists(%q) = %v, want %v", name, got, want)
		}
	}
}
