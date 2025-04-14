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
                if b'\0' in sample and not (b'\0\0' in sample or b'\0\n' in sample):
                    # Contains null bytes but not in a way typical for UTF-16
                    continue
        except Exception:
            continue  # If we can't check, just skip it

        # Check for UTF-16 encoding (via BOM detection)
        encoding = 'utf-8'  # Default encoding
        try:
            with open(filename, 'rb') as f:
                raw_data = f.read(4)  # Just need the first few bytes for BOM detection
                # Check for UTF-16 BOMs
                if raw_data.startswith(b'\xff\xfe') or raw_data.startswith(b'\xfe\xff'):
                    encoding = 'utf-16'
        except Exception:
            # If detection fails, stick with utf-8
            pass

        # Process text files with detected encoding
        try:
            with open(filename, 'r', encoding=encoding) as f:
                line_count = sum(1 for line in f if line.strip())  # Count non-empty lines

            if line_count > MAX_LINES:
                print(f"[WARNING] File '{filename}' exceeds {MAX_LINES} lines ({line_count} lines). Consider refactoring.", file=sys.stderr)
        except UnicodeDecodeError:
            # Skip files that can't be decoded with the detected encoding
            # This handles other encodings we don't need to specifically support
            pass
    except Exception as e:
        print(f"[ERROR] Failed to process {filename}: {e}", file=sys.stderr)

# Always exit with 0 to allow the commit to proceed
sys.exit(0)
