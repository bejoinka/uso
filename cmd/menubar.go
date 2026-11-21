package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/bejoinka/uso/internal/config"
	"github.com/bejoinka/uso/internal/hook"
	"github.com/spf13/cobra"
)

// The menu-bar list of recent sessions is a SwiftBar (https://swiftbar.app)
// plugin. `uso menubar` prints the plugin: a short script that runs
// `uso _swiftbar`, which prints the menu or, when a row is clicked, copies
// that session's resume command.

const menubarRows = 15

// menubarPlugin is the plugin script. Its file name has no interval and it
// sets refreshOnOpen, so SwiftBar runs it only when the menu opens. With
// runInBash false, SwiftBar runs it directly rather than through a shell.
const menubarPlugin = `#!/bin/sh
# <swiftbar.title>Recent Claude Code sessions</swiftbar.title>
# <swiftbar.refreshOnOpen>true</swiftbar.refreshOnOpen>
# <swiftbar.runInBash>false</swiftbar.runInBash>
# <swiftbar.hideRunInTerminal>true</swiftbar.hideRunInTerminal>
# <swiftbar.hideDisablePlugin>true</swiftbar.hideDisablePlugin>
#
# Made by 'uso menubar'. Lists recent Claude Code sessions across uso profiles;
# clicking one copies the command that resumes it.

# Options to add to the claude command that a click copies
export USO_CLAUDE_FLAGS='%s'
exec '%s' _swiftbar "$0" "$@"
`

func runMenubar(cmd *cobra.Command, args []string) error {
	flags, _ := cmd.Flags().GetString("claude-flags")
	fmt.Fprintf(cmd.OutOrStdout(), menubarPlugin, hook.ShellEscape(flags), hook.ShellEscape(hook.BinPath()))
	return nil
}

// runSwiftbar is what the plugin runs: `_swiftbar <plugin>` prints the menu,
// and `_swiftbar <plugin> copy <profile> <id>` copies a session's resume command.
func runSwiftbar(cmd *cobra.Command, args []string) error {
	tools, err := config.LoadTools()
	if err != nil {
		return err
	}
	claude, ok := tools["claude"]
	if !ok {
		return errors.New("Claude Code isn't registered with uso; run uso init")
	}
	if len(args) == 4 && args[1] == "copy" {
		return copyResume(claude.AccountsDir, args[2], args[3])
	}

	sessions, err := config.ListSessions(claude.AccountsDir, "")
	if err != nil {
		return err
	}
	slices.SortFunc(sessions, func(a, b config.Session) int { return b.Modified.Compare(a.Modified) })

	out := cmd.OutOrStdout()
	mono := "font=Menlo size=12 trim=false"
	fmt.Fprintln(out, "| sfimage=terminal")
	fmt.Fprintln(out, "---")
	fmt.Fprintf(out, "%-8s %-34s %-30s %6s | %s color=gray\n", "PROFILE", "SESSION", "PATH", "LAST", mono)
	now := time.Now()
	for _, s := range sessions[:min(menubarRows, len(sessions))] {
		fmt.Fprintf(out, "%-8s %-34s %-30s %6s | %s bash=%s param1=copy param2=%s param3=%s terminal=false refresh=false\n",
			fit(s.Profile, 8), fit(s.Label(claude.AccountsDir), 34), fit(tildePath(s.WorkingDir(claude.AccountsDir)), 30),
			ago(now, s.Modified), mono, swiftbarQuote(args[0]), s.Profile, s.ID)
	}
	return nil
}

// Replaced in tests, so that they don't touch the clipboard or notify.
var (
	copyToClipboard = func(text string) error {
		pbcopy := exec.Command("pbcopy")
		pbcopy.Stdin = strings.NewReader(text)
		return pbcopy.Run()
	}
	notify = func(title, text string) {
		script := fmt.Sprintf("display notification %s with title %s", appleScriptString(text), appleScriptString(title))
		exec.Command("osascript", "-e", script).Run() // a missing notification isn't worth failing over
	}
)

func copyResume(accountsDir, profile, id string) error {
	found, err := config.FindSessions(accountsDir, id, profile)
	if err != nil {
		return err
	}
	if len(found) != 1 {
		return fmt.Errorf("no session %s in profile '%s'", id, profile)
	}
	s := found[0]
	line := resumeCommand(accountsDir, s, profile, os.Getenv("USO_CLAUDE_FLAGS"))
	if err := copyToClipboard(line); err != nil {
		return err
	}
	notify("Copied: "+s.Label(accountsDir), line)
	return nil
}

// resumeCommand returns the command that switches to profile, changes to the
// directory the session ran in, and resumes the session with claude.
func resumeCommand(accountsDir string, s config.Session, profile, claudeFlags string) string {
	line := "uso " + profile
	if dir := s.WorkingDir(accountsDir); dir != "" {
		line += fmt.Sprintf(" && cd '%s'", hook.ShellEscape(dir))
	}
	claude := strings.TrimSpace("claude " + strings.TrimSpace(claudeFlags))
	return line + " && " + claude + " --resume " + s.ID
}

// fit shortens text to width characters, and replaces the | that SwiftBar
// reads as the start of a line's options.
func fit(text string, width int) string {
	r := []rune(strings.ReplaceAll(text, "|", "¦"))
	if len(r) <= width {
		return string(r)
	}
	return string(r[:width-1]) + "…"
}

func tildePath(dir string) string {
	home, _ := os.UserHomeDir()
	switch {
	case dir == "":
		return "?"
	case home != "" && dir == home:
		return "~"
	case home != "" && strings.HasPrefix(dir, home+"/"):
		return "~" + dir[len(home):]
	}
	return dir
}

func ago(now, t time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm", max(1, int(d.Minutes())))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
	return t.Format("Jan 2")
}

func swiftbarQuote(s string) string {
	if !strings.ContainsAny(s, " \"'") {
		return s
	}
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

func appleScriptString(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}
