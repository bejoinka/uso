package cmd

import (
	"fmt"
	"os"

	"github.com/bejoinka/uso/internal/config"
	"github.com/bejoinka/uso/internal/hook"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "uso",
	Short: "Switch AI CLI tool profiles with a single command",
	Long:  "uso manages symlinks and env vars across Claude Code, Codex, and other AI CLI tools.\nSet up with: eval \"$(uso hook zsh)\" in your .zshrc",
	// If called directly (not via shell wrapper), show help
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// ── hook ────────────────────────────────────────────────────────

var hookCmd = &cobra.Command{
	Use:   "hook [zsh|bash]",
	Short: "Output shell hook for eval",
	Args:  cobra.ExactArgs(1),
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
	Args:  cobra.ExactArgs(4),
	RunE:  runAddTool,
}

var addProfileCmd = &cobra.Command{
	Use:   "_add-profile <name>",
	Short: "Create a profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runAddProfile,
}

var setCmd = &cobra.Command{
	Use:   "_set <profile> <KEY> [value]",
	Short: "Set/unset env var on a profile",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runSet,
}

var showCmd = &cobra.Command{
	Use:   "_show [profile]",
	Short: "Show profile config",
	Args:  cobra.MaximumNArgs(1),
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
		fmt.Println("home")
		profiles, _ := config.ListProfiles()
		for _, p := range profiles {
			fmt.Println(p)
		}
		return nil
	},
}

func init() {
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
