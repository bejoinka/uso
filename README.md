# uso

Switch AI CLI tool profiles with a single command.

```
$ uso work
-> claude: work  [~/.claude-accounts/work]
-> codex:  work  [~/.codex-accounts/work]

$ uso personal
-> claude: personal  [~/.claude-accounts/personal]
-> codex:  personal  [~/.codex-accounts/personal]
```

If you work across multiple accounts — freelance clients, day job, side projects — you need different API keys, MCP servers, and settings for each. uso switches all your AI CLI tools at once.

## Install

```sh
brew tap bejoinka/tap
brew install uso
```

Or with Go:

```sh
go install github.com/bejoinka/uso@latest
```

Then add one line to your `~/.zshrc`:

```sh
eval "$(command uso hook zsh)"
```

(For bash, use `eval "$(command uso hook bash)"` in `~/.bashrc`.)

Restart your shell.

## Setup

### 1. Pick your tools

```sh
uso init
```

An interactive picker lets you choose which tools to manage:

```
> [x] Claude Code
  [x] Codex CLI

  arrows move, space toggles, enter confirms
```

This registers each tool with its config directory and env var. You can always add more later.

### 2. Create profiles

```sh
uso add-profile work
uso add-profile personal
```

This creates config directories for each tool:

```
  claude: ~/.claude-accounts/work/
  codex:  ~/.codex-accounts/work/
```

### 3. Migrate existing configs

If you've already been using Claude Code or Codex, your config lives in a real directory (e.g. `~/.claude/`). You need to turn it into a symlink so uso can swap it:

```sh
# Copy your current config into the "home" profile
cp -a ~/.claude/* ~/.claude-accounts/home/

# Replace the directory with a symlink
mv ~/.claude ~/.claude.bak
ln -sfn ~/.claude-accounts/home ~/.claude
```

Do the same for `~/.codex` → `~/.codex-accounts/home` (or any other tool).

### 4. Set up API keys and env vars

Each profile can carry its own environment variables. These get exported when you switch.

```sh
# Work profile uses a company API key
uso set work ANTHROPIC_API_KEY "sk-ant-..."
uso set work OPENAI_API_KEY "sk-proj-..."

# Personal profile uses your own keys
uso set personal ANTHROPIC_API_KEY "sk-ant-..."
uso set personal OPENAI_API_KEY "sk-proj-..."
```

You can set any env var — they're all exported on switch:

```sh
uso set work AWS_PROFILE "company-prod"
uso set work NODE_ENV "development"
```

### 5. Set up terminal visuals (optional)

Two special keys change your terminal appearance on switch, so you always know which profile is active:

```sh
# iTerm2 tab color (R,G,B 0-255)
uso set work USO_COLOR "0,168,120"
uso set personal USO_COLOR "139,92,246"

# Terminal title
uso set work USO_TITLE "work"
uso set personal USO_TITLE "personal"
```

### 6. Switch

```sh
uso work
```

That's it. All your tools now point to the `work` config, your API keys are exported, and your terminal tab changes color.

## Adding a custom tool

uso ships with Claude Code and Codex CLI built in. Add anything else that uses file-based config:

```sh
uso add-tool <name> <env_var> <symlink_path> <accounts_dir>
```

For example:

```sh
uso add-tool gemini GEMINI_CONFIG_DIR ~/.gemini ~/.gemini-accounts
uso add-tool cursor CURSOR_CONFIG_DIR ~/.cursor ~/.cursor-accounts
```

After adding a tool, create its profile directories:

```sh
uso add-profile work   # creates ~/.gemini-accounts/work/, etc.
```

## Commands

| Command | What it does |
|---------|-------------|
| `uso <profile>` | Switch to a profile |
| `uso` | Print the current profile |
| `uso list` | List all profiles |
| `uso status` | Show tools, symlinks, and env vars |
| `uso show <profile>` | Show a profile's config |
| `uso set <profile> <KEY> <value>` | Set an env var on a profile |
| `uso set <profile> <KEY>` | Remove an env var |
| `uso init` | Interactive tool setup |
| `uso add-profile <name>` | Create a new profile |
| `uso add-tool <name> ...` | Register a custom tool |
| `uso help` | Help |

## How it works

`eval "$(command uso hook zsh)"` installs a shell function that wraps the `uso` binary. When you type `uso work`:

1. The binary reads your registered tools and the profile config
2. Swaps each tool's symlink (e.g. `~/.claude` → `~/.claude-accounts/work`)
3. Outputs `export` and `printf` statements
4. The shell function evals them into your session

On shell startup, the hook restores your last active profile — you pick up where you left off.

## Profile names

Lowercase letters, numbers, and hyphens. Max 32 characters.

```
work           ✓
my-client      ✓
Client A       ✗
```

## Hooks

For custom logic beyond env vars and terminal colors, create an executable script at `~/.config/uso/post-switch`. It receives the profile name as `$1`.

## File layout

```
~/.config/uso/
├── current                    # last active profile
├── tools/
│   ├── claude.conf            # ENV_VAR, SYMLINK, ACCOUNTS_DIR
│   └── codex.conf
└── profiles/
    ├── work.conf              # KEY=VALUE pairs (exported on switch)
    └── personal.conf

~/.claude-accounts/
├── home/                      # default profile
├── work/                      # work configs, sessions, MCP settings
└── personal/

~/.codex-accounts/
├── home/
├── work/
└── personal/
```

## License

MIT
