#!/usr/bin/env python3
import sys
import os
import re
import fnmatch

# Configuration
# Files to exclude from SQL order check
EXCLUDED_PATTERNS = [
    '*_test.go',              # Test files may have simplified queries
    'testutils/**',           # Test utility files
    'migrations/**',          # Migration files
    'internal/platform/postgres/migrations/**',  # SQL migrations
]

# Exempt patterns - these are acceptable non-deterministic order contexts
EXEMPT_PATTERNS = [
    r'//\s*ALLOW[\-_]NONDETERMINISTIC[\-_]ORDER', # Line with comment // ALLOW-NONDETERMINISTIC-ORDER
    r'//\s*lint:allow nondeterministic-order',  # Line with comment // lint:allow nondeterministic-order
    r'/\*.*ALLOW[\-_]NONDETERMINISTIC[\-_]ORDER.*\*/', # Comment block with ALLOW-NONDETERMINISTIC-ORDER
]

def is_excluded(filename):
    """Check if a file matches any excluded pattern."""
    for pattern in EXCLUDED_PATTERNS:
        if fnmatch.fnmatch(filename, pattern):
            return True
    return False

def has_exemption(lines, index):
    """Check if the code at the given index has an exemption comment nearby."""
    # Check 3 lines before and 3 lines after for an exemption comment
    start = max(0, index - 3)
    end = min(len(lines), index + 4)

    for i in range(start, end):
        for pattern in EXEMPT_PATTERNS:
            if re.search(pattern, lines[i]):
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
            content = f.read()
            lines = content.splitlines()

        # Search for SQL queries with ORDER BY using simpler regex pattern
        # This approach is more reliable than trying to track backticks
        for i, line in enumerate(lines):
            # Look for variable assignments with SQL that includes ORDER BY
            if ('ORDER BY' in line.upper() or
                ('ORDER BY' in content.upper() and ('`' in line and '=' in line or ':=' in line))):

                # Extract the SQL context (10 lines should be enough for most queries)
                context_start = max(0, i-5)
                context_end = min(len(lines), i+10)
                context = lines[context_start:context_end]
                context_text = '\n'.join(context)

                # Check if there's an exemption in the context
                if has_exemption(lines, i):
                    continue

                # Check if query has ORDER BY and look for a comma in the ORDER BY clause
                has_order_by = re.search(r'ORDER\s+BY', context_text, re.IGNORECASE)
                has_comma_in_order_by = re.search(r'ORDER\s+BY\s+[^,]+(,)', context_text, re.IGNORECASE)

                # If the query has ORDER BY but no comma (indicating a single column), report an error
                if has_order_by and not has_comma_in_order_by:
                    relative_path = os.path.relpath(filename)
                    print(f"[ERROR] {relative_path}:{i+1}: SQL query with potentially non-deterministic ordering.", file=sys.stderr)
                    print(f"    The ORDER BY clause should include a secondary sort key (usually 'id') for deterministic ordering.", file=sys.stderr)
                    print(f"    Example: 'ORDER BY created_at DESC, id ASC'", file=sys.stderr)
                    print(f"    Add // ALLOW-NONDETERMINISTIC-ORDER comment to exempt this query if needed.", file=sys.stderr)
                    print(f"", file=sys.stderr)
                    exit_code = 1
                    # Skip to avoid multiple errors for the same query
                    i = context_end

    except Exception as e:
        print(f"[ERROR] Failed to process {filename}: {e}", file=sys.stderr)

# Exit with the determined exit code
sys.exit(exit_code)
