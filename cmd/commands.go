package cmd

import (
	"fmt"
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
	for _, idx := range selected {
		t := known[idx]
		if err := config.SaveTool(t.ID, config.Tool{
			EnvVar:      t.EnvVar,
			Symlink:     t.Symlink,
			AccountsDir: t.AccountsDir,
		}); err != nil {
			return err
		}
		os.MkdirAll(filepath.Join(t.AccountsDir, "home"), 0755)
		fmt.Printf("  + %s\n", t.Label)
	}

	fmt.Println()
	fmt.Println("Next: uso add-profile <name>")
	return nil
}

func runList(cmd *cobra.Command, args []string) error {
	current := config.CurrentProfile()
	profiles, err := config.ListProfiles()
	if err != nil {
		return err
	}

	marker := func(name string) string {
		if name == current {
			return "* "
		}
		return "  "
	}

	fmt.Printf("%shome\n", marker("home"))
	for _, p := range profiles {
		fmt.Printf("%s%s\n", marker(p), p)
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

	os.MkdirAll(filepath.Join(accountsDir, "home"), 0755)
	fmt.Printf("Added tool: %s ($%s -> %s)\n", name, envVar, symlink)
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
	fmt.Printf("  uso set %s USO_COLOR \"R,G,B\"\n", profile)
	return nil
}

func runSet(cmd *cobra.Command, args []string) error {
	profile := args[0]
	key := args[1]
	value := ""
	if len(args) > 2 {
		value = strings.Join(args[2:], " ")
	}

	if profile != "home" && !config.ProfileExists(profile) {
		return fmt.Errorf("profile '%s' not found", profile)
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

	if profile != "home" && !config.ProfileExists(profile) {
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
