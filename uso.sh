#!/usr/bin/env bash
# uso — lightweight AI CLI tool profile switcher
# Zero dependencies. Source in your shell rc:
#   source /path/to/uso.sh

USO_DIR="${USO_DIR:-$HOME/.config/uso}"

# ── Known tools (id|env_var|default_symlink|default_accounts_dir|label) ──

_USO_KNOWN_TOOLS=(
  "claude|CLAUDE_CONFIG_DIR|$HOME/.claude|$HOME/.claude-accounts|Claude Code"
  "codex|CODEX_HOME|$HOME/.codex|$HOME/.codex-accounts|Codex CLI"
)

# ── Entry point ──────────────────────────────────────────────────

uso() {
  case "${1:-}" in
    "")              _uso_current ;;
    list|ls)         _uso_list ;;
    status)          _uso_status ;;
    init)            _uso_init ;;
    add-tool)        shift; _uso_add_tool "$@" ;;
    add-profile)     shift; _uso_add_profile "$@" ;;
    set)             shift; _uso_set "$@" ;;
    show)            shift; _uso_show "$@" ;;
    help|--help|-h)  _uso_help ;;
    -*)              echo "uso: unknown flag '$1'" >&2; return 1 ;;
    *)               _uso_switch "$1" ;;
  esac
}

# ── Validation ───────────────────────────────────────────────────

_uso_valid_name() {
  [[ -n "$1" ]] && [[ "$1" =~ ^[a-z0-9]([a-z0-9-]*[a-z0-9])?$ ]] && (( ${#1} <= 32 ))
}

# ── Cross-shell helpers ──────────────────────────────────────────

_uso_read1() {
  if [[ -n "${ZSH_VERSION:-}" ]]; then
    IFS= read -rsk1 "$1"
  else
    IFS= read -rsn1 "$1"
  fi
}

_uso_read2() {
  if [[ -n "${ZSH_VERSION:-}" ]]; then
    IFS= read -rsk2 "$1"
  else
    IFS= read -rsn2 "$1"
  fi
}

# ── Interactive multi-select ─────────────────────────────────────

_uso_multiselect() {
  [[ -n "${ZSH_VERSION:-}" ]] && setopt localoptions ksharrays 2>/dev/null
  local -a labels=("$@")
  local count=${#labels[@]}
  local cursor=0
  local -a sel
  local i
  for ((i = 0; i < count; i++)); do sel[$i]=1; done

  printf "\033[?25l" # hide cursor

  local _drawn=0
  _uso_draw() {
    ((_drawn)) && printf "\033[%dA" "$count"
    _drawn=1
    local i
    for ((i = 0; i < count; i++)); do
      local arrow="  "; [[ $i -eq $cursor ]] && arrow="> "
      local box="[ ]";  [[ ${sel[$i]} -eq 1 ]] && box="[x]"
      printf "\033[2K%s%s %s\n" "$arrow" "$box" "${labels[$i]}"
    done
  }

  _uso_draw
  while true; do
    local key
    _uso_read1 key
    case "$key" in
      $'\x1b')
        _uso_read2 key
        case "$key" in
          '[A') ((cursor > 0)) && ((cursor--)) ;;
          '[B') ((cursor < count - 1)) && ((cursor++)) ;;
        esac
        ;;
      ' ') sel[$cursor]=$(( 1 - ${sel[$cursor]} )) ;;
      '')  break ;;
    esac
    _uso_draw
  done

  printf "\033[?25h" # restore cursor

  _USO_SELECTED=()
  for ((i = 0; i < count; i++)); do
    [[ ${sel[$i]} -eq 1 ]] && _USO_SELECTED+=("$i")
  done
}

# ── Commands ─────────────────────────────────────────────────────

_uso_init() {
  [[ -n "${ZSH_VERSION:-}" ]] && setopt localoptions ksharrays 2>/dev/null
  mkdir -p "$USO_DIR"/{tools,profiles}
  [[ -f "$USO_DIR/current" ]] || echo "home" > "$USO_DIR/current"

  if [[ -t 0 && -t 1 ]]; then
    echo "Select tools to manage (arrows move, space toggles, enter confirms):"
    echo ""

    local -a labels=()
    local entry
    for entry in "${_USO_KNOWN_TOOLS[@]}"; do
      labels+=("${entry##*|}")
    done

    _uso_multiselect "${labels[@]}"
    echo ""

    local idx
    for idx in "${_USO_SELECTED[@]}"; do
      IFS='|' read -r id env_var symlink accounts_dir label <<< "${_USO_KNOWN_TOOLS[$idx]}"
      cat > "$USO_DIR/tools/$id.conf" <<CONF
ENV_VAR=$env_var
SYMLINK=$symlink
ACCOUNTS_DIR=$accounts_dir
CONF
      mkdir -p "$accounts_dir/home"
      echo "  + $label"
    done
  else
    # Non-interactive: register all
    local entry
    for entry in "${_USO_KNOWN_TOOLS[@]}"; do
      IFS='|' read -r id env_var symlink accounts_dir label <<< "$entry"
      cat > "$USO_DIR/tools/$id.conf" <<CONF
ENV_VAR=$env_var
SYMLINK=$symlink
ACCOUNTS_DIR=$accounts_dir
CONF
      mkdir -p "$accounts_dir/home"
      echo "  + $label"
    done
  fi

  echo ""
  echo "Next: uso add-profile <name>"
}

_uso_switch() {
  local profile="$1"

  # Must be a registered profile (or "home")
  if [[ "$profile" != "home" && ! -f "$USO_DIR/profiles/$profile.conf" ]]; then
    echo "uso: unknown profile '$profile'" >&2
    local registered
    registered=$(cd "$USO_DIR/profiles" 2>/dev/null && ls *.conf 2>/dev/null | sed 's/\.conf$//' | tr '\n' ' ')
    [[ -n "$registered" ]] && echo "  profiles: home $registered" >&2
    return 1
  fi

  if ! compgen -G "$USO_DIR/tools/*.conf" >/dev/null 2>&1; then
    echo "uso: no tools configured. Run 'uso init'." >&2
    return 1
  fi

  # Clear previous profile env vars
  local prev_profile
  prev_profile=$(_uso_current)
  if [[ -f "$USO_DIR/profiles/$prev_profile.conf" ]]; then
    while IFS='=' read -r key _; do
      [[ -z "$key" || "$key" =~ ^[[:space:]]*# ]] && continue
      key="${key## }"; key="${key%% }"
      unset "$key" 2>/dev/null
    done < "$USO_DIR/profiles/$prev_profile.conf"
  fi

  # Switch tool symlinks + env vars
  local tool_file tool_name
  for tool_file in "$USO_DIR/tools"/*.conf; do
    [[ -f "$tool_file" ]] || continue
    local ENV_VAR="" SYMLINK="" ACCOUNTS_DIR=""
    source "$tool_file"
    tool_name=$(basename "$tool_file" .conf)
    local profile_dir="$ACCOUNTS_DIR/$profile"

    if [[ ! -d "$profile_dir" ]]; then
      echo "  $tool_name: no dir at $profile_dir — skipping" >&2
      continue
    fi

    if [[ -d "$SYMLINK" && ! -L "$SYMLINK" ]]; then
      echo "  $tool_name: $SYMLINK is a real directory — skipping" >&2
      echo "    fix: mv $SYMLINK ${SYMLINK}.bak && ln -sfn $ACCOUNTS_DIR/home $SYMLINK" >&2
      continue
    fi

    export "$ENV_VAR"="$profile_dir"
    ln -sfn "$profile_dir" "$SYMLINK"
    echo "-> $tool_name: $profile  [$profile_dir]"
  done

  # Export profile env vars
  if [[ -f "$USO_DIR/profiles/$profile.conf" ]]; then
    while IFS='=' read -r key value; do
      [[ -z "$key" || "$key" =~ ^[[:space:]]*# ]] && continue
      key="${key## }"; key="${key%% }"
      value="${value## }"; value="${value%% }"
      export "$key"="$value"
    done < "$USO_DIR/profiles/$profile.conf"
  fi

  export USO_PROFILE="$profile"
  echo "$profile" > "$USO_DIR/current"

  # Built-in: iTerm tab color
  if [[ -n "${USO_COLOR:-}" ]]; then
    IFS=',' read -r r g b <<< "$USO_COLOR"
    printf "\033]6;1;bg;red;brightness;%d\a"   "$r"
    printf "\033]6;1;bg;green;brightness;%d\a" "$g"
    printf "\033]6;1;bg;blue;brightness;%d\a"  "$b"
  fi

  # Built-in: terminal title
  [[ -n "${USO_TITLE:-}" ]] && printf "\033]0;%s\a" "$USO_TITLE"

  # Post-switch hook
  [[ -x "$USO_DIR/post-switch" ]] && "$USO_DIR/post-switch" "$profile"
}

_uso_current() {
  [[ -f "$USO_DIR/current" ]] && cat "$USO_DIR/current" || echo "home"
}

_uso_list() {
  [[ -d "$USO_DIR/profiles" ]] || { echo "uso: run 'uso init' first." >&2; return 1; }

  local current
  current=$(_uso_current)

  # Always show home
  [[ "$current" == "home" ]] && echo "* home" || echo "  home"

  local f
  for f in "$USO_DIR/profiles"/*.conf; do
    [[ -f "$f" ]] || continue
    local name
    name=$(basename "$f" .conf)
    [[ "$name" == "$current" ]] && echo "* $name" || echo "  $name"
  done
}

_uso_status() {
  local current
  current=$(_uso_current)
  echo "profile: $current"
  echo ""

  if [[ -f "$USO_DIR/profiles/$current.conf" ]]; then
    local has=0
    while IFS='=' read -r key value; do
      [[ -z "$key" || "$key" =~ ^[[:space:]]*# ]] && continue
      ((has)) || echo "  env:"
      has=1
      echo "    $key=$value"
    done < "$USO_DIR/profiles/$current.conf"
    ((has)) && echo ""
  fi

  echo "  tools:"
  local tool_file
  for tool_file in "$USO_DIR/tools"/*.conf; do
    [[ -f "$tool_file" ]] || continue
    local ENV_VAR="" SYMLINK="" ACCOUNTS_DIR=""
    source "$tool_file"
    local tool_name target
    tool_name=$(basename "$tool_file" .conf)
    target=$(readlink "$SYMLINK" 2>/dev/null || echo "(not a symlink)")
    echo "    $tool_name: $SYMLINK -> $target"
  done
}

_uso_add_tool() {
  local name="$1" env_var="$2" symlink="$3" accounts_dir="$4"

  if [[ -z "$name" || -z "$env_var" || -z "$symlink" || -z "$accounts_dir" ]]; then
    echo "Usage: uso add-tool <name> <env_var> <symlink> <accounts_dir>" >&2
    echo "" >&2
    echo "Examples:" >&2
    echo "  uso add-tool claude CLAUDE_CONFIG_DIR ~/.claude ~/.claude-accounts" >&2
    echo "  uso add-tool codex  CODEX_HOME        ~/.codex ~/.codex-accounts" >&2
    return 1
  fi

  if ! _uso_valid_name "$name"; then
    echo "uso: invalid name '$name' (lowercase alphanumeric + hyphens, max 32)" >&2
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
  echo "Added tool: $name (\$$env_var -> $symlink)"
}

_uso_add_profile() {
  local profile="$1"

  if [[ -z "$profile" ]]; then
    echo "Usage: uso add-profile <name>" >&2
    return 1
  fi

  if ! _uso_valid_name "$profile"; then
    echo "uso: invalid name '$profile' (lowercase alphanumeric + hyphens, max 32)" >&2
    return 1
  fi

  mkdir -p "$USO_DIR/profiles"
  touch "$USO_DIR/profiles/$profile.conf"

  # Create tool directories
  if compgen -G "$USO_DIR/tools/*.conf" >/dev/null 2>&1; then
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
  fi

  echo ""
  echo "Profile '$profile' ready. Configure:"
  echo "  uso set $profile USO_TITLE \"$profile\""
  echo "  uso set $profile USO_COLOR \"R,G,B\""
  echo "  uso set $profile ANY_ENV_VAR \"value\""
}

_uso_set() {
  local profile="$1" key="$2"
  shift 2 2>/dev/null
  local value="$*"

  if [[ -z "$profile" || -z "$key" ]]; then
    echo "Usage: uso set <profile> <KEY> [value]" >&2
    echo "" >&2
    echo "Special keys (applied on switch):" >&2
    echo "  USO_COLOR   iTerm2 tab color as R,G,B" >&2
    echo "  USO_TITLE   Terminal title" >&2
    echo "" >&2
    echo "All other keys are exported as env vars." >&2
    echo "Omit value to unset a key." >&2
    return 1
  fi

  if [[ "$profile" != "home" && ! -f "$USO_DIR/profiles/$profile.conf" ]]; then
    echo "uso: profile '$profile' not found" >&2
    return 1
  fi

  # Ensure home profile conf exists
  [[ "$profile" == "home" ]] && { mkdir -p "$USO_DIR/profiles"; touch "$USO_DIR/profiles/$profile.conf"; }

  local conf="$USO_DIR/profiles/$profile.conf"

  # Remove existing key
  local tmp
  tmp=$(grep -v "^${key}=" "$conf" 2>/dev/null || true)
  # Strip trailing blank lines
  tmp=$(echo "$tmp" | sed -e :a -e '/^[[:space:]]*$/{ $d; N; ba; }')

  if [[ -z "$value" ]]; then
    echo "$tmp" > "$conf"
    echo "Removed $key from $profile"
  else
    echo "$tmp" > "$conf"
    echo "${key}=${value}" >> "$conf"
    echo "$profile: $key=$value"
  fi
}

_uso_show() {
  local profile="${1:-$(_uso_current)}"

  if [[ "$profile" != "home" && ! -f "$USO_DIR/profiles/$profile.conf" ]]; then
    echo "uso: profile '$profile' not found" >&2
    return 1
  fi

  echo "profile: $profile"
  if [[ -f "$USO_DIR/profiles/$profile.conf" ]]; then
    local has=0
    while IFS='=' read -r key value; do
      [[ -z "$key" || "$key" =~ ^[[:space:]]*# ]] && continue
      echo "  $key=$value"
      has=1
    done < "$USO_DIR/profiles/$profile.conf"
    ((has)) || echo "  (no config)"
  else
    echo "  (no config)"
  fi
}

_uso_help() {
  cat <<'EOF'
uso — switch AI CLI tool profiles

Commands:
  uso <profile>                Switch to a profile
  uso                          Show current profile
  uso list                     List profiles
  uso status                   Detailed status
  uso init                     Interactive setup
  uso add-tool  <n> <e> <s> <d>  Register a CLI tool
  uso add-profile <name>       Create a profile
  uso set <profile> <K> [V]    Set/unset env var or config
  uso show [profile]           Show profile config
  uso help                     This help

Examples:
  uso init
  uso add-profile work
  uso set work USO_COLOR "0,168,120"
  uso set work USO_TITLE "work"
  uso set work MY_API_KEY "sk-..."
  uso work

Profile names: lowercase, digits, hyphens. Max 32 chars.

Special keys (applied automatically on switch):
  USO_COLOR    iTerm2 tab color as R,G,B (0-255)
  USO_TITLE    Terminal title string

All other keys are exported as environment variables.

Environment:
  USO_DIR       Config location (default: ~/.config/uso)
  USO_PROFILE   Active profile (set on every switch)
EOF
}

# ── Shell startup: restore last profile ──────────────────────────

_uso_restore() {
  [[ -d "$USO_DIR/tools" ]] || return 0
  local current="home"
  [[ -f "$USO_DIR/current" ]] && current=$(cat "$USO_DIR/current")
  export USO_PROFILE="$current"

  # Restore tool symlinks + env vars
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

  # Restore profile env vars
  if [[ -f "$USO_DIR/profiles/$current.conf" ]]; then
    while IFS='=' read -r key value; do
      [[ -z "$key" || "$key" =~ ^[[:space:]]*# ]] && continue
      key="${key## }"; key="${key%% }"
      value="${value## }"; value="${value%% }"
      export "$key"="$value"
    done < "$USO_DIR/profiles/$current.conf"
  fi

  # Apply visual effects silently on restore
  if [[ -n "${USO_COLOR:-}" ]]; then
    IFS=',' read -r r g b <<< "$USO_COLOR"
    printf "\033]6;1;bg;red;brightness;%d\a"   "$r"
    printf "\033]6;1;bg;green;brightness;%d\a" "$g"
    printf "\033]6;1;bg;blue;brightness;%d\a"  "$b"
  fi
  [[ -n "${USO_TITLE:-}" ]] && printf "\033]0;%s\a" "$USO_TITLE"
}
_uso_restore

# ── Completions ──────────────────────────────────────────────────

if [[ -n "${ZSH_VERSION:-}" ]]; then
  _uso_completions() {
    local -a subcmds profiles
    subcmds=(list status init add-tool add-profile set show help)
    profiles=(home)

    local f
    for f in "$USO_DIR/profiles"/*.conf(N); do
      profiles+=("$(basename "$f" .conf)")
    done

    if (( CURRENT == 2 )); then
      compadd -a subcmds profiles
    elif (( CURRENT == 3 )) && [[ "${words[2]}" == (set|show) ]]; then
      compadd -a profiles
    fi
  }
  compdef _uso_completions uso 2>/dev/null
elif [[ -n "${BASH_VERSION:-}" ]]; then
  _uso_completions_bash() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    local subcmds="list status init add-tool add-profile set show help"
    local profiles="home"

    local f
    for f in "$USO_DIR/profiles"/*.conf; do
      [[ -f "$f" ]] && profiles="$profiles $(basename "$f" .conf)"
    done

    if [[ $COMP_CWORD -eq 1 ]]; then
      COMPREPLY=($(compgen -W "$subcmds $profiles" -- "$cur"))
    elif [[ $COMP_CWORD -eq 2 ]] && [[ "${COMP_WORDS[1]}" == "set" || "${COMP_WORDS[1]}" == "show" ]]; then
      COMPREPLY=($(compgen -W "$profiles" -- "$cur"))
    fi
  }
  complete -F _uso_completions_bash uso
fi
