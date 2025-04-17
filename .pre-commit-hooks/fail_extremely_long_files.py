#!/usr/bin/env python3
import sys
import os

# Configuration
FAIL_LINES_THRESHOLD = 1000  # Fail the commit if file exceeds this length

# Process each file
exit_code = 0  # Start with success, will be set to 1 if any files exceed limit

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

            if line_count > FAIL_LINES_THRESHOLD:
                print(f"[ERROR] File '{filename}' exceeds {FAIL_LINES_THRESHOLD} lines ({line_count} lines). Commits with extremely long files are not allowed.", file=sys.stderr)
                exit_code = 1  # Set exit code to failure
        except UnicodeDecodeError:
            # Skip files that can't be decoded with the detected encoding
            pass
    except Exception as e:
        print(f"[ERROR] Failed to process {filename}: {e}", file=sys.stderr)

# Exit with the determined exit code - fails if any file exceeds limit
sys.exit(exit_code)
