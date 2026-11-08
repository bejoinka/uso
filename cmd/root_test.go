package cmd

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/bejoinka/uso/internal/config"
	"github.com/spf13/cobra"
)

// execute runs the uso binary's root command with args and returns what it printed.
func execute(t *testing.T, args ...string) (string, error) {
	t.Helper()
	if args == nil {
		args = []string{} // nil would make cobra read the test binary's own arguments
	}
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs(args)
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
		// cobra keeps --help set between runs
		for _, c := range append([]*cobra.Command{rootCmd}, rootCmd.Commands()...) {
			if f := c.Flags().Lookup("help"); f != nil {
				f.Value.Set("false")
			}
		}
	})
	err := rootCmd.Execute()
	return out.String(), err
}

var internalCommands = []string{"_init", "_list", "_status", "_set", "_show", "_add-tool", "_add-profile"}

func TestHelpShowsCommandsAsUsersTypeThem(t *testing.T) {
	// The shell function runs `uso --help` for `uso help`, `uso --help` and `uso -h`.
	for _, args := range [][]string{{"--help"}, {"help"}, {}} {
		t.Run(strings.Join(append([]string{"uso"}, args...), " "), func(t *testing.T) {
			out, err := execute(t, args...)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{
				"Switch AI CLI tool profiles", // the Homebrew formula's test checks for this
				"uso <profile>",
				"uso list",
				"uso set <profile> <KEY> <value>",
				"uso add-tool <name> <ENV_VAR> <default-dir> <accounts-dir>",
				"uso init",
			} {
				if !strings.Contains(out, want) {
					t.Errorf("help does not mention %q:\n%s", want, out)
				}
			}
			for _, internal := range append(internalCommands, "completion") {
				if strings.Contains(out, internal) {
					t.Errorf("help shows the internal command %q:\n%s", internal, out)
				}
			}
		})
	}
}

func TestSubcommandHelpIsUnchanged(t *testing.T) {
	out, err := execute(t, "hook", "--help")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "uso hook [zsh|bash]") {
		t.Errorf("uso hook --help:\n%s", out)
	}
}

func TestArgumentErrorsShowCommandsAsUsersTypeThem(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"_set", "work"}, "usage: uso set <profile> <KEY> [value]"},
		{[]string{"_add-profile"}, "usage: uso add-profile <name>"},
		{[]string{"_add-tool", "gemini"}, "usage: uso add-tool <name> <ENV_VAR> <default-dir> <accounts-dir>"},
		{[]string{"_show", "a", "b"}, "usage: uso show [profile]"},
		{[]string{"hook"}, "usage: uso hook zsh|bash"},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			out, err := execute(t, tt.args...)
			if err == nil || err.Error() != tt.want {
				t.Errorf("error = %v, want %q", err, tt.want)
			}
			for _, internal := range internalCommands {
				if strings.Contains(out, internal) {
					t.Errorf("output shows the internal command %q:\n%s", internal, out)
				}
			}
		})
	}
}

// The shell function runs `uso eval <what the user typed>`, so eval must
// not treat the user's arguments as its own flags.
func TestEvalPassesFlagLikeArgumentsThrough(t *testing.T) {
	useTempConfig(t)
	bin := "command " + os.Args[0]
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"eval", "--help"}, bin + " --help\n"},
		{[]string{"eval", "-h"}, bin + " --help\n"},
		{
			[]string{"eval", "set", "work", "NODE_OPTIONS", "--max-old-space-size=4096"},
			bin + " _set 'work' 'NODE_OPTIONS' '--max-old-space-size=4096'\n",
		},
		{[]string{"eval", "--restore"}, ""}, // no tools configured, so nothing to restore
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			out, err := execute(t, tt.args...)
			if err != nil {
				t.Fatal(err)
			}
			if out != tt.want {
				t.Errorf("uso %s printed:\n%s\nwant:\n%s", strings.Join(tt.args, " "), out, tt.want)
			}
		})
	}
}

func TestSetStoresAValueThatStartsWithADash(t *testing.T) {
	useTempConfig(t)
	if err := config.EnsureProfileDir("work", nil); err != nil {
		t.Fatal(err)
	}

	if _, err := execute(t, "_set", "work", "NODE_OPTIONS", "--max-old-space-size=4096"); err != nil {
		t.Fatal(err)
	}

	env, err := config.LoadProfileEnv("work")
	if err != nil {
		t.Fatal(err)
	}
	if want := [][2]string{{"NODE_OPTIONS", "--max-old-space-size=4096"}}; !reflect.DeepEqual(env, want) {
		t.Errorf("work's env = %v, want %v", env, want)
	}
}
