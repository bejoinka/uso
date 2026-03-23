package hook

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bejoinka/uso/internal/config"
)

// Zsh outputs the shell code for eval "$(uso hook zsh)".
func Zsh() string {
	return shellHook("zsh")
}

// Bash outputs the shell code for eval "$(uso hook bash)".
func Bash() string {
	return shellHook("bash")
}

func shellHook(shell string) string {
	bin, _ := os.Executable()
	// Resolve symlinks (e.g., Homebrew cellar -> bin)
	if resolved, err := filepath.EvalSymlinks(bin); err == nil {
		bin = resolved
	}

	var b strings.Builder

	// Main function: runs the binary with "eval" subcommand, evals the output
	fmt.Fprintf(&b, `uso() { eval "$(%s eval "$@")" }`, bin)
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
	if profile != "home" && !config.ProfileExists(profile) {
		profiles, _ := config.ListProfiles()
		all := append([]string{"home"}, profiles...)
		return fmt.Sprintf(`echo "uso: unknown profile '%s'" >&2; echo "  profiles: %s" >&2`,
			profile, strings.Join(all, " ")), nil
	}

	var b strings.Builder

	// Clear previous profile env vars
	prev := config.CurrentProfile()
	if prevEnv, _ := config.LoadProfileEnv(prev); len(prevEnv) > 0 {
		for _, kv := range prevEnv {
			fmt.Fprintf(&b, "unset %s 2>/dev/null\n", kv[0])
		}
	}

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

		fmt.Fprintf(&b, "export %s='%s'\n", t.EnvVar, profileDir)
		fmt.Fprintf(&b, "echo '-> %s: %s  [%s]'\n", name, profile, profileDir)
	}

	// Export profile env vars
	env, _ := config.LoadProfileEnv(profile)
	for _, kv := range env {
		fmt.Fprintf(&b, "export %s='%s'\n", kv[0], kv[1])
	}

	fmt.Fprintf(&b, "export USO_PROFILE='%s'\n", profile)

	// Built-in: iTerm tab color
	for _, kv := range env {
		if kv[0] == "USO_COLOR" {
			parts := strings.SplitN(kv[1], ",", 3)
			if len(parts) == 3 {
				fmt.Fprintf(&b, `printf "\033]6;1;bg;red;brightness;%s\a"`, strings.TrimSpace(parts[0]))
				b.WriteString("\n")
				fmt.Fprintf(&b, `printf "\033]6;1;bg;green;brightness;%s\a"`, strings.TrimSpace(parts[1]))
				b.WriteString("\n")
				fmt.Fprintf(&b, `printf "\033]6;1;bg;blue;brightness;%s\a"`, strings.TrimSpace(parts[2]))
				b.WriteString("\n")
			}
		}
		if kv[0] == "USO_TITLE" {
			fmt.Fprintf(&b, `printf "\033]0;%s\a"`, kv[1])
			b.WriteString("\n")
		}
	}

	// Save current profile
	config.SetCurrentProfile(profile)

	// Post-switch hook
	hookPath := filepath.Join(config.Dir(), "post-switch")
	if info, err := os.Stat(hookPath); err == nil && info.Mode()&0111 != 0 {
		fmt.Fprintf(&b, "'%s' '%s'\n", hookPath, profile)
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

	fmt.Fprintf(&b, "export USO_PROFILE='%s'\n", current)

	for _, t := range tools {
		profileDir := filepath.Join(t.AccountsDir, current)
		if info, err := os.Stat(profileDir); err == nil && info.IsDir() {
			// Only update symlink if it's already a symlink
			if linfo, err := os.Lstat(t.Symlink); err == nil && linfo.Mode()&os.ModeSymlink != 0 {
				os.Remove(t.Symlink)
				os.Symlink(profileDir, t.Symlink)
			}
			fmt.Fprintf(&b, "export %s='%s'\n", t.EnvVar, profileDir)
		}
	}

	// Restore profile env vars
	env, _ := config.LoadProfileEnv(current)
	for _, kv := range env {
		fmt.Fprintf(&b, "export %s='%s'\n", kv[0], kv[1])
	}

	// Apply visuals silently
	for _, kv := range env {
		if kv[0] == "USO_COLOR" {
			parts := strings.SplitN(kv[1], ",", 3)
			if len(parts) == 3 {
				fmt.Fprintf(&b, `printf "\033]6;1;bg;red;brightness;%s\a"`, strings.TrimSpace(parts[0]))
				b.WriteString("\n")
				fmt.Fprintf(&b, `printf "\033]6;1;bg;green;brightness;%s\a"`, strings.TrimSpace(parts[1]))
				b.WriteString("\n")
				fmt.Fprintf(&b, `printf "\033]6;1;bg;blue;brightness;%s\a"`, strings.TrimSpace(parts[2]))
				b.WriteString("\n")
			}
		}
		if kv[0] == "USO_TITLE" {
			fmt.Fprintf(&b, `printf "\033]0;%s\a"`, kv[1])
			b.WriteString("\n")
		}
	}

	return b.String(), nil
}
