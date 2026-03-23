#!/bin/sh
set -e

USO_INSTALL_DIR="${USO_INSTALL_DIR:-$HOME/.local/share/uso}"

echo "Installing uso to $USO_INSTALL_DIR..."

if [ -d "$USO_INSTALL_DIR" ]; then
  echo "Updating existing installation..."
  cd "$USO_INSTALL_DIR" && git pull
else
  git clone https://github.com/bejoinka/uso.git "$USO_INSTALL_DIR"
fi

# Detect shell rc file
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
  echo "Add this to your shell rc file:"
  echo "  source \"$USO_INSTALL_DIR/uso.sh\""
fi

echo ""
echo "Done! Restart your shell, then run:"
echo "  uso init"
