package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bejoinka/uso/internal/config"
	"github.com/bejoinka/uso/internal/tui"
	"github.com/spf13/cobra"
)

func runInit(cmd *cobra.Command, args []string) error {
	if err := config.EnsureInit(); err != nil {
		return err
	}

	// 1. Pick tools
	known := config.KnownTools()
	labels := make([]string, len(known))
	for i, t := range known {
		labels[i] = t.Label
	}

	fmt.Println("Select tools to manage:")
	selected, err := tui.MultiSelect(labels)
	if err != nil {
		return err
	}
	fmt.Println()

	// 2. Ask for default profile name
	defaultProfile := "home"
	fmt.Printf("Default profile name [%s]: ", defaultProfile)
	var input string
	fmt.Scanln(&input)
	input = strings.TrimSpace(input)
	if input != "" {
		if !config.ValidName(input) {
			return fmt.Errorf("invalid name '%s' (lowercase alphanumeric + hyphens, max 32)", input)
		}
		defaultProfile = input
	}
	fmt.Println()

	// 3. Register tools and migrate existing configs
	tools := make([]config.KnownTool, len(selected))
	for i, idx := range selected {
		tools[i] = known[idx]
	}
	if err := applyInit(cmd.OutOrStdout(), tools, defaultProfile); err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("Your existing setup is now the '%s' profile, your default. `uso list` marks it.\n", defaultProfile)
	fmt.Println("Next: uso add-profile <name>")
	return nil
}

// applyInit registers the chosen tools, migrates their existing config into
// the default profile, and makes that profile the default and current one.
func applyInit(out io.Writer, tools []config.KnownTool, defaultProfile string) error {
	for _, t := range tools {
		if err := config.SaveTool(t.ID, config.Tool{
			EnvVar:      t.EnvVar,
			Symlink:     t.Symlink,
			AccountsDir: t.AccountsDir,
		}); err != nil {
			return err
		}

		result := config.MigrateTool(t.ID, t.Symlink, t.AccountsDir, defaultProfile)
		if result.Error != nil {
			fmt.Fprintf(out, "  ! %s: %s\n", t.Label, result.Error)
		} else if result.Copied {
			fmt.Fprintf(out, "  + %s (migrated %s -> %s/%s/)\n", t.Label, t.Symlink, t.AccountsDir, defaultProfile)
		} else if result.WasSymlink {
			fmt.Fprintf(out, "  + %s (already a symlink)\n", t.Label)
		} else {
			fmt.Fprintf(out, "  + %s\n", t.Label)
		}
		if result.Error != nil {
			continue
		}

		copied, err := config.CopyStateFile(t.StateFile, filepath.Join(t.AccountsDir, defaultProfile))
		if err != nil {
			fmt.Fprintf(out, "  ! %s: copy %s: %s\n", t.Label, t.StateFile, err)
		} else if copied {
			fmt.Fprintf(out, "    copied %s into %s/%s/\n", t.StateFile, t.AccountsDir, defaultProfile)
		}
	}

	registered, err := config.LoadTools()
	if err != nil {
		return err
	}
	if err := config.EnsureProfileDir(defaultProfile, registered); err != nil {
		return err
	}
	if err := config.SetDefaultProfile(defaultProfile); err != nil {
		return err
	}
	return config.SetCurrentProfile(defaultProfile)
}

func runList(cmd *cobra.Command, args []string) error {
	current := config.CurrentProfile()
	defaultProfile := config.DefaultProfile()
	profiles, err := config.ListProfiles()
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	for _, p := range profiles {
		marker, suffix := "  ", ""
		if p == current {
			marker = "* "
		}
		if p == defaultProfile {
			suffix = " (default)"
		}
		fmt.Fprintf(out, "%s%s%s\n", marker, p, suffix)
	}
	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	current := config.CurrentProfile()
	fmt.Printf("profile: %s\n\n", current)

	env, _ := config.LoadProfileEnv(current)
	if len(env) > 0 {
		fmt.Println("  env:")
		for _, kv := range env {
			fmt.Printf("    %s=%s\n", kv[0], kv[1])
		}
		fmt.Println()
	}

	tools, err := config.LoadTools()
	if err != nil {
		return err
	}

	fmt.Println("  tools:")
	for name, t := range tools {
		target, err := os.Readlink(t.Symlink)
		if err != nil {
			target = "(not a symlink)"
		}
		fmt.Printf("    %s: %s -> %s\n", name, t.Symlink, target)
	}
	return nil
}

func runAddTool(cmd *cobra.Command, args []string) error {
	name, envVar, symlink, accountsDir := args[0], args[1], args[2], args[3]

	if !config.ValidName(name) {
		return fmt.Errorf("invalid name '%s' (lowercase alphanumeric + hyphens, max 32)", name)
	}

	home, _ := os.UserHomeDir()
	if strings.HasPrefix(symlink, "~") {
		symlink = filepath.Join(home, symlink[1:])
	}
	if strings.HasPrefix(accountsDir, "~") {
		accountsDir = filepath.Join(home, accountsDir[1:])
	}

	if err := config.SaveTool(name, config.Tool{
		EnvVar:      envVar,
		Symlink:     symlink,
		AccountsDir: accountsDir,
	}); err != nil {
		return err
	}

	// Give the tool a directory in every existing profile
	profiles, err := config.ListProfiles()
	if err != nil {
		return err
	}
	for _, p := range profiles {
		if err := os.MkdirAll(filepath.Join(accountsDir, p), 0755); err != nil {
			return err
		}
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Added tool: %s ($%s -> %s)\n", name, envVar, symlink)
	return nil
}

func runAddProfile(cmd *cobra.Command, args []string) error {
	profile := args[0]

	if !config.ValidName(profile) {
		return fmt.Errorf("invalid name '%s' (lowercase alphanumeric + hyphens, max 32)", profile)
	}

	tools, err := config.LoadTools()
	if err != nil {
		return err
	}

	if err := config.EnsureProfileDir(profile, tools); err != nil {
		return err
	}

	for name, t := range tools {
		fmt.Printf("  %s: %s/\n", name, filepath.Join(t.AccountsDir, profile))
	}

	fmt.Printf("\nProfile '%s' ready. Configure:\n", profile)
	fmt.Printf("  uso set %s USO_TITLE \"%s\"\n", profile, profile)
	fmt.Printf("  uso set %s USO_COLOR blue    # or: red, orange, teal, purple, 0,168,120\n", profile)
	return nil
}

func runSet(cmd *cobra.Command, args []string) error {
	profile := args[0]
	key := args[1]
	value := ""
	if len(args) > 2 {
		value = strings.Join(args[2:], " ")
	}

	if !config.ProfileExists(profile) {
		return fmt.Errorf("profile '%s' not found", profile)
	}

	// Resolve named colors for USO_COLOR
	if key == "USO_COLOR" && value != "" {
		resolved, err := config.ResolveColor(value)
		if err != nil {
			return err
		}
		value = resolved
	}

	if err := config.SetProfileKey(profile, key, value); err != nil {
		return err
	}

	if value == "" {
		fmt.Printf("Removed %s from %s\n", key, profile)
	} else {
		fmt.Printf("%s: %s=%s\n", profile, key, value)
	}
	return nil
}

func runShow(cmd *cobra.Command, args []string) error {
	profile := config.CurrentProfile()
	if len(args) > 0 {
		profile = args[0]
	}

	if !config.ProfileExists(profile) {
		return fmt.Errorf("profile '%s' not found", profile)
	}

	fmt.Printf("profile: %s\n", profile)

	env, _ := config.LoadProfileEnv(profile)
	if len(env) == 0 {
		fmt.Println("  (no config)")
		return nil
	}
	for _, kv := range env {
		fmt.Printf("  %s=%s\n", kv[0], kv[1])
	}
	return nil
}
