#!/usr/bin/env python3
import sys
import os

# Configuration
MAX_LINES = 500  # Adjust this threshold as needed

# Process each file
for filename in sys.argv[1:]:
    try:
        # Skip binary files - quick check if file appears to be binary
        try:
            with open(filename, 'rb') as f:
                sample = f.read(1024)
                if b'\0' in sample:  # Simple binary check: contains null bytes
                    continue
        except Exception:
            continue  # If we can't check, just skip it

        # Process text files
        with open(filename, 'r', encoding='utf-8') as f:
            line_count = sum(1 for line in f if line.strip())  # Count non-empty lines

        if line_count > MAX_LINES:
            print(f"[WARNING] File '{filename}' exceeds {MAX_LINES} lines ({line_count} lines). Consider refactoring.", file=sys.stderr)
    except UnicodeDecodeError:
        # Skip files that can't be decoded as UTF-8 (likely binary)
        continue
    except Exception as e:
        print(f"[ERROR] Failed to process {filename}: {e}", file=sys.stderr)

# Always exit with 0 to allow the commit to proceed
sys.exit(0)
