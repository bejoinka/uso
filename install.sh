#!/bin/sh
set -e

USO_INSTALL_DIR="${USO_INSTALL_DIR:-$HOME/.local/share/uso}"

echo "Installing uso to $USO_INSTALL_DIR..."

if [ -d "$USO_INSTALL_DIR/.git" ]; then
  cd "$USO_INSTALL_DIR" && git pull --quiet
  echo "Updated."
else
  rm -rf "$USO_INSTALL_DIR"
  git clone --quiet https://github.com/bejoinka/uso.git "$USO_INSTALL_DIR"
  echo "Cloned."
fi

# Detect shell rc
SHELL_RC=""
if [ -f "$HOME/.zshrc" ]; then
  SHELL_RC="$HOME/.zshrc"
elif [ -f "$HOME/.bashrc" ]; then
  SHELL_RC="$HOME/.bashrc"
fi

if [ -n "$SHELL_RC" ]; then
  if ! grep -q 'uso/uso.sh' "$SHELL_RC" 2>/dev/null; then
    printf '\n# uso — AI CLI tool profile switcher\nsource "%s/uso.sh"\n' "$USO_INSTALL_DIR" >> "$SHELL_RC"
    echo "Added source line to $SHELL_RC"
  else
    echo "Already sourced in $SHELL_RC"
  fi
else
  echo "Add to your shell rc:"
  echo "  source \"$USO_INSTALL_DIR/uso.sh\""
fi

echo ""
echo "Restart your shell, then run: uso init"
