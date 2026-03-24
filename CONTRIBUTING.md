# Contributing

uso is a small Go project. Contributions are welcome — especially adding support for new tools.

## Adding a new tool

The easiest way to contribute. Each tool needs three things:

1. **The env var** the tool reads for its config directory
2. **The default symlink path** (e.g. `~/.claude`)
3. **The accounts directory** where profiles are stored (e.g. `~/.claude-accounts`)

To add a built-in tool, edit `internal/config/config.go` and add an entry to `KnownTools()`:

```go
func KnownTools() []KnownTool {
    home, _ := os.UserHomeDir()
    return []KnownTool{
        {"claude", "CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude"), filepath.Join(home, ".claude-accounts"), "Claude Code"},
        {"codex", "CODEX_HOME", filepath.Join(home, ".codex"), filepath.Join(home, ".codex-accounts"), "Codex CLI"},
        // add yours here
    }
}
```

If you're not sure about the env var or config path, open an issue and we'll figure it out together.

## Development

```sh
git clone https://github.com/bejoinka/uso.git
cd uso
go build -o uso .
./uso --help
```

Test the eval output directly:

```sh
./uso eval <profile>       # see what shell commands would be generated
./uso hook zsh             # see the shell hook output
./uso _list                # run commands without the eval wrapper
```

## Project structure

```
main.go                    # entry point
cmd/
  root.go                  # cobra CLI setup, eval command, hook command
  commands.go              # init, list, status, set, show, add-tool, add-profile
internal/
  config/
    config.go              # profiles, tools, file I/O
    colors.go              # named color resolution
    migrate.go             # auto-migration of real dirs to symlinks
  hook/
    hook.go                # shell hook generation, eval output
  tui/
    multiselect.go         # interactive picker (bubbletea)
```

## Pull requests

- Keep changes focused — one feature or fix per PR
- Test with both `zsh` and `bash` if you're touching hook output
- Run `go build` before submitting (no CI yet)

## Adding a named color

Edit `internal/config/colors.go` and add to the `namedColors` map. Keep the list in `ColorNames()` in sync.
