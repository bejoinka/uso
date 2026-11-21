package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bejoinka/uso/internal/config"
)

const sessionID = "313678d1-50b7-4f10-836b-d50c56e47f71"

// setupSession registers Claude Code with the profiles awell and awell2, puts
// a session titled "systems" that ran in /work/my repo into awell, and
// returns Claude Code's accounts directory.
func setupSession(t *testing.T) string {
	t.Helper()
	accounts := setupShare(t)
	tool := config.Tool{AccountsDir: accounts}
	if err := config.EnsureProfileDir("awell2", map[string]config.Tool{"claude": tool}); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(accounts, "awell", "projects", "-work-my-repo")
	writeFile(t, filepath.Join(project, sessionID+".jsonl"),
		`{"type":"permission-mode"}`+"\n"+`{"type":"user","cwd":"/work/my repo"}`+"\n", 0600)
	writeFile(t, filepath.Join(project, sessionID, "custom-title.json"), `{"customTitle":"systems"}`, 0600)
	writeFile(t, filepath.Join(project, sessionID, "subagents", "agent-1.jsonl"), "subagent", 0600)
	writeFile(t, filepath.Join(accounts, "awell", "file-history", sessionID, "abc@v1"), "checkpoint", 0600)
	return accounts
}

func TestCopySessionByTitle(t *testing.T) {
	accounts := setupSession(t)

	out, err := execute(t, "_copy-session", "systems", "awell2")
	if err != nil {
		t.Fatal(err)
	}

	project := filepath.Join(accounts, "awell2", "projects", "-work-my-repo")
	for path, want := range map[string]string{
		filepath.Join(project, sessionID+".jsonl"):                                        `{"type":"permission-mode"}` + "\n" + `{"type":"user","cwd":"/work/my repo"}` + "\n",
		filepath.Join(project, sessionID, "custom-title.json"):                            `{"customTitle":"systems"}`,
		filepath.Join(project, sessionID, "subagents", "agent-1.jsonl"):                   "subagent",
		filepath.Join(accounts, "awell2", "file-history", sessionID, "abc@v1"):            "checkpoint",
		filepath.Join(accounts, "awell", "projects", "-work-my-repo", sessionID+".jsonl"): `{"type":"permission-mode"}` + "\n" + `{"type":"user","cwd":"/work/my repo"}` + "\n",
	} {
		if got := readFile(t, path); got != want {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}
	want := "uso awell2 && cd '/work/my repo' && claude --resume " + sessionID
	if !strings.Contains(out, want) {
		t.Errorf("output %q doesn't give the resume command %q", out, want)
	}
}

func TestCopySessionByIDPrefix(t *testing.T) {
	accounts := setupSession(t)

	if _, err := execute(t, "_copy-session", "313678d1", "awell2"); err != nil {
		t.Fatal(err)
	}
	if !exists(filepath.Join(accounts, "awell2", "projects", "-work-my-repo", sessionID+".jsonl")) {
		t.Error("the session wasn't copied")
	}
}

func TestCopySessionRefusesToOverwrite(t *testing.T) {
	accounts := setupSession(t)
	existing := filepath.Join(accounts, "awell2", "projects", "-work-my-repo", sessionID+".jsonl")
	writeFile(t, existing, "awell2's own copy", 0600)

	// awell2's copy isn't a source, so the only source is awell's.
	_, err := execute(t, "_copy-session", "systems", "awell2", "--from", "awell")
	if err == nil || !strings.Contains(err.Error(), "already has") {
		t.Errorf("err = %v, want one saying awell2 already has the session", err)
	}
	if got := readFile(t, existing); got != "awell2's own copy" {
		t.Errorf("awell2's copy was overwritten with %q", got)
	}
}

func TestCopySessionNamesEveryMatchWhenAmbiguous(t *testing.T) {
	accounts := setupSession(t)
	other := "313678d1-0000-0000-0000-000000000000"
	writeFile(t, filepath.Join(accounts, "awell", "projects", "-elsewhere", other+".jsonl"), "{}\n", 0600)

	_, err := execute(t, "_copy-session", "313678d1", "awell2")
	if err == nil {
		t.Fatal("copied one of two matching sessions")
	}
	for _, id := range []string{sessionID, other} {
		if !strings.Contains(err.Error(), id) {
			t.Errorf("error %q doesn't list %s", err, id)
		}
	}
}

func TestCopySessionThatIsOnlyInTheTarget(t *testing.T) {
	setupSession(t)

	_, err := execute(t, "_copy-session", "systems", "awell")
	if err == nil || !strings.Contains(err.Error(), "already in 'awell'") {
		t.Errorf("err = %v, want one saying the session is already in awell", err)
	}
}

func TestCopySessionThatDoesNotExist(t *testing.T) {
	setupSession(t)

	_, err := execute(t, "_copy-session", "nope", "awell2")
	if err == nil || !strings.Contains(err.Error(), "no session") {
		t.Errorf("err = %v, want one saying there's no such session", err)
	}
}

func TestCopySessionThroughTheShellFunction(t *testing.T) {
	useTempConfig(t)

	out, err := execute(t, "eval", "copy-session", "systems", "awell2", "--from", "awell")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "_copy-session 'systems' 'awell2' '--from' 'awell'") {
		t.Errorf("eval printed %q, want it to run _copy-session with the arguments quoted", out)
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
