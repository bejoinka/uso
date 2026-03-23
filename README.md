# uso

Switch AI CLI tool profiles with a single command.

```
$ uso work
-> claude: work  [~/.claude-accounts/work]
-> codex:  work  [~/.codex-accounts/work]
```

One command switches all your AI CLI tools — Claude Code, Codex, and anything else with file-based config — to the same profile. Each profile carries its own env vars, API keys, and settings. No dependencies, just shell.

## Why

If you freelance, consult, or just keep work and personal separate, you need different API keys, MCP servers, and settings for each context. Today that means logging out and back in, or juggling shell aliases. uso makes it one command.

## Install

**Homebrew:**

```sh
brew tap bejoinka/tap
brew install uso
```

Then add to your `~/.zshrc` or `~/.bashrc`:

```sh
source "$(brew --prefix)/share/uso/uso.sh"
```

**curl:**

```sh
curl -fsSL https://raw.githubusercontent.com/bejoinka/uso/main/install.sh | sh
```

**Manual:**

```sh
git clone https://github.com/bejoinka/uso.git ~/.local/share/uso
echo 'source ~/.local/share/uso/uso.sh' >> ~/.zshrc
```

## Quick Start

```sh
# Interactive setup — pick which tools to manage
uso init

# Create profiles
uso add-profile work
uso add-profile personal

# Configure env vars and visuals
uso set work USO_TITLE "work"
uso set work USO_COLOR "0,168,120"
uso set work CUSTOM_API_KEY "sk-..."

uso set personal USO_TITLE "personal"
uso set personal USO_COLOR "139,92,246"

# Switch
uso work
uso personal
```

## Commands

| Command | Description |
|---------|-------------|
| `uso <profile>` | Switch to a profile |
| `uso` | Show current profile |
| `uso list` | List profiles |
| `uso status` | Detailed status |
| `uso init` | Interactive setup (pick tools) |
| `uso add-profile <name>` | Create a profile |
| `uso set <profile> <KEY> [value]` | Set/unset env var or config |
| `uso show [profile]` | Show profile config |
| `uso add-tool <args>` | Register a custom CLI tool |
| `uso help` | Help |

## How It Works

When you run `uso work`:

1. Switches each registered tool's symlink to the profile's config dir
2. Exports all env vars defined in the profile
3. Applies `USO_COLOR` (iTerm tab) and `USO_TITLE` (terminal title) if set
4. Runs the post-switch hook if one exists

On shell startup, uso restores the last active profile.

## Profile Config

Profiles store env vars in `~/.config/uso/profiles/<name>.conf` as KEY=VALUE pairs. All keys are exported on switch. Two special keys get visual treatment:

| Key | Effect |
|-----|--------|
| `USO_COLOR` | Sets iTerm2 tab color (R,G,B values 0-255) |
| `USO_TITLE` | Sets terminal title |

Everything else is exported as a regular env var:

```sh
uso set work ANTHROPIC_API_KEY "sk-ant-..."
uso set work OPENAI_API_KEY "sk-proj-..."
uso set work NODE_ENV "development"
```

## Supported Tools

`uso init` offers these out of the box:

| Tool | env_var | symlink |
|------|---------|---------|
| Claude Code | `CLAUDE_CONFIG_DIR` | `~/.claude` |
| Codex CLI | `CODEX_HOME` | `~/.codex` |

Add custom tools with `uso add-tool`:

```sh
uso add-tool gemini GEMINI_CONFIG_DIR ~/.gemini ~/.gemini-accounts
uso add-tool cursor CURSOR_CONFIG_DIR ~/.cursor ~/.cursor-accounts
```

## Hooks

Create `~/.config/uso/post-switch` (chmod +x) to run custom logic after every switch. Receives the profile name as `$1`.

See [examples/post-switch](examples/post-switch) for an example with iTerm tab colors.

## Migrating Existing Configs

If your tool's config is a real directory (not a symlink yet):

```sh
# Copy existing config to "home" profile
mkdir -p ~/.claude-accounts/home
cp -a ~/.claude/* ~/.claude-accounts/home/

# Replace with symlink
mv ~/.claude ~/.claude.bak
ln -sfn ~/.claude-accounts/home ~/.claude
```

## Profile Names

Lowercase alphanumeric and hyphens only. Max 32 characters.

```
work           # ok
my-client      # ok
Client A       # invalid
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `USO_DIR` | `~/.config/uso` | Where uso stores its config |
| `USO_PROFILE` | _(set on switch)_ | Active profile name |

## Requirements

- bash 4+ or zsh 5+
- No external dependencies

## License

MIT
