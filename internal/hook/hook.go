package hook

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bejoinka/uso/internal/config"
)

// ShellEscape escapes a string for safe use in single-quoted shell context.
// Replaces ' with '\'' (end quote, escaped quote, start quote).
func ShellEscape(s string) string {
	return strings.ReplaceAll(s, "'", `'\''`)
}

// Zsh outputs the shell code for eval "$(uso hook zsh)".
func Zsh() string {
	return shellHook("zsh", binPath())
}

// Bash outputs the shell code for eval "$(uso hook bash)".
func Bash() string {
	return shellHook("bash", binPath())
}

// binPath returns the path of the running binary, with symlinks resolved
// (e.g., Homebrew bin -> cellar).
func binPath() string {
	bin, _ := os.Executable()
	if resolved, err := filepath.EvalSymlinks(bin); err == nil {
		bin = resolved
	}
	return bin
}

func shellHook(shell, bin string) string {
	var b strings.Builder

	// Main function: runs the binary with "eval" subcommand, evals the output
	// The ';' before '}' is required by bash.
	fmt.Fprintf(&b, `uso() { eval "$(%s eval "$@")"; }`, bin)
	b.WriteString("\n")

	// Completions
	if shell == "zsh" {
		fmt.Fprintf(&b, `
_uso_completions() {
  local -a subcmds profiles
  subcmds=(list status init add-tool add-profile set show help)
  profiles=($(%s completions profiles 2>/dev/null))
  if (( CURRENT == 2 )); then
    compadd -a subcmds profiles
  elif (( CURRENT == 3 )) && [[ "${words[2]}" == (set|show) ]]; then
    compadd -a profiles
  fi
}
compdef _uso_completions uso 2>/dev/null
`, bin)
	} else {
		fmt.Fprintf(&b, `
_uso_completions_bash() {
  local cur="${COMP_WORDS[COMP_CWORD]}"
  local subcmds="list status init add-tool add-profile set show help"
  local profiles="$(%s completions profiles 2>/dev/null)"
  if [[ $COMP_CWORD -eq 1 ]]; then
    COMPREPLY=($(compgen -W "$subcmds $profiles" -- "$cur"))
  elif [[ $COMP_CWORD -eq 2 ]] && [[ "${COMP_WORDS[1]}" == "set" || "${COMP_WORDS[1]}" == "show" ]]; then
    COMPREPLY=($(compgen -W "$profiles" -- "$cur"))
  fi
}
complete -F _uso_completions_bash uso
`, bin)
	}

	// Restore last profile on shell startup
	fmt.Fprintf(&b, "\n# restore last profile\neval \"$(%s eval --restore)\"\n", bin)

	return b.String()
}

// shellProfile returns the profile the calling shell is on. That is
// $USO_PROFILE, which the hook exports, not the last profile used in any
// terminal; it falls back to the latter if $USO_PROFILE is unset.
func shellProfile() string {
	if p := os.Getenv("USO_PROFILE"); config.ValidName(p) {
		return p
	}
	return config.CurrentProfile()
}

// writeUnsets writes an unset command for each of a profile's env vars.
func writeUnsets(b *strings.Builder, profile string) {
	env, _ := config.LoadProfileEnv(profile)
	for _, kv := range env {
		fmt.Fprintf(b, "unset %s 2>/dev/null\n", kv[0])
	}
}

// writeTerminalVisuals writes printf commands that set the iTerm2 tab color
// from USO_COLOR and the terminal title from USO_TITLE. Values go in as
// single-quoted arguments, never into the format string, so the shell and
// printf pass them through verbatim.
func writeTerminalVisuals(b *strings.Builder, env [][2]string) {
	for _, kv := range env {
		switch kv[0] {
		case "USO_COLOR":
			parts := strings.SplitN(kv[1], ",", 3)
			if len(parts) != 3 {
				continue
			}
			for i, channel := range []string{"red", "green", "blue"} {
				fmt.Fprintf(b, `printf '\033]6;1;bg;%s;brightness;%%s\a' '%s'`+"\n",
					channel, ShellEscape(strings.TrimSpace(parts[i])))
			}
		case "USO_TITLE":
			fmt.Fprintf(b, `printf '\033]0;%%s\a' '%s'`+"\n", ShellEscape(kv[1]))
		}
	}
}

// EvalSwitch outputs shell commands to switch to a profile.
func EvalSwitch(profile string) (string, error) {
	tools, err := config.LoadTools()
	if err != nil {
		return "", err
	}
	if len(tools) == 0 {
		return `echo "uso: no tools configured. Run 'uso init'." >&2`, nil
	}

	// Validate profile
	if !config.ProfileExists(profile) {
		profiles, _ := config.ListProfiles()
		return fmt.Sprintf(`echo "uso: unknown profile '%s'" >&2; echo "  profiles: %s" >&2`,
			profile, strings.Join(profiles, " ")), nil
	}

	var b strings.Builder

	// Clear the env vars of the profile this shell is on
	writeUnsets(&b, shellProfile())

	// Switch tool symlinks
	for name, t := range tools {
		profileDir := filepath.Join(t.AccountsDir, profile)
		info, err := os.Stat(profileDir)
		if err != nil || !info.IsDir() {
			fmt.Fprintf(&b, "echo '  %s: no dir at %s — skipping' >&2\n", name, profileDir)
			continue
		}

		// Check symlink safety
		symlinkInfo, err := os.Lstat(t.Symlink)
		if err == nil && symlinkInfo.IsDir() && symlinkInfo.Mode()&os.ModeSymlink == 0 {
			fmt.Fprintf(&b, "echo '  %s: %s is a real directory — skipping' >&2\n", name, t.Symlink)
			continue
		}

		// Do the symlink swap (binary does this directly)
		os.Remove(t.Symlink)
		if err := os.Symlink(profileDir, t.Symlink); err != nil {
			fmt.Fprintf(&b, "echo '  %s: symlink failed: %s' >&2\n", name, err)
			continue
		}

		fmt.Fprintf(&b, "export %s='%s'\n", t.EnvVar, ShellEscape(profileDir))
		fmt.Fprintf(&b, "echo '-> %s: %s  [%s]'\n", ShellEscape(name), ShellEscape(profile), ShellEscape(profileDir))
	}

	// Export profile env vars
	env, _ := config.LoadProfileEnv(profile)
	for _, kv := range env {
		fmt.Fprintf(&b, "export %s='%s'\n", kv[0], ShellEscape(kv[1]))
	}

	fmt.Fprintf(&b, "export USO_PROFILE='%s'\n", ShellEscape(profile))

	writeTerminalVisuals(&b, env)

	// Save current profile
	config.SetCurrentProfile(profile)

	// Post-switch hook
	hookPath := filepath.Join(config.Dir(), "post-switch")
	if info, err := os.Stat(hookPath); err == nil && info.Mode()&0111 != 0 {
		fmt.Fprintf(&b, "'%s' '%s'\n", ShellEscape(hookPath), ShellEscape(profile))
	}

	return b.String(), nil
}

// EvalRestore outputs shell commands to restore the last profile (silent).
func EvalRestore() (string, error) {
	tools, err := config.LoadTools()
	if err != nil || len(tools) == 0 {
		return "", nil
	}

	current := config.CurrentProfile()
	var b strings.Builder

	// A new shell can inherit another profile's env vars, e.g. a tmux pane
	// or an editor terminal started from a shell on that profile.
	if inherited := os.Getenv("USO_PROFILE"); config.ValidName(inherited) && inherited != current {
		writeUnsets(&b, inherited)
	}

	fmt.Fprintf(&b, "export USO_PROFILE='%s'\n", ShellEscape(current))

	for _, t := range tools {
		profileDir := filepath.Join(t.AccountsDir, current)
		if info, err := os.Stat(profileDir); err == nil && info.IsDir() {
			// Only update symlink if it's already a symlink
			if linfo, err := os.Lstat(t.Symlink); err == nil && linfo.Mode()&os.ModeSymlink != 0 {
				os.Remove(t.Symlink)
				os.Symlink(profileDir, t.Symlink)
			}
			fmt.Fprintf(&b, "export %s='%s'\n", t.EnvVar, ShellEscape(profileDir))
		}
	}

	// Restore profile env vars
	env, _ := config.LoadProfileEnv(current)
	for _, kv := range env {
		fmt.Fprintf(&b, "export %s='%s'\n", kv[0], ShellEscape(kv[1]))
	}

	writeTerminalVisuals(&b, env)

	return b.String(), nil
}
