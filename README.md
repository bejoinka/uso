# uso

Switch AI CLI tool profiles with a single command.

```sh
$ uso work
→ claude: work  [~/.claude-accounts/work]
→ codex:  work  [~/.codex-accounts/work]
```

One command switches all your AI CLI tools — Claude Code, Codex, Gemini CLI, and anything else that uses file-based config — to the same profile. No dependencies, just shell.

## Why

If you freelance, consult, or just keep work and personal separate, you need different API keys, MCP servers, and settings for each context. Today that means logging out and back in, or juggling shell aliases. uso makes it one command.

## Install

```sh
git clone https://github.com/bejoinka/uso.git ~/.local/share/uso
echo 'source ~/.local/share/uso/uso.sh' >> ~/.zshrc
source ~/.zshrc
```

Or run the installer:

```sh
curl -fsSL https://raw.githubusercontent.com/bejoinka/uso/main/install.sh | sh
```

## Quick Start

```sh
# Initialize
uso init

# Register your tools
uso add-tool claude CLAUDE_CONFIG_DIR ~/.claude ~/.claude-accounts
uso add-tool codex  CODEX_HOME        ~/.codex ~/.codex-accounts

# Create profiles
uso add-profile work
uso add-profile personal

# Switch
uso work
uso personal
```

## Commands

| Command | Description |
|---------|-------------|
| `uso <profile>` | Switch to a profile |
| `uso` | Show current profile |
| `uso list` | List all profiles |
| `uso status` | Show detailed status |
| `uso add-tool <args>` | Register a CLI tool |
| `uso add-profile <name>` | Create a new profile |
| `uso init` | Initialize uso |
| `uso help` | Show help |

## How It Works

uso manages symlinks and environment variables. When you run `uso work`:

1. For each registered tool, points the symlink (e.g. `~/.claude`) at the profile's config dir (e.g. `~/.claude-accounts/work`)
2. Exports the tool's env var (e.g. `CLAUDE_CONFIG_DIR`) so new processes pick it up
3. Sets `USO_PROFILE=work`
4. Runs your post-switch hook if one exists

On shell startup, uso restores the last active profile.

## Supported Tools

uso works with any CLI tool that stores config in a known directory:

| Tool | env_var | symlink | accounts_dir |
|------|---------|---------|-------------|
| Claude Code | `CLAUDE_CONFIG_DIR` | `~/.claude` | `~/.claude-accounts` |
| Codex CLI | `CODEX_HOME` | `~/.codex` | `~/.codex-accounts` |
| Gemini CLI | `GEMINI_CONFIG_DIR` | `~/.gemini` | `~/.gemini-accounts` |

Add your own with `uso add-tool`.

## Hooks

Create `~/.config/uso/post-switch` to run custom logic after every switch:

```sh
#!/bin/sh
profile="$1"

# iTerm2 tab colors
case "$profile" in
  work)     printf "\033]6;1;bg;red;brightness;0\a\033]6;1;bg;green;brightness;168\a\033]6;1;bg;blue;brightness;120\a" ;;
  personal) printf "\033]6;1;bg;red;brightness;139\a\033]6;1;bg;green;brightness;92\a\033]6;1;bg;blue;brightness;246\a" ;;
esac

printf "\033]0;%s\a" "$profile"
```

```sh
chmod +x ~/.config/uso/post-switch
```

See [examples/post-switch](examples/post-switch) for a full example.

## Migrating Existing Configs

If your tool's config is a real directory (not a symlink yet):

```sh
# Copy existing config to "home" profile
mkdir -p ~/.claude-accounts/home
cp -a ~/.claude/* ~/.claude-accounts/home/

# Replace with symlink
mv ~/.claude ~/.claude.bak
ln -sfn ~/.claude-accounts/home ~/.claude

# Register
uso add-tool claude CLAUDE_CONFIG_DIR ~/.claude ~/.claude-accounts
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `USO_DIR` | `~/.config/uso` | Where uso stores its config |
| `USO_PROFILE` | _(set after switch)_ | Active profile name |

## Requirements

- bash 4+ or zsh 5+
- No external dependencies

## License

MIT
