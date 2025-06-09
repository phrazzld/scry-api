# CGo Requirements for Database Integration

This document outlines the CGo requirements for the Scry API, particularly for PostgreSQL database integration.

## Overview

The Scry API uses the PostgreSQL database driver which requires CGo to be enabled and specific C libraries to be installed. Without these, database connectivity and tests will fail.

## Requirements

### 1. CGo Enabled

CGo must be enabled when building or running code that interacts with the database:

```bash
# Set the environment variable
export CGO_ENABLED=1

# Or prepend it to commands
CGO_ENABLED=1 go test ./...
```

### 2. Required C Libraries

The following libraries must be installed on your system:

- **GCC** - C compiler required by CGo
- **libpq-dev** - PostgreSQL client development libraries

#### Installation Instructions

**Ubuntu/Debian:**
```bash
sudo apt-get update
sudo apt-get install -y gcc libpq-dev
```

**macOS (with Homebrew):**
```bash
brew install gcc postgresql
```

**Windows:**
- Install GCC via MinGW or MSYS2
- Install PostgreSQL from the official installer which includes the required libraries

### 3. Verification

To verify you have the correct libraries installed, run:

```bash
# Verify GCC
gcc --version

# Verify libpq
pkg-config --libs libpq   # Should output linking information
```

## Common Issues

### 1. "cgo: C compiler not found"

This error indicates GCC is not installed or not in your PATH. Install GCC using the instructions above.

### 2. "cannot find -lpq"

This error indicates the PostgreSQL client libraries are missing. Install libpq-dev using the instructions above.

### 3. Tests pass locally but fail in CI

Make sure to run tests with CGO_ENABLED=1 locally to match the CI environment.

## CI Configuration

In our CI environment, we ensure these requirements are met by:

1. Setting `CGO_ENABLED=1` in all relevant workflow steps
2. Installing the required C libraries (`gcc`, `libpq-dev`) via apt-get
3. Verifying installations before running database-dependent tests

## Related Documentation

For more information, see:
- [Go Database Documentation](https://golang.org/pkg/database/sql/)
- [PostgreSQL Driver Documentation](https://github.com/lib/pq)
