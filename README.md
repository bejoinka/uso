# uso

Switch every AI CLI tool to a different account with one command.

```
$ uso work
-> claude: work  [~/.claude-accounts/work]
-> codex:  work  [~/.codex-accounts/work]

$ uso personal
-> claude: personal  [~/.claude-accounts/personal]
-> codex:  personal  [~/.codex-accounts/personal]
```

If you use Claude Code or Codex CLI with more than one account (a day job, a client, a side project), each account needs its own login, settings, MCP servers and API keys. uso keeps a separate copy of each tool's config per account and switches all of them at once.

## How it works

A **profile** is a name, such as `work`, that stands for two things:

- **One config directory per tool.** `~/.claude-accounts/work` for Claude Code and `~/.codex-accounts/work` for Codex.
- **A list of environment variables.** API keys, `AWS_PROFILE`, or anything else.

When you run `uso work`, uso:

1. Points `~/.claude` and `~/.codex` at the `work` directories, and exports `CLAUDE_CONFIG_DIR` and `CODEX_HOME`.
2. Unsets the previous profile's environment variables and exports the `work` ones.

To Claude Code and Codex, each profile looks like a separate install. Each profile has its own login, settings, MCP servers, history and `CLAUDE.md`.

## What uso changes on your machine

Read this before you install:

- **`uso init` moves your existing config.** It copies `~/.claude` to `~/.claude-accounts/home/`, renames the original to `~/.claude.bak`, and puts a symlink at `~/.claude`. It does the same for `~/.codex`. If a `~/.claude.bak` or `~/.codex.bak` already exists, uso replaces it. It also copies `~/.claude.json` into `~/.claude-accounts/home/` and leaves the original in place.
- **It creates `~/.config/uso/`** to hold profiles and tool settings.
- **You add one line to `~/.zshrc` or `~/.bashrc`.** It defines a shell function called `uso` and restores your last profile in each new terminal.

To undo all of this, see [Uninstall](#uninstall).

## Install

**Homebrew:**

```sh
brew tap bejoinka/tap
brew install uso
```

**Or with Go** (1.26 or later):

```sh
go install github.com/bejoinka/uso@latest
```

Then add this line to your `~/.zshrc`. If you use oh-my-zsh or call `compinit`, put it after that, so that tab completion works:

```sh
eval "$(command uso hook zsh)"
```

For bash, add `eval "$(command uso hook bash)"` to `~/.bashrc` instead.

Open a new terminal and check that the hook is loaded:

```sh
$ type uso
uso is a shell function from zsh
```

If the output says `function`, the hook is loaded. If it shows a path such as `/opt/homebrew/bin/uso`, it isn't, and uso can't change your shell's environment.

## Quick start

### 1. Run `uso init`

```sh
uso init
```

Choose your tools, then name your **default profile**, the one that will hold your current setup. Press Enter to call it `home`:

```
Select tools to manage:
> [x] Claude Code
  [x] Codex CLI

  arrows move, space toggles, enter confirms

Default profile name [home]:

  + Claude Code (migrated ~/.claude -> ~/.claude-accounts/home/)
    copied ~/.claude.json into ~/.claude-accounts/home/
  + Codex CLI (migrated ~/.codex -> ~/.codex-accounts/home/)

Your existing setup is now the 'home' profile, your default. `uso list` marks it.
Next: uso add-profile <name>
```

#### What carries over into your default profile

Your default profile starts as a copy of your setup from before uso:

- **Everything in `~/.claude`:** settings, `CLAUDE.md`, agents, commands, plugins and history.
- **`~/.claude.json`:** your account, user-level MCP servers and per-project state. Claude Code keeps this file next to the `~/.claude` folder, not inside it. Once `CLAUDE_CONFIG_DIR` is set, Claude Code reads it from the profile, so uso copies it there.
- **Everything in `~/.codex`,** including your Codex login.

Depending on where your system stores the Claude Code login, Claude Code may ask you to log in once more in the default profile.

Two things don't move:

- **Environment variables in your rc file,** such as an `export ANTHROPIC_API_KEY=...` line in `~/.zshrc`, stay set in every profile. To tie one to your default profile only, remove it from the rc file and run `uso set home ANTHROPIC_API_KEY "..."`.
- **`.claude/` folders and `CLAUDE.md` files inside your repositories.** They don't belong to any profile and apply in all of them.

To check which profile is your default later, run `uso list`:

```
$ uso list
  home (default)
* work
```

### 2. Create a profile

```sh
uso add-profile work
```

```
  claude: ~/.claude-accounts/work/
  codex: ~/.codex-accounts/work/

Profile 'work' ready. Configure:
  uso set work USO_TITLE "work"
  uso set work USO_COLOR blue    # or: red, orange, teal, purple, 0,168,120
```

### 3. Switch to it and log in

```sh
uso work
claude        # then run /login and sign in with your work account
codex login
```

**A new profile starts empty.** It has no login, no settings, no MCP servers and no `CLAUDE.md`. Log in once per profile, and uso keeps that login in the profile's directory.

To carry settings over from another profile, copy the files:

```sh
cp ~/.claude-accounts/home/settings.json ~/.claude-accounts/work/
cp ~/.claude-accounts/home/CLAUDE.md     ~/.claude-accounts/work/
```

To keep a profile's settings, skills, plugins and memory in step with another's instead, share them. See [Sharing between profiles](#sharing-between-profiles).

### 4. Add environment variables (optional)

Each profile can export its own variables when you switch to it:

```sh
uso set work AWS_PROFILE company-prod
uso set work GH_TOKEN "ghp_..."
```

You can store API keys the same way:

```sh
uso set work ANTHROPIC_API_KEY "sk-ant-..."
uso set work OPENAI_API_KEY "sk-proj-..."
```

When `ANTHROPIC_API_KEY` is set, Claude Code prefers it over your logged-in account. Only set it on profiles that should bill to an API key.

uso stores these values in plain text in `~/.config/uso/profiles/<profile>.conf`, and `uso status` and `uso show` print them.

### 5. Color-code your terminal (optional)

To see at a glance which profile a terminal is using, set a tab color and a title:

```sh
uso set work USO_COLOR teal
uso set work USO_TITLE "work"

uso set personal USO_COLOR purple
uso set personal USO_TITLE "personal"
```

Named colors: `red`, `orange`, `yellow`, `green`, `teal`, `blue`, `indigo`, `purple`, `pink`, `gray`, `white`. You can also give an RGB value, for example `uso set work USO_COLOR "0,168,120"`.

`USO_COLOR` sets the tab color in iTerm2 and has no effect in other terminals. `USO_TITLE` sets the window title in most terminals.

## Day to day

| Command | What it does |
|---------|-------------|
| `uso <profile>` | Switch to a profile |
| `uso` | Print the profile you switched to most recently, in any terminal |
| `uso list` (or `uso ls`) | List profiles. `*` marks the most recent one, `(default)` your default profile, and `(shares awell)` a profile that shares with awell |
| `uso status` | Show the current profile, its variables, and where each tool's symlink points |
| `uso show [profile]` | Show a profile's variables. Defaults to the current profile |
| `uso set <profile> <KEY> <value>` | Set a variable on a profile |
| `uso set <profile> <KEY>` | Remove a variable from a profile |
| `uso add-profile <name>` | Create a profile. Safe to rerun on an existing profile |
| `uso add-profile <name> --share <profile>` | Create or convert a profile that shares everything but the login with another. See [Sharing between profiles](#sharing-between-profiles) |
| `uso add-tool <name> <ENV_VAR> <default-dir> <accounts-dir>` | Register another tool. See [Adding other tools](#adding-other-tools) |
| `uso init` | Run the interactive setup |
| `uso hook zsh` / `uso hook bash` | Print the shell hook. Used by the line in your rc file |

Profile names use lowercase letters, numbers and hyphens, up to 32 characters.

To find out which profile the current terminal is using, run `echo $USO_PROFILE`.

To delete a profile, remove its files. uso has no command for this yet.

```sh
rm ~/.config/uso/profiles/work.conf
rm -rf ~/.claude-accounts/work ~/.codex-accounts/work
```

### What's shared between terminals

Some of what `uso work` changes applies only to the terminal you ran it in, and some applies to every terminal:

- **This terminal only:** environment variables, including `CLAUDE_CONFIG_DIR` and `CODEX_HOME`. Switching in one terminal doesn't affect a `claude` session that is already running in another.
- **Every terminal:** the `~/.claude` and `~/.codex` symlinks, and the record of your most recent profile. A new terminal opens in the profile you switched to last. Programs that don't start from your shell, such as IDE extensions and desktop apps, follow the symlink.

Setting `USO_COLOR` and `USO_TITLE` makes it easy to tell terminals apart.

## Sharing between profiles

Two profiles can share everything except the login. For example, two accounts can work on the same repositories with one memory:

```sh
uso add-profile awell2 --share awell
```

```
  claude: ~/.claude-accounts/awell2/
    shares settings.json, agents, commands, skills, output-styles, projects, plugins with awell
```

For Claude Code, these items in `awell2` become links to the copies in `awell`:

| Shared | What it holds |
|--------|---------------|
| `settings.json`, `CLAUDE.md` | Settings, permissions, hooks, enabled plugins, and your global instructions |
| `agents/`, `commands/`, `skills/`, `output-styles/` | Your custom agents, slash commands, skills and output styles |
| `plugins/` | Installed plugins and marketplaces |
| `projects/` | Memory and session history for each repository. `claude --resume` lists sessions from both profiles |

Each profile keeps its own login and `.claude.json`, which holds the account, user-level MCP servers and trusted folders. It also keeps its organization's managed settings and its runtime state. Log in once in the new profile, and add user-level MCP servers in each profile.

Claude Code keeps memory per repository, in `projects/<repository path>/memory`. Different repositories never share memory, whether or not their profiles share.

`uso list` shows `awell2 (shares awell)`. Running the command again is safe.

### Converting a profile you already use

Run the same command on an existing profile. Close Claude Code in both profiles first. The profile's own files move into the other profile:

- **`agents/`, `commands/`, `skills/`, `output-styles/` and `projects/`** are merged, so both profiles' sessions and memory are kept.
- **`settings.json`, `CLAUDE.md` and `plugins/`** are not merged: the other profile's copy is kept whole.
- **Anything only this profile has** moves to the other profile.
- **Where the two profiles differ,** the other profile's version wins. This profile's version moves to a `uso-backup-*` folder inside this profile, and the output lists each one. Nothing is deleted except identical files and links that point to the same place.

### Notes

- Only Claude Code has shared items so far. Other tools stay separate per profile.
- Don't delete a profile that others share from (`awell` above). Their links would break.
- To stop sharing an item, replace the link with a copy:

  ```sh
  rm ~/.claude-accounts/awell2/skills
  cp -R ~/.claude-accounts/awell/skills ~/.claude-accounts/awell2/
  ```

## Adding other tools

uso ships with Claude Code and Codex CLI. You can add any tool that keeps its config in a single directory in your home folder.

`uso add-tool` doesn't move an existing config directory for you, so move it first. For example, for Gemini CLI:

```sh
# 1. Move your existing config into your default profile (home here)
mkdir -p ~/.gemini-accounts
mv ~/.gemini ~/.gemini-accounts/home
ln -s ~/.gemini-accounts/home ~/.gemini

# 2. Register the tool
uso add-tool gemini GEMINI_CONFIG_DIR ~/.gemini ~/.gemini-accounts
```

`uso add-tool` creates an empty directory for the tool in each of your other profiles.

The second argument names the environment variable that uso exports for the tool. If the tool doesn't read that variable, the symlink still switches it, but the switch applies to every terminal, not only the current one. Check the tool's documentation for the variable it uses, if it has one.

## Troubleshooting

| You see | What it means |
|---------|---------------|
| `Error: unknown command "work" for "uso"` | The shell hook isn't loaded. Check the line in your rc file, then open a new terminal and run `type uso`. |
| `uso: no tools configured. Run 'uso init'.` | Run `uso init`. |
| `uso: unknown profile 'x'` | Create the profile with `uso add-profile x`. |
| `<tool>: no dir at ~/.<tool>-accounts/<profile> — skipping` | The tool has no directory for that profile. Run `uso add-profile <profile>` to create it. |
| `<tool>: ~/.<tool> is a real directory — skipping` | The tool's config is a real folder, not a symlink. Move it into the accounts directory as shown in [Adding other tools](#adding-other-tools). |
| Your MCP servers are gone after `uso init` | Older versions of `uso init` didn't copy `~/.claude.json`. `cp -n ~/.claude.json ~/.claude-accounts/home/` copies it if the profile doesn't have one yet. |

## Under the hood

The line in your rc file defines a shell function named `uso`. When you type `uso work`:

1. The function runs the `uso` binary.
2. The binary repoints each tool's symlink at the profile directory, for example `~/.claude` → `~/.claude-accounts/work`.
3. The binary prints `export` and `printf` statements.
4. The function runs those statements in your current shell.

A program can't change its parent shell's environment, which is why uso needs the function. When you open a new terminal, the hook restores the profile you used last.

To run your own commands after each switch, create an executable script at `~/.config/uso/post-switch`. uso passes it the profile name as `$1`.

### File layout

```
~/.config/uso/
├── current                    # the profile you switched to last
├── default                    # your default profile, set by uso init
├── post-switch                # optional script, run after each switch
├── tools/
│   ├── claude.conf            # ENV_VAR, SYMLINK, ACCOUNTS_DIR
│   └── codex.conf
└── profiles/
    ├── work.conf              # KEY=VALUE pairs, exported on switch
    └── personal.conf

~/.claude -> ~/.claude-accounts/work
~/.claude-accounts/
├── home/
├── work/
└── personal/

~/.codex -> ~/.codex-accounts/work
~/.codex-accounts/
├── home/
├── work/
└── personal/
```

## Uninstall

1. Remove the `eval "$(command uso hook ...)"` line from your rc file and open a new terminal.
2. Turn each tool's symlink back into a real directory. Choose the profile you want to keep; this example keeps `home`:

   ```sh
   rm ~/.claude && mv ~/.claude-accounts/home ~/.claude    # rm removes the symlink only
   rm ~/.codex  && mv ~/.codex-accounts/home  ~/.codex
   ```

   Without uso, Claude Code reads `~/.claude.json` again, not the copy inside the profile. To keep the profile's login and MCP servers, copy it back with `cp ~/.claude/.claude.json ~/.claude.json`.

3. Remove uso's config and the binary:

   ```sh
   rm -rf ~/.config/uso
   brew uninstall uso      # or: rm "$(go env GOPATH)/bin/uso"
   ```

Your other profiles stay in `~/.claude-accounts/` and `~/.codex-accounts/` until you delete them.

## License

MIT
