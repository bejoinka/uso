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

**Homebrew:**

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

### 1. Run init

```sh
uso init
```

This does three things:

1. **Picks your tools** — an interactive selector lets you choose Claude Code, Codex CLI, or both
2. **Names your default profile** — defaults to `home`, but you can call it anything
3. **Migrates existing config** — if `~/.claude` or `~/.codex` is a real directory, uso copies it into your default profile and replaces it with a symlink automatically

```
Select tools to manage:
> [x] Claude Code
  [x] Codex CLI

  arrows move, space toggles, enter confirms

Default profile name [home]:

  + Claude Code (migrated ~/.claude -> ~/.claude-accounts/home/)
  + Codex CLI (migrated ~/.codex -> ~/.codex-accounts/home/)

Next: uso add-profile <name>
```

Your original config directories are backed up as `~/.claude.bak` and `~/.codex.bak`.

### 2. Create profiles

```sh
uso add-profile work
uso add-profile personal
```

### 3. Set up API keys

Each profile carries its own env vars. They're exported when you switch.

```sh
uso set work ANTHROPIC_API_KEY "sk-ant-..."
uso set work OPENAI_API_KEY "sk-proj-..."

uso set personal ANTHROPIC_API_KEY "sk-ant-..."
uso set personal OPENAI_API_KEY "sk-proj-..."
```

Any env var works:

```sh
uso set work AWS_PROFILE "company-prod"
uso set work NODE_ENV "development"
```

### 4. Set terminal colors (optional)

So you always know which profile is active:

```sh
uso set work USO_COLOR teal
uso set work USO_TITLE "work"

uso set personal USO_COLOR purple
uso set personal USO_TITLE "personal"
```

Named colors: `red`, `orange`, `yellow`, `green`, `teal`, `blue`, `indigo`, `purple`, `pink`, `gray`, `white`. Or use RGB directly: `uso set work USO_COLOR "0,168,120"`.

`USO_COLOR` sets the iTerm2 tab color. `USO_TITLE` sets the terminal title.

### 5. Switch

```sh
uso work
```

Done. All tools point to the `work` config, your API keys are exported, and your terminal tab changes color.

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
| `uso init` | Interactive setup with auto-migration |
| `uso add-profile <name>` | Create a new profile |
| `uso add-tool <name> ...` | Register a custom tool |
| `uso help` | Help |

## Adding custom tools

uso ships with Claude Code and Codex CLI. Add anything else that uses file-based config:

```sh
uso add-tool gemini GEMINI_CONFIG_DIR ~/.gemini ~/.gemini-accounts
uso add-tool cursor CURSOR_CONFIG_DIR ~/.cursor ~/.cursor-accounts
```

Then create profile directories for it:

```sh
uso add-profile work   # creates dirs for all registered tools
```

## How it works

`eval "$(command uso hook zsh)"` installs a shell function that wraps the `uso` binary. When you type `uso work`:

1. The binary reads your tools and profile config
2. Swaps each tool's symlink (e.g. `~/.claude` → `~/.claude-accounts/work`)
3. Outputs `export` and `printf` statements
4. The shell function evals them into your session

On shell startup, the hook restores your last active profile.

## Profile names

Lowercase letters, numbers, and hyphens. Max 32 characters.

## Hooks

For custom logic, create an executable script at `~/.config/uso/post-switch`. It receives the profile name as `$1`.

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
├── home/
├── work/
└── personal/

~/.codex-accounts/
├── home/
├── work/
└── personal/
```

## License

MIT
