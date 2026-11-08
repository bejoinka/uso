package hook

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bejoinka/uso/internal/config"
)

func TestShellHookDefinesAWorkingFunction(t *testing.T) {
	for _, shell := range []string{"bash", "zsh"} {
		t.Run(shell, func(t *testing.T) {
			sh, err := exec.LookPath(shell)
			if err != nil {
				t.Skipf("%s is not installed", shell)
			}

			// A stand-in for the uso binary: prints a command that echoes its arguments.
			bin := filepath.Join(t.TempDir(), "uso")
			if err := os.WriteFile(bin, []byte("#!/bin/sh\necho \"echo 'got: $*'\"\n"), 0755); err != nil {
				t.Fatal(err)
			}

			script := shellHook(shell, bin) + "\nuso work\n"
			args := []string{"-c", script}
			if shell == "zsh" {
				args = append([]string{"-f"}, args...) // skip the developer's rc files
			}
			out, err := exec.Command(sh, args...).CombinedOutput()
			if err != nil {
				t.Fatalf("%s failed to run the hook: %v\n%s", shell, err, out)
			}
			if !strings.Contains(string(out), "got: eval work") {
				t.Errorf("uso work did not call the binary with 'eval work'; output:\n%s", out)
			}
		})
	}
}

// setupProfiles points uso at a temp config with one tool and the profiles
// home, work and personal. work sets ANTHROPIC_API_KEY; personal sets GH_TOKEN.
func setupProfiles(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("USO_DIR", filepath.Join(root, "uso"))
	t.Setenv("USO_PROFILE", "")

	tools := map[string]config.Tool{"claude": {
		EnvVar:      "CLAUDE_CONFIG_DIR",
		Symlink:     filepath.Join(root, ".claude"),
		AccountsDir: filepath.Join(root, ".claude-accounts"),
	}}
	steps := []error{config.EnsureInit(), config.SaveTool("claude", tools["claude"])}
	for _, p := range []string{"home", "work", "personal"} {
		steps = append(steps, config.EnsureProfileDir(p, tools))
	}
	steps = append(steps,
		config.SetProfileKey("work", "ANTHROPIC_API_KEY", "sk-work"),
		config.SetProfileKey("personal", "GH_TOKEN", "gh-personal"),
	)
	for _, err := range steps {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestEvalSwitchUnsetsTheVariablesOfThisShellsProfile(t *testing.T) {
	tests := []struct {
		name       string
		usoProfile string // this shell's $USO_PROFILE
		wantUnset  string
		wantKept   string
	}{
		{
			name:       "this shell is on work while another terminal last switched to personal",
			usoProfile: "work",
			wantUnset:  "ANTHROPIC_API_KEY",
			wantKept:   "GH_TOKEN",
		},
		{
			name:       "falls back to the last profile when USO_PROFILE is not set",
			usoProfile: "",
			wantUnset:  "GH_TOKEN",
			wantKept:   "ANTHROPIC_API_KEY",
		},
		{
			name:       "falls back to the last profile when USO_PROFILE is not a valid name",
			usoProfile: "../work",
			wantUnset:  "GH_TOKEN",
			wantKept:   "ANTHROPIC_API_KEY",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupProfiles(t)
			if err := config.SetCurrentProfile("personal"); err != nil {
				t.Fatal(err)
			}
			t.Setenv("USO_PROFILE", tt.usoProfile)

			out, err := EvalSwitch("home")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out, "unset "+tt.wantUnset) {
				t.Errorf("expected 'unset %s'; output:\n%s", tt.wantUnset, out)
			}
			if strings.Contains(out, "unset "+tt.wantKept) {
				t.Errorf("did not expect 'unset %s'; output:\n%s", tt.wantKept, out)
			}
		})
	}
}

func TestEvalRestoreUnsetsAnInheritedProfilesVariables(t *testing.T) {
	setupProfiles(t)
	if err := config.SetCurrentProfile("personal"); err != nil {
		t.Fatal(err)
	}
	// A new shell (e.g. a tmux pane) inherited the environment of a shell on work.
	t.Setenv("USO_PROFILE", "work")

	out, err := EvalRestore()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "unset ANTHROPIC_API_KEY") {
		t.Errorf("expected work's ANTHROPIC_API_KEY to be unset; output:\n%s", out)
	}
	if !strings.Contains(out, "export GH_TOKEN='gh-personal'") {
		t.Errorf("expected personal's GH_TOKEN to be exported; output:\n%s", out)
	}
}

func TestEvalRestoreUnsetsNothingWhenAlreadyOnTheProfile(t *testing.T) {
	setupProfiles(t)
	if err := config.SetCurrentProfile("personal"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("USO_PROFILE", "personal")

	out, err := EvalRestore()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "unset ") {
		t.Errorf("expected no unset; output:\n%s", out)
	}
}

func TestTitleAndColorReachTheTerminalVerbatim(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh is not installed")
	}

	tests := []struct {
		name, key, value string
		want             string // what the terminal must receive
	}{
		{
			name:  "title with quotes, percent signs and a backslash",
			key:   "USO_TITLE",
			value: `Bob's "work" 100% done \n`,
			want:  "\033]0;Bob's \"work\" 100% done \\n\a",
		},
		{
			name:  "title with command substitution",
			key:   "USO_TITLE",
			value: "$(touch pwned) `touch pwned` $HOME",
			want:  "\033]0;$(touch pwned) `touch pwned` $HOME\a",
		},
		{
			name:  "color edited by hand to contain a command",
			key:   "USO_COLOR",
			value: "1,2,$(touch pwned)",
			want:  "\033]6;1;bg;blue;brightness;$(touch pwned)\a",
		},
	}
	evals := map[string]func() (string, error){
		"switch":  func() (string, error) { return EvalSwitch("work") },
		"restore": EvalRestore,
	}
	for _, tt := range tests {
		for evalName, eval := range evals {
			t.Run(tt.name+"/"+evalName, func(t *testing.T) {
				setupProfiles(t)
				if err := config.SetProfileKey("work", tt.key, tt.value); err != nil {
					t.Fatal(err)
				}
				if err := config.SetCurrentProfile("work"); err != nil {
					t.Fatal(err)
				}
				script, err := eval()
				if err != nil {
					t.Fatal(err)
				}

				dir := t.TempDir()
				cmd := exec.Command(sh, "-c", script)
				cmd.Dir = dir
				out, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("the shell rejected the output: %v\n%s\nscript:\n%s", err, out, script)
				}
				if !strings.Contains(string(out), tt.want) {
					t.Errorf("terminal received %q, want it to contain %q", out, tt.want)
				}
				if _, err := os.Stat(filepath.Join(dir, "pwned")); err == nil {
					t.Error("the value ran a command")
				}
			})
		}
	}
}
