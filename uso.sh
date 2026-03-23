#!/usr/bin/env bash
# uso — lightweight AI CLI tool profile switcher
# Source this in your .zshrc or .bashrc:
#   source /path/to/uso.sh

USO_DIR="${USO_DIR:-$HOME/.config/uso}"

uso() {
  case "${1:-}" in
    "")              _uso_current ;;
    list|ls)         _uso_list ;;
    status)          _uso_status ;;
    add-tool)        shift; _uso_add_tool "$@" ;;
    add-profile)     shift; _uso_add_profile "$@" ;;
    init)            _uso_init ;;
    help|--help|-h)  _uso_help ;;
    -*)              echo "uso: unknown flag '$1'" >&2; return 1 ;;
    *)               _uso_switch "$1" ;;
  esac
}

# ── Commands ──────────────────────────────────────────────────────

_uso_init() {
  mkdir -p "$USO_DIR/tools"
  [[ -f "$USO_DIR/current" ]] || echo "home" > "$USO_DIR/current"
  echo "Initialized uso at $USO_DIR"
  echo ""
  echo "Next steps:"
  echo "  uso add-tool claude CLAUDE_CONFIG_DIR ~/.claude ~/.claude-accounts"
  echo "  uso add-tool codex  CODEX_HOME        ~/.codex ~/.codex-accounts"
  echo "  uso add-profile work"
  echo "  uso work"
}

_uso_switch() {
  local profile="$1"

  if [[ ! -d "$USO_DIR/tools" ]] || ! compgen -G "$USO_DIR/tools/*.conf" >/dev/null 2>&1; then
    echo "uso: no tools configured. Run 'uso init' first." >&2
    return 1
  fi

  # Validate profile exists for every registered tool
  local tool_file tool_name
  for tool_file in "$USO_DIR/tools"/*.conf; do
    [[ -f "$tool_file" ]] || continue
    local ENV_VAR="" SYMLINK="" ACCOUNTS_DIR=""
    source "$tool_file"
    tool_name=$(basename "$tool_file" .conf)
    if [[ ! -d "$ACCOUNTS_DIR/$profile" ]]; then
      echo "uso: profile '$profile' not found for $tool_name" >&2
      echo "  expected: $ACCOUNTS_DIR/$profile/" >&2
      echo "  create it: uso add-profile $profile" >&2
      return 1
    fi
  done

  # Switch all tools
  for tool_file in "$USO_DIR/tools"/*.conf; do
    [[ -f "$tool_file" ]] || continue
    local ENV_VAR="" SYMLINK="" ACCOUNTS_DIR=""
    source "$tool_file"
    tool_name=$(basename "$tool_file" .conf)
    local profile_dir="$ACCOUNTS_DIR/$profile"

    # Safety: refuse to overwrite a real directory
    if [[ -d "$SYMLINK" && ! -L "$SYMLINK" ]]; then
      echo "uso: $SYMLINK is a real directory, not a symlink — skipping $tool_name" >&2
      echo "  migrate: mv $SYMLINK ${SYMLINK}.bak && ln -sfn $ACCOUNTS_DIR/home $SYMLINK" >&2
      continue
    fi

    export "$ENV_VAR"="$profile_dir"
    ln -sfn "$profile_dir" "$SYMLINK"
    echo "→ $tool_name: $profile  [$profile_dir]"
  done

  export USO_PROFILE="$profile"
  echo "$profile" > "$USO_DIR/current"

  # Post-switch hook
  if [[ -x "$USO_DIR/post-switch" ]]; then
    "$USO_DIR/post-switch" "$profile"
  fi
}

_uso_current() {
  if [[ -f "$USO_DIR/current" ]]; then
    cat "$USO_DIR/current"
  else
    echo "home"
  fi
}

_uso_list() {
  if [[ ! -d "$USO_DIR/tools" ]]; then
    echo "uso: not initialized. Run 'uso init' first." >&2
    return 1
  fi

  local -a profiles=()
  local tool_file
  for tool_file in "$USO_DIR/tools"/*.conf; do
    [[ -f "$tool_file" ]] || continue
    local ENV_VAR="" SYMLINK="" ACCOUNTS_DIR=""
    source "$tool_file"
    for d in "$ACCOUNTS_DIR"/*/; do
      [[ -d "$d" ]] || continue
      local name
      name=$(basename "$d")
      local found=0
      local p
      for p in "${profiles[@]}"; do
        [[ "$p" == "$name" ]] && found=1 && break
      done
      [[ $found -eq 0 ]] && profiles+=("$name")
    done
  done

  local current
  current=$(_uso_current)
  local p
  for p in "${profiles[@]}"; do
    if [[ "$p" == "$current" ]]; then
      echo "* $p"
    else
      echo "  $p"
    fi
  done
}

_uso_status() {
  local current
  current=$(_uso_current)
  echo "profile: $current"
  echo ""

  local tool_file
  for tool_file in "$USO_DIR/tools"/*.conf; do
    [[ -f "$tool_file" ]] || continue
    local ENV_VAR="" SYMLINK="" ACCOUNTS_DIR=""
    source "$tool_file"
    local tool_name
    tool_name=$(basename "$tool_file" .conf)
    local target
    target=$(readlink "$SYMLINK" 2>/dev/null || echo "(not a symlink)")
    local env_val
    env_val=$(eval echo "\$$ENV_VAR")
    echo "  $tool_name:"
    echo "    $SYMLINK → $target"
    echo "    \$$ENV_VAR=$env_val"
  done
}

_uso_add_tool() {
  local name="$1" env_var="$2" symlink="$3" accounts_dir="$4"

  if [[ -z "$name" || -z "$env_var" || -z "$symlink" || -z "$accounts_dir" ]]; then
    echo "Usage: uso add-tool <name> <env_var> <symlink_path> <accounts_dir>" >&2
    echo "" >&2
    echo "Examples:" >&2
    echo "  uso add-tool claude CLAUDE_CONFIG_DIR ~/.claude ~/.claude-accounts" >&2
    echo "  uso add-tool codex  CODEX_HOME        ~/.codex ~/.codex-accounts" >&2
    return 1
  fi

  symlink="${symlink/#\~/$HOME}"
  accounts_dir="${accounts_dir/#\~/$HOME}"

  mkdir -p "$USO_DIR/tools"
  cat > "$USO_DIR/tools/$name.conf" <<EOF
ENV_VAR=$env_var
SYMLINK=$symlink
ACCOUNTS_DIR=$accounts_dir
EOF

  mkdir -p "$accounts_dir/home"

  echo "Added tool: $name"
  echo "  env var:      \$$env_var"
  echo "  symlink:      $symlink"
  echo "  accounts dir: $accounts_dir"
}

_uso_add_profile() {
  local profile="$1"

  if [[ -z "$profile" ]]; then
    echo "Usage: uso add-profile <name>" >&2
    return 1
  fi

  if ! compgen -G "$USO_DIR/tools/*.conf" >/dev/null 2>&1; then
    echo "uso: no tools configured. Run 'uso add-tool' first." >&2
    return 1
  fi

  local tool_file
  for tool_file in "$USO_DIR/tools"/*.conf; do
    [[ -f "$tool_file" ]] || continue
    local ENV_VAR="" SYMLINK="" ACCOUNTS_DIR=""
    source "$tool_file"
    local tool_name
    tool_name=$(basename "$tool_file" .conf)
    mkdir -p "$ACCOUNTS_DIR/$profile"
    echo "  $tool_name: $ACCOUNTS_DIR/$profile/"
  done

  echo "Profile '$profile' ready"
}

_uso_help() {
  cat <<'EOF'
uso — switch AI CLI tool profiles

Usage:
  uso <profile>           Switch to a profile
  uso                     Show current profile
  uso list                List all profiles
  uso status              Show detailed status
  uso add-tool <args>     Register a CLI tool
  uso add-profile <name>  Create a new profile
  uso init                Initialize uso
  uso help                Show this help

Examples:
  uso init
  uso add-tool claude CLAUDE_CONFIG_DIR ~/.claude ~/.claude-accounts
  uso add-tool codex  CODEX_HOME        ~/.codex ~/.codex-accounts
  uso add-profile work
  uso work

Hooks:
  Create ~/.config/uso/post-switch (chmod +x) to run after every
  switch. Receives the profile name as $1.

Environment:
  USO_DIR       Config directory (default: ~/.config/uso)
  USO_PROFILE   Active profile name (set after every switch)
EOF
}

# ── Shell startup: restore last profile ───────────────────────────

_uso_restore() {
  [[ -d "$USO_DIR/tools" ]] || return 0
  local current="home"
  [[ -f "$USO_DIR/current" ]] && current=$(cat "$USO_DIR/current")
  export USO_PROFILE="$current"

  local tool_file
  for tool_file in "$USO_DIR/tools"/*.conf; do
    [[ -f "$tool_file" ]] || continue
    local ENV_VAR="" SYMLINK="" ACCOUNTS_DIR=""
    source "$tool_file"
    local profile_dir="$ACCOUNTS_DIR/$current"
    if [[ -d "$profile_dir" ]]; then
      [[ -L "$SYMLINK" ]] && ln -sfn "$profile_dir" "$SYMLINK"
      export "$ENV_VAR"="$profile_dir"
    fi
  done
}
_uso_restore

# ── Completions ───────────────────────────────────────────────────

if [[ -n "${ZSH_VERSION:-}" ]]; then
  _uso_completions() {
    local -a subcommands profiles
    subcommands=(list status add-tool add-profile init help)

    if (( ${#words[@]} == 2 )); then
      profiles=()
      local tool_file
      for tool_file in "$USO_DIR/tools"/*.conf; do
        [[ -f "$tool_file" ]] || continue
        local ENV_VAR="" SYMLINK="" ACCOUNTS_DIR=""
        source "$tool_file"
        for d in "$ACCOUNTS_DIR"/*/; do
          [[ -d "$d" ]] || continue
          profiles+=($(basename "$d"))
        done
      done
      profiles=(${(u)profiles})
      compadd -a subcommands profiles
    fi
  }
  compdef _uso_completions uso
elif [[ -n "${BASH_VERSION:-}" ]]; then
  _uso_completions_bash() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    local subcommands="list status add-tool add-profile init help"
    local profiles=""

    local tool_file
    for tool_file in "$USO_DIR/tools"/*.conf; do
      [[ -f "$tool_file" ]] || continue
      local ENV_VAR="" SYMLINK="" ACCOUNTS_DIR=""
      source "$tool_file"
      for d in "$ACCOUNTS_DIR"/*/; do
        [[ -d "$d" ]] || continue
        profiles="$profiles $(basename "$d")"
      done
    done

    COMPREPLY=($(compgen -W "$subcommands $profiles" -- "$cur"))
  }
  complete -F _uso_completions_bash uso
fi
