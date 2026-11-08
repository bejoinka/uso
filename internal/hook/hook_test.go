package hook

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
