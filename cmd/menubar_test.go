package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMenubarListsSessionsNewestFirst(t *testing.T) {
	accounts := setupSession(t) // awell has "systems", which ran in /work/my repo
	newer := "aaaaaaaa-0000-0000-0000-000000000000"
	transcript := filepath.Join(accounts, "awell2", "projects", "-other", newer+".jsonl")
	writeFile(t, transcript, `{"type":"user","cwd":"/other"}`+"\n"+`{"type":"ai-title","aiTitle":"Generated | title"}`+"\n", 0600)
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(filepath.Join(accounts, "awell", "projects", "-work-my-repo", sessionID+".jsonl"), old, old); err != nil {
		t.Fatal(err)
	}

	out, err := execute(t, "_swiftbar", "/plugins/uso sessions.sh")
	if err != nil {
		t.Fatal(err)
	}

	rows := strings.Split(strings.TrimSpace(out), "\n")[3:] // after the icon, separator and header
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2:\n%s", len(rows), out)
	}
	// A session never renamed shows the title Claude Code generated, without
	// the | that SwiftBar would read as the start of the row's options.
	for i, want := range [][]string{
		{"awell2", "Generated ¦ title", "/other", "1m", "param2=awell2 param3=" + newer},
		{"awell ", "systems", "/work/my repo", "2h", "param2=awell param3=" + sessionID},
	} {
		for _, w := range want {
			if !strings.Contains(rows[i], w) {
				t.Errorf("row %d = %q, want it to contain %q", i+1, rows[i], w)
			}
		}
		if !strings.Contains(rows[i], `bash="/plugins/uso sessions.sh" param1=copy`) {
			t.Errorf("row %d = %q doesn't run the plugin to copy", i+1, rows[i])
		}
	}
}

func TestMenubarClickCopiesTheResumeCommand(t *testing.T) {
	setupSession(t)
	var copied, title string
	realCopy, realNotify := copyToClipboard, notify
	t.Cleanup(func() { copyToClipboard, notify = realCopy, realNotify })
	copyToClipboard = func(text string) error { copied = text; return nil }
	notify = func(t, _ string) { title = t }
	t.Setenv("USO_CLAUDE_FLAGS", "--dangerously-skip-permissions")

	if _, err := execute(t, "_swiftbar", "/plugins/uso-sessions.sh", "copy", "awell", sessionID); err != nil {
		t.Fatal(err)
	}

	want := "uso awell && cd '/work/my repo' && claude --dangerously-skip-permissions --resume " + sessionID
	if copied != want {
		t.Errorf("copied %q, want %q", copied, want)
	}
	if title != "Copied: systems" {
		t.Errorf("notification title %q, want %q", title, "Copied: systems")
	}
}

func TestMenubarPrintsAPluginThatRunsUsoByFullPath(t *testing.T) {
	useTempConfig(t)

	out, err := execute(t, "_menubar", "--claude-flags", "--dangerously-skip-permissions")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"<swiftbar.refreshOnOpen>true</swiftbar.refreshOnOpen>",
		"export USO_CLAUDE_FLAGS='--dangerously-skip-permissions'",
		"exec '/",
		`' _swiftbar "$0" "$@"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("plugin doesn't contain %q:\n%s", want, out)
		}
	}

	out, err = execute(t, "eval", "menubar")
	if err != nil || !strings.Contains(out, " _menubar") {
		t.Errorf("eval printed %q (%v), want it to run _menubar", out, err)
	}
}

func TestFit(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"short", "short"},
		{"a | b", "a ¦ b"},
		{"exactly10!", "exactly10!"},
		{"one too long", "one too l…"},
	} {
		if got := fit(c.in, 10); got != c.want {
			t.Errorf("fit(%q, 10) = %q, want %q", c.in, got, c.want)
		}
	}
}
