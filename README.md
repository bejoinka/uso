# uso

Switch AI CLI tool profiles with a single command.

```
$ uso work
-> claude: work  [~/.claude-accounts/work]
-> codex:  work  [~/.codex-accounts/work]
```

One command switches all your AI CLI tools — Claude Code, Codex, and anything else with file-based config — to the same profile. Each profile carries its own env vars, API keys, and settings. Single binary, no runtime dependencies.

## Install

**Homebrew:**

```sh
brew tap bejoinka/tap
brew install uso
```

**Go:**

```sh
go install github.com/bejoinka/uso@latest
```

Then add to your `~/.zshrc`:

```sh
eval "$(uso hook zsh)"
```

Or `~/.bashrc`:

```sh
eval "$(uso hook bash)"
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

`eval "$(uso hook zsh)"` in your shell rc installs a thin wrapper function. When you run `uso work`, the binary:

1. Switches each registered tool's symlink to the profile's config dir
2. Outputs `export` statements for the profile's env vars
3. Applies `USO_COLOR` (iTerm tab) and `USO_TITLE` (terminal title)
4. The shell wrapper evals all of this into your session

On shell startup, the hook restores the last active profile.

## Profile Config

Profiles store env vars as KEY=VALUE pairs. All keys are exported on switch. Two special keys get visual treatment:

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

Add custom tools:

```sh
uso add-tool gemini GEMINI_CONFIG_DIR ~/.gemini ~/.gemini-accounts
```

## Hooks

Create `~/.config/uso/post-switch` (chmod +x) to run custom logic after every switch. Receives the profile name as `$1`.

## Migrating Existing Configs

If your tool's config is a real directory (not a symlink yet):

```sh
mkdir -p ~/.claude-accounts/home
cp -a ~/.claude/* ~/.claude-accounts/home/
mv ~/.claude ~/.claude.bak
ln -sfn ~/.claude-accounts/home ~/.claude
```

## Profile Names

Lowercase alphanumeric and hyphens only. Max 32 characters.

## Requirements

- macOS or Linux
- zsh or bash

## License

MIT
