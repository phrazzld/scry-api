#!/usr/bin/env python3
import sys
import os
import re
import fnmatch

# Configuration
# Files to exclude from panic check (test files and specific exceptions)
EXCLUDED_PATTERNS = [
    '*_test.go',              # Test files
    'testutils/**',           # Test utility files
    '**/mocks/**',            # Mock implementations
    'cmd/server/main.go',     # Main entry point that handles startup panics
]

# Regular expressions to identify panic calls
PANIC_PATTERNS = [
    r'\bpanic\([^)]*\)',      # Direct panic calls
]

# Exempt patterns - these are acceptable panic contexts
EXEMPT_PATTERNS = [
    r'//\s*ALLOW[\-_]PANIC',   # Line with comment // ALLOW-PANIC or // ALLOW_PANIC
    r'//\s*lint:allow panic',  # Line with comment // lint:allow panic
    r'/\*.*ALLOW[\-_]PANIC.*\*/', # Comment block with ALLOW-PANIC
    r'func init\(\)',          # Init functions can have panics for things that must succeed
    r'^\s*if\s+testing\.',     # Test conditions like "if testing.Short()"
]

def is_excluded(filename):
    """Check if a file matches any excluded pattern."""
    for pattern in EXCLUDED_PATTERNS:
        if fnmatch.fnmatch(filename, pattern):
            return True
    return False

def has_exemption(line, prev_line=None):
    """Check if the line has an exemption comment."""
    for pattern in EXEMPT_PATTERNS:
        if re.search(pattern, line):
            return True
        if prev_line and re.search(pattern, prev_line):
            return True
    return False

# Process each file
exit_code = 0  # Start with success

for filename in sys.argv[1:]:
    # Skip non-Go files and excluded files
    if not filename.endswith('.go') or is_excluded(filename):
        continue

    try:
        with open(filename, 'r', encoding='utf-8') as f:
            lines = f.readlines()

        # Check each line for panic calls
        for i, line in enumerate(lines):
            prev_line = lines[i-1] if i > 0 else ""

            # Look for panic patterns
            for pattern in PANIC_PATTERNS:
                if re.search(pattern, line):
                    # Check if there's an exemption
                    if not has_exemption(line, prev_line):
                        relative_path = os.path.relpath(filename)
                        print(f"[ERROR] {relative_path}:{i+1}: Direct use of panic() detected.", file=sys.stderr)
                        print(f"    {line.strip()}", file=sys.stderr)
                        print(f"    Prefer returning errors instead of panic in production code.", file=sys.stderr)
                        print(f"    Add // ALLOW-PANIC comment to exempt this line if panic is necessary.", file=sys.stderr)
                        print(f"", file=sys.stderr)
                        exit_code = 1
    except Exception as e:
        print(f"[ERROR] Failed to process {filename}: {e}", file=sys.stderr)

# Exit with the determined exit code
sys.exit(exit_code)
