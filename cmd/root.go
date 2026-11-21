package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/bejoinka/uso/internal/config"
	"github.com/bejoinka/uso/internal/hook"
	"github.com/spf13/cobra"
)

// rootHelp is printed for `uso`, `uso help`, `uso --help` and `uso -h`. It
// lists commands as typed through the shell function, not the binary's
// internal _-prefixed subcommands. The Homebrew formula's test checks for
// the first line.
const rootHelp = `Switch AI CLI tool profiles with a single command.

Usage:
  uso <profile>                      Switch to a profile
  uso                                Print the profile you switched to most recently
  uso list                           List profiles
  uso status                         Show the current profile, its env vars and tool symlinks
  uso show [profile]                 Show a profile's env vars
  uso set <profile> <KEY> <value>    Set an env var on a profile
  uso set <profile> <KEY>            Remove an env var from a profile
  uso add-profile <name>             Create a profile
  uso add-profile <name> --share <profile>
                                     Create or convert a profile that shares settings,
                                     skills, plugins and project memory with another;
                                     only the login stays separate
  uso copy-session <session> <profile>
                                     Copy a Claude Code session, by ID, ID prefix or
                                     /rename title, into another profile to resume it there
  uso menubar [--claude-flags <flags>]
                                     Print a SwiftBar plugin that lists recent Claude Code
                                     sessions in the menu bar; a click copies the command
                                     that resumes one
  uso add-tool <name> <ENV_VAR> <default-dir> <accounts-dir>
                                     Register another tool
  uso init                           Set up uso and move your existing config into a default profile
  uso hook zsh|bash                  Print the shell hook
  uso help                           Show this help

Setup:
  Add this line to ~/.zshrc (for bash: hook bash, in ~/.bashrc), open a new
  terminal, and run uso init:

    eval "$(command uso hook zsh)"

More: https://github.com/bejoinka/uso
`

var rootCmd = &cobra.Command{
	Use:   "uso",
	Short: "Switch AI CLI tool profiles with a single command",
	// Argument errors carry their own usage line; see usageArgs
	SilenceUsage: true,
	// If called directly (not via shell wrapper), show help
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// ── hook ────────────────────────────────────────────────────────

var hookCmd = &cobra.Command{
	Use:   "hook [zsh|bash]",
	Short: "Output shell hook for eval",
	Args:  usageArgs(cobra.ExactArgs(1), "uso hook zsh|bash"),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "zsh":
			fmt.Print(hook.Zsh())
		case "bash":
			fmt.Print(hook.Bash())
		default:
			return fmt.Errorf("unsupported shell: %s (use zsh or bash)", args[0])
		}
		return nil
	},
}

// ── eval (called by shell wrapper) ──────────────────────────────

var evalCmd = &cobra.Command{
	Use:    "eval [profile|subcommand] [args...]",
	Short:  "Output shell commands (used by the shell wrapper)",
	Hidden: true,
	Args:   cobra.ArbitraryArgs,
	// Everything after `eval` is what the user typed after `uso`, so
	// `uso --help` and values like --max-old-space-size=4096 must reach
	// RunE as arguments, not be parsed as eval's own flags.
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()
		if len(args) > 0 && args[0] == "--restore" {
			script, err := hook.EvalRestore()
			if err != nil {
				return err
			}
			fmt.Fprint(out, script)
			return nil
		}

		if len(args) == 0 {
			// uso (no args) → print current profile
			fmt.Fprintf(out, "echo '%s'\n", hook.ShellEscape(config.CurrentProfile()))
			return nil
		}

		switch args[0] {
		case "init":
			// init is interactive, run directly
			passThrough(out, "_init", nil)
		case "list", "ls":
			passThrough(out, "_list", nil)
		case "status":
			passThrough(out, "_status", nil)
		case "add-tool":
			passThrough(out, "_add-tool", args[1:])
		case "add-profile":
			passThrough(out, "_add-profile", args[1:])
		case "set":
			passThrough(out, "_set", args[1:])
		case "show":
			passThrough(out, "_show", args[1:])
		case "copy-session":
			passThrough(out, "_copy-session", args[1:])
		case "menubar":
			passThrough(out, "_menubar", args[1:])
		case "hook":
			passThrough(out, "hook", args[1:])
		case "help", "--help", "-h":
			passThrough(out, "--help", nil)
		default:
			// Treat as profile name → switch
			script, err := hook.EvalSwitch(args[0])
			if err != nil {
				return err
			}
			fmt.Fprint(out, script)
		}
		return nil
	},
}

// passThrough prints a command that runs one of the binary's own
// subcommands directly, with the user's arguments single-quoted so the
// shell passes them through verbatim.
func passThrough(out io.Writer, sub string, args []string) {
	fmt.Fprintf(out, "command %s %s", os.Args[0], sub)
	for _, a := range args {
		fmt.Fprintf(out, " '%s'", hook.ShellEscape(a))
	}
	fmt.Fprintln(out)
}

// ── Direct subcommands (called via `command uso _xxx`) ──────────

var initCmd = &cobra.Command{
	Use:   "_init",
	Short: "Interactive setup",
	RunE:  runInit,
}

var listCmd = &cobra.Command{
	Use:   "_list",
	Short: "List profiles",
	RunE:  runList,
}

var statusCmd = &cobra.Command{
	Use:   "_status",
	Short: "Detailed status",
	RunE:  runStatus,
}

var addToolCmd = &cobra.Command{
	Use:   "_add-tool <name> <env_var> <symlink> <accounts_dir>",
	Short: "Register a CLI tool",
	Args:  usageArgs(cobra.ExactArgs(4), "uso add-tool <name> <ENV_VAR> <default-dir> <accounts-dir>"),
	RunE:  runAddTool,
}

var addProfileCmd = &cobra.Command{
	Use:   "_add-profile <name>",
	Short: "Create a profile",
	Args:  usageArgs(cobra.ExactArgs(1), "uso add-profile <name> [--share <profile>]"),
	RunE:  runAddProfile,
}

var setCmd = &cobra.Command{
	Use:   "_set <profile> <KEY> [value]",
	Short: "Set/unset env var on a profile",
	Args:  usageArgs(cobra.MinimumNArgs(2), "uso set <profile> <KEY> [value]"),
	RunE:  runSet,
}

var showCmd = &cobra.Command{
	Use:   "_show [profile]",
	Short: "Show profile config",
	Args:  usageArgs(cobra.MaximumNArgs(1), "uso show [profile]"),
	RunE:  runShow,
}

var copySessionCmd = &cobra.Command{
	Use:   "_copy-session <session> <profile>",
	Short: "Copy a Claude Code session into another profile",
	Args:  usageArgs(cobra.ExactArgs(2), "uso copy-session <session> <profile> [--from <profile>]"),
	RunE:  runCopySession,
}

var menubarCmd = &cobra.Command{
	Use:   "_menubar",
	Short: "Print a SwiftBar plugin that lists recent Claude Code sessions",
	Args:  usageArgs(cobra.NoArgs, "uso menubar [--claude-flags <flags>]"),
	RunE:  runMenubar,
}

var swiftbarCmd = &cobra.Command{
	Use:    "_swiftbar <plugin> [copy <profile> <id>]",
	Short:  "Print the menu, or copy a session's resume command (run by the plugin)",
	Hidden: true,
	Args: cobra.MatchAll(cobra.RangeArgs(1, 4), func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 && len(args) != 4 {
			return fmt.Errorf("usage: uso _swiftbar <plugin> [copy <profile> <id>]")
		}
		return nil
	}),
	RunE: runSwiftbar,
}

var completionsCmd = &cobra.Command{
	Use:    "completions",
	Short:  "Completion helpers",
	Hidden: true,
}

var completionsProfilesCmd = &cobra.Command{
	Use:    "profiles",
	Short:  "List profile names for completion",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		profiles, _ := config.ListProfiles()
		for _, p := range profiles {
			fmt.Println(p)
		}
		return nil
	},
}

// usageArgs wraps an argument check so that its error shows the command as
// typed through the shell function ("uso set"), not the internal one
// ("uso _set").
func usageArgs(check cobra.PositionalArgs, usage string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if check(cmd, args) != nil {
			return fmt.Errorf("usage: %s", usage)
		}
		return nil
	}
}

func init() {
	// Replace the root help, which would list the internal subcommands, but
	// keep cobra's help for subcommands such as hook.
	defaultHelp := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd != rootCmd {
			defaultHelp(cmd, args)
			return
		}
		fmt.Fprint(cmd.OutOrStdout(), rootHelp)
	})

	// A value such as --max-old-space-size=4096 is an argument, not a flag
	setCmd.Flags().SetInterspersed(false)

	addProfileCmd.Flags().String("share", "", "share settings, skills, plugins and project memory with this profile")

	copySessionCmd.Flags().String("from", "", "the profile to copy from, when the session is in several")
	menubarCmd.Flags().String("claude-flags", "", "options to add to the claude command that a click copies")

	completionsCmd.AddCommand(completionsProfilesCmd)

	rootCmd.AddCommand(hookCmd, evalCmd, initCmd, listCmd, statusCmd,
		addToolCmd, addProfileCmd, setCmd, showCmd, copySessionCmd, menubarCmd, swiftbarCmd, completionsCmd)
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
