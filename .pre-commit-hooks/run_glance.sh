#!/bin/bash
# Post-commit hook to run glance on the repository
# This generates or updates glance.md files that provide directory overviews

set -e

# Source shell profile to get proper PATH
if [ -f "$HOME/.profile" ]; then
  source "$HOME/.profile"
elif [ -f "$HOME/.bash_profile" ]; then
  source "$HOME/.bash_profile"
elif [ -f "$HOME/.zshrc" ]; then
  source "$HOME/.zshrc"
fi

# Add common Go paths to PATH if not already there
export PATH="$PATH:$HOME/go/bin:/usr/local/go/bin"

echo "Running glance to update directory overviews..."
echo "PATH: $PATH"
echo "Glance location: $(command -v glance)"

# Find and use glance from PATH
glance ./
echo "Glance directory overviews updated successfully."
