#!/usr/bin/env python3
import sys

# Configuration
MAX_LINES = 500  # Adjust this threshold as needed

# Process each file
for filename in sys.argv[1:]:
    try:
        with open(filename, 'r', encoding='utf-8') as f:
            line_count = sum(1 for line in f if line.strip())  # Count non-empty lines

        if line_count > MAX_LINES:
            print(f"[WARNING] File '{filename}' exceeds {MAX_LINES} lines ({line_count} lines). Consider refactoring.", file=sys.stderr)
    except Exception as e:
        print(f"[ERROR] Failed to process {filename}: {e}", file=sys.stderr)

# Always exit with 0 to allow the commit to proceed
sys.exit(0)
