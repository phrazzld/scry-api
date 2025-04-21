#!/bin/bash
# Post-commit hook to run glance on the repository
# This generates or updates glance.md files that provide directory overviews
# Modified to work better in background execution

# Disable error exit to avoid background process crashing
# set -e

# Create a log file for debugging if needed
LOG_FILE="/tmp/glance_post_commit_$(date +%s).log"

{
  echo "Starting glance post-commit hook at $(date)"

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

  # Check if glance is available
  if command -v glance &> /dev/null; then
    echo "Glance location: $(command -v glance)"

    # Run glance and capture its output
    if glance ./; then
      echo "Glance directory overviews updated successfully."
    else
      echo "Glance failed with exit code $?"
    fi
  else
    echo "ERROR: Glance command not found in PATH"
  fi

  echo "Completed glance post-commit hook at $(date)"
} > "$LOG_FILE" 2>&1
