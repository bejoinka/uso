package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bejoinka/uso/internal/config"
)

// setupShare registers Claude Code under a temp root, creates the profile
// awell, and returns Claude Code's accounts directory.
func setupShare(t *testing.T) string {
	t.Helper()
	root := useTempConfig(t)
	accounts := filepath.Join(root, ".claude-accounts")
	tool := config.Tool{EnvVar: "CLAUDE_CONFIG_DIR", Symlink: filepath.Join(root, ".claude"), AccountsDir: accounts}
	if err := config.SaveTool("claude", tool); err != nil {
		t.Fatal(err)
	}
	if err := config.EnsureProfileDir("awell", map[string]config.Tool{"claude": tool}); err != nil {
		t.Fatal(err)
	}
	return accounts
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("read %s: %v", path, err)
	}
	return string(data)
}

func assertLink(t *testing.T, link, want string) {
	t.Helper()
	if got, err := os.Readlink(link); err != nil || got != want {
		t.Errorf("%s links to %q (%v), want %q", link, got, err, want)
	}
}

func TestShareLinksANewProfileToAnother(t *testing.T) {
	accounts := setupShare(t)
	awell, awell2 := filepath.Join(accounts, "awell"), filepath.Join(accounts, "awell2")
	writeFile(t, filepath.Join(awell, "settings.json"), `{"model":"opus"}`, 0600)
	writeFile(t, filepath.Join(awell, "agents", "reviewer.md"), "agent", 0644)
	writeFile(t, filepath.Join(awell, "projects", "-repo", "memory", "MEMORY.md"), "remember", 0644)
	writeFile(t, filepath.Join(awell, ".claude.json"), "awell's account", 0600)

	out, err := execute(t, "_add-profile", "awell2", "--share", "awell")
	if err != nil {
		t.Fatal(err)
	}

	// skills and commands don't exist in awell yet; they are created there
	// so that skills added later in either profile are shared too.
	for _, item := range []string{"settings.json", "agents", "projects", "skills", "commands"} {
		assertLink(t, filepath.Join(awell2, item), filepath.Join(awell, item))
	}
	if got := readFile(t, filepath.Join(awell2, "projects", "-repo", "memory", "MEMORY.md")); got != "remember" {
		t.Errorf("awell2 sees memory %q, want awell's", got)
	}
	if _, err := os.Lstat(filepath.Join(awell2, ".claude.json")); err == nil {
		t.Error(".claude.json was shared; awell2 must keep its own login")
	}
	if _, err := os.Lstat(filepath.Join(awell2, "CLAUDE.md")); err == nil {
		t.Error("CLAUDE.md exists in neither profile, so nothing should be linked")
	}
	if !strings.Contains(out, "shares") || !strings.Contains(out, "with awell") {
		t.Errorf("output does not say what is shared:\n%s", out)
	}
}

func TestShareMergesAnExistingProfile(t *testing.T) {
	accounts := setupShare(t)
	awell, awell2 := filepath.Join(accounts, "awell"), filepath.Join(accounts, "awell2")
	writeFile(t, filepath.Join(awell, "settings.json"), `{"model":"opus"}`, 0600)
	writeFile(t, filepath.Join(awell, "projects", "-repo", "memory", "MEMORY.md"), "awell's memory", 0644)
	writeFile(t, filepath.Join(awell, "projects", "-repo", "s1.jsonl"), "session 1", 0600)
	writeFile(t, filepath.Join(awell, "agents", "same.md"), "same", 0644)

	// awell2 was used on its own before sharing.
	writeFile(t, filepath.Join(awell2, "settings.json"), `{"model":"sonnet"}`, 0600) // differs
	writeFile(t, filepath.Join(awell2, "projects", "-repo", "s2.jsonl"), "session 2", 0600)
	writeFile(t, filepath.Join(awell2, "projects", "-other", "memory", "MEMORY.md"), "other memory", 0644)
	writeFile(t, filepath.Join(awell2, "agents", "same.md"), "same", 0644) // identical
	writeFile(t, filepath.Join(awell2, ".claude.json"), "awell2's account", 0600)
	// A link made by hand, as on a machine that shared one repo's memory.
	if err := os.Symlink(filepath.Join(awell, "projects", "-repo", "memory"), filepath.Join(awell2, "projects", "-repo", "memory")); err != nil {
		t.Fatal(err)
	}

	out, err := execute(t, "_add-profile", "awell2", "--share", "awell")
	if err != nil {
		t.Fatal(err)
	}

	// Both profiles now see one merged set.
	for path, want := range map[string]string{
		filepath.Join("projects", "-repo", "s1.jsonl"):             "session 1",
		filepath.Join("projects", "-repo", "s2.jsonl"):             "session 2",
		filepath.Join("projects", "-repo", "memory", "MEMORY.md"):  "awell's memory",
		filepath.Join("projects", "-other", "memory", "MEMORY.md"): "other memory",
		"settings.json": `{"model":"opus"}`,
	} {
		for _, profile := range []string{awell, awell2} {
			if got := readFile(t, filepath.Join(profile, path)); got != want {
				t.Errorf("%s = %q, want %q", filepath.Join(profile, path), got, want)
			}
		}
	}

	// awell's settings win; awell2's are kept in a backup, and the output says where.
	backups, _ := filepath.Glob(filepath.Join(awell2, "uso-backup-*", "settings.json"))
	if len(backups) != 1 {
		t.Fatalf("want awell2's settings.json in one backup, found %v", backups)
	}
	if got := readFile(t, backups[0]); got != `{"model":"sonnet"}` {
		t.Errorf("backup = %q, want awell2's settings", got)
	}
	if !strings.Contains(out, backups[0]) {
		t.Errorf("output does not mention the backup %s:\n%s", backups[0], out)
	}
	// Identical files and the hand-made link are not conflicts.
	if extra, _ := filepath.Glob(filepath.Join(awell2, "uso-backup-*", "*", "*")); len(extra) != 0 {
		t.Errorf("unexpected backups: %v", extra)
	}
	// The login stays put.
	if got := readFile(t, filepath.Join(awell2, ".claude.json")); got != "awell2's account" {
		t.Errorf("awell2's .claude.json = %q", got)
	}
}

func TestShareIsSafeToRunAgain(t *testing.T) {
	accounts := setupShare(t)
	writeFile(t, filepath.Join(accounts, "awell", "settings.json"), "{}", 0600)

	for i := 0; i < 2; i++ {
		if _, err := execute(t, "_add-profile", "awell2", "--share", "awell"); err != nil {
			t.Fatalf("run %d: %v", i+1, err)
		}
	}

	assertLink(t, filepath.Join(accounts, "awell2", "settings.json"), filepath.Join(accounts, "awell", "settings.json"))
	if backups, _ := filepath.Glob(filepath.Join(accounts, "awell2", "uso-backup-*")); len(backups) != 0 {
		t.Errorf("a second run made backups: %v", backups)
	}
}

func TestShareWithAProfileThatSharesLinksToTheOriginal(t *testing.T) {
	accounts := setupShare(t)
	if _, err := execute(t, "_add-profile", "awell2", "--share", "awell"); err != nil {
		t.Fatal(err)
	}

	if _, err := execute(t, "_add-profile", "awell3", "--share", "awell2"); err != nil {
		t.Fatal(err)
	}

	assertLink(t, filepath.Join(accounts, "awell3", "projects"), filepath.Join(accounts, "awell", "projects"))
}

func TestShareRejectsAMissingOrSameProfile(t *testing.T) {
	setupShare(t)

	if _, err := execute(t, "_add-profile", "awell", "--share", "awell"); err == nil {
		t.Error("a profile sharing with itself should fail")
	}
	if _, err := execute(t, "_add-profile", "new", "--share", "nope"); err == nil || !strings.Contains(err.Error(), "nope") {
		t.Errorf("sharing with a missing profile: err = %v", err)
	}
	if config.ProfileExists("new") {
		t.Error("the profile was created although sharing failed")
	}
}

func TestShareKeepsPluginsWholeRatherThanMixingThem(t *testing.T) {
	accounts := setupShare(t)
	awell, awell2 := filepath.Join(accounts, "awell"), filepath.Join(accounts, "awell2")
	writeFile(t, filepath.Join(awell, "plugins", "installed_plugins.json"), "awell's installs", 0644)
	writeFile(t, filepath.Join(awell2, "plugins", "installed_plugins.json"), "awell2's installs", 0644)
	writeFile(t, filepath.Join(awell2, "plugins", "cache", "only-in-awell2", "plugin.json"), "{}", 0644)

	if _, err := execute(t, "_add-profile", "awell2", "--share", "awell"); err != nil {
		t.Fatal(err)
	}

	assertLink(t, filepath.Join(awell2, "plugins"), filepath.Join(awell, "plugins"))
	if _, err := os.Lstat(filepath.Join(awell, "plugins", "cache", "only-in-awell2")); err == nil {
		t.Error("awell2's plugin files were mixed into awell's plugins")
	}
	backups, _ := filepath.Glob(filepath.Join(awell2, "uso-backup-*", "plugins", "cache", "only-in-awell2", "plugin.json"))
	if len(backups) != 1 {
		t.Errorf("awell2's whole plugins directory should be in the backup, found %v", backups)
	}
}

func TestShareTreatsIdenticalLinksAsTheSame(t *testing.T) {
	accounts := setupShare(t)
	// The same relative link in both profiles, e.g. to a skill kept outside
	// them; it doesn't need to resolve to count as identical.
	for _, profile := range []string{"awell", "awell2"} {
		dir := filepath.Join(accounts, profile, "skills")
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("../../../.agents/skills/checkly", filepath.Join(dir, "checkly")); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := execute(t, "_add-profile", "awell2", "--share", "awell"); err != nil {
		t.Fatal(err)
	}

	if backups, _ := filepath.Glob(filepath.Join(accounts, "awell2", "uso-backup-*")); len(backups) != 0 {
		t.Errorf("identical links were backed up as conflicts: %v", backups)
	}
}
