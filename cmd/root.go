package cmd

import (
	"fmt"
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

var evalRestore bool

var evalCmd = &cobra.Command{
	Use:    "eval [profile|subcommand] [args...]",
	Short:  "Output shell commands (used by the shell wrapper)",
	Hidden: true,
	Args:   cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if evalRestore {
			out, err := hook.EvalRestore()
			if err != nil {
				return err
			}
			fmt.Print(out)
			return nil
		}

		if len(args) == 0 {
			// uso (no args) → print current profile
			fmt.Printf("echo '%s'\n", config.CurrentProfile())
			return nil
		}

		switch args[0] {
		case "init":
			// init is interactive, run directly
			fmt.Printf("command %s _init\n", os.Args[0])
			return nil
		case "list", "ls":
			fmt.Printf("command %s _list\n", os.Args[0])
			return nil
		case "status":
			fmt.Printf("command %s _status\n", os.Args[0])
			return nil
		case "add-tool":
			fmt.Printf("command %s _add-tool", os.Args[0])
			for _, a := range args[1:] {
				fmt.Printf(" '%s'", a)
			}
			fmt.Println()
			return nil
		case "add-profile":
			fmt.Printf("command %s _add-profile", os.Args[0])
			for _, a := range args[1:] {
				fmt.Printf(" '%s'", a)
			}
			fmt.Println()
			return nil
		case "set":
			fmt.Printf("command %s _set", os.Args[0])
			for _, a := range args[1:] {
				fmt.Printf(" '%s'", a)
			}
			fmt.Println()
			return nil
		case "show":
			fmt.Printf("command %s _show", os.Args[0])
			for _, a := range args[1:] {
				fmt.Printf(" '%s'", a)
			}
			fmt.Println()
			return nil
		case "help", "--help", "-h":
			fmt.Printf("command %s --help\n", os.Args[0])
			return nil
		default:
			// Treat as profile name → switch
			out, err := hook.EvalSwitch(args[0])
			if err != nil {
				return err
			}
			fmt.Print(out)
			return nil
		}
	},
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
	Args:  usageArgs(cobra.ExactArgs(1), "uso add-profile <name>"),
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

	evalCmd.Flags().BoolVar(&evalRestore, "restore", false, "restore last profile")

	completionsCmd.AddCommand(completionsProfilesCmd)

	rootCmd.AddCommand(hookCmd, evalCmd, initCmd, listCmd, statusCmd,
		addToolCmd, addProfileCmd, setCmd, showCmd, completionsCmd)
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
