# Code Coverage Thresholds

This file explains the code coverage threshold configuration in `coverage-thresholds.json`.

## Overview

The Scry API project enforces code coverage thresholds to maintain and improve code quality. These thresholds are defined in `coverage-thresholds.json` and are enforced during the CI build process.

## Configuration Structure

The configuration file has three main sections:

```json
{
  "default_threshold": 70,              // Default threshold for all packages
  "package_thresholds": {               // Package-specific thresholds
    "internal/domain": 90,
    "internal/domain/srs": 95,
    "internal/service": 85,
    ...
  },
  "excluded_packages": [                // Packages excluded from threshold checks
    "internal/testutils",
    "internal/testdb"
  ]
}
```

## Key Components

### Default Threshold

The `default_threshold` (70%) applies to:
- The overall project coverage
- Any package without a specific threshold defined

### Package-Specific Thresholds

The `package_thresholds` section defines higher requirements for critical code:
- Core domain logic (90-95%)
- Service implementations (85-90%)
- Data access layers (85%)

### Excluded Packages

The `excluded_packages` section lists packages excluded from coverage requirements:
- Test utilities and helpers
- Generated code

## Modifying Thresholds

When adjusting thresholds:
1. Consider the critical nature of the package
2. Avoid lowering thresholds for core business logic
3. Document reasons for any threshold changes in PR descriptions

## CI Enforcement

The CI pipeline enforces these thresholds:
- Per-package checks during the test job
- Overall project coverage check in the coverage-check job

Pull requests will fail if they reduce coverage below the defined thresholds.

## Writing Testable Code

To meet coverage requirements:
- Design for testability from the start
- Use dependency injection
- Keep functions small and focused
- Separate core logic from infrastructure
- Avoid complex conditional paths that are hard to test
