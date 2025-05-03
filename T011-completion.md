# T011 - SQL Query Patterns for Sensitive Data Analysis

## Overview

This document analyzes the SQL query patterns in the Scry API codebase to identify where sensitive data might appear in logs. The goal is to improve the SQL redaction logic to better protect sensitive information.

## Current Redaction Implementation

The current implementation in `internal/redact/redact.go` uses a single regex pattern for SQL queries:

```go
// SQL queries and fragments
sqlRegex = regexp.MustCompile(
    `(?i)(SELECT|INSERT|UPDATE|DELETE|CREATE|ALTER|DROP|GRANT)[\s\w,*()]+(?:FROM|INTO|SET|TABLE|DATABASE|SCHEMA|VIEW)(?:[\s\w,*()='"]+)?`,
),
```

This pattern is too broad and simply redacts entire SQL queries with `[REDACTED_SQL]`. While this is secure, it's not optimal for debugging since it hides both sensitive and non-sensitive parts of queries.

## SQL Query Patterns Identified

After analyzing the codebase, we identified the following common SQL query patterns that might contain sensitive data:

### 1. Query Parameter Placeholders

All SQL queries in the codebase use parameterized queries with placeholders (`$1`, `$2`, etc.), which provides protection against SQL injection and prevents sensitive data from appearing directly in query strings.

Example from `user_store.go`:
```go
_, err := s.db.ExecContext(ctx, `
    INSERT INTO users (id, email, hashed_password, created_at, updated_at)
    VALUES ($1, $2, $3, $4, $5)
`, user.ID, user.Email, user.HashedPassword, user.CreatedAt, user.UpdatedAt)
```

### 2. Sensitive Data in WHERE Clauses

WHERE clauses often contain identifiers or other potentially sensitive data:

- User IDs: `WHERE user_id = $1`
- Email addresses: `WHERE LOWER(email) = LOWER($1)`
- Card IDs: `WHERE id = $1`
- Memo IDs: `WHERE id = $1`

Examples:
```sql
SELECT id, email, hashed_password, created_at, updated_at
FROM users
WHERE id = $1

SELECT id, email, hashed_password, created_at, updated_at
FROM users
WHERE LOWER(email) = LOWER($1)
```

### 3. Sensitive Data in INSERT Statements

INSERT statements with VALUES clauses contain sensitive data:

- User email addresses: `VALUES ($1, $2, $3, $4, $5)` where `$2` is the email
- Hashed passwords: `VALUES ($1, $2, $3, $4, $5)` where `$3` is the hashed password
- User IDs: `VALUES ($1, $2, $3, $4, $5, $6)` where `$1` and/or `$2` might be user IDs
- Card content: `VALUES ($1, $2, $3, $4, $5, $6)` where `$4` might be card content

Examples:
```sql
INSERT INTO users (id, email, hashed_password, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5)

INSERT INTO cards (id, user_id, memo_id, content, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
```

### 4. Sensitive Data in UPDATE Statements

UPDATE statements with SET clauses contain sensitive data:

- Email addresses: `SET email = $1, ...`
- Hashed passwords: `SET ... hashed_password = $2, ...`
- Card content: `SET content = $1, ...`

Examples:
```sql
UPDATE users
SET email = $1, hashed_password = $2, updated_at = $3
WHERE id = $4

UPDATE cards
SET content = $1, updated_at = $2
WHERE id = $3
```

### 5. JOIN Statements with Sensitive Fields

JOINs often include user IDs or other sensitive identifiers:

```sql
SELECT c.id, c.user_id, c.memo_id, c.content, c.created_at, c.updated_at
FROM cards c
JOIN user_card_stats ucs ON c.id = ucs.card_id
WHERE c.user_id = $1
  AND ucs.user_id = $1
  AND ucs.next_review_at <= NOW()
```

### 6. DELETE Statements with Sensitive Data in WHERE Clauses

DELETE statements often contain sensitive identifiers in WHERE clauses:

```sql
DELETE FROM users
WHERE id = $1

DELETE FROM cards
WHERE id = $1
```

## Potentially Sensitive Data Types

Based on the analysis, the following data types should be considered sensitive:

1. **User Identifiers**
   - User IDs (UUID format)
   - Email addresses

2. **Authentication Data**
   - Hashed passwords
   - JWT tokens (if logged)

3. **Card/Memo Content**
   - Card content JSON
   - Memo text

4. **Timestamps**
   - Related to user activity (might leak usage patterns)

## Patterns for Enhanced Redaction

To improve SQL redaction while maintaining debugging utility, we should target the following specific patterns:

1. **Values in WHERE Clauses**
   - `WHERE\s+[\w_.]+\s*=\s*['"]?([^'")\s]+)['"]?`
   - Example: `WHERE user_id = '12345'` → `WHERE user_id = '[REDACTED]'`

2. **Literal Values in INSERT Statements**
   - `VALUES\s*\(([^)]*)\)`
   - Example: `VALUES ('id', 'email@example.com', 'hashed_pwd')` → `VALUES ('[REDACTED]', '[REDACTED]', '[REDACTED]')`

3. **Values in SET Clauses**
   - `SET\s+[\w_.]+\s*=\s*['"]?([^'")\s,]+)['"]?`
   - Example: `SET email = 'user@example.com'` → `SET email = '[REDACTED]'`

4. **Literal UUIDs**
   - `[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`
   - Example: `123e4567-e89b-12d3-a456-426614174000` → `[REDACTED_UUID]`

5. **Email Addresses**
   - `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`
   - Example: `user@example.com` → `[REDACTED_EMAIL]`

## Recommendations

Based on our analysis, we recommend the following approach for T012:

1. **Pattern-Based Redaction**:
   - Instead of redacting entire SQL queries, implement targeted redaction for sensitive values.
   - Keep the SQL structure visible for debugging, but redact specific values.

2. **Structured Logging Contexts**:
   - Avoid logging full SQL queries in error contexts where possible.
   - Use structured logging with field-specific redaction.

3. **Implementation Strategy**:
   - Use multiple regex patterns for different types of SQL clauses.
   - Implement special handling for known sensitive data types (UUIDs, emails, etc.).
   - Consider using a SQL parser for more reliable redaction in complex cases.

4. **Contingency Plan**:
   - If regex patterns become too complex or brittle, implement a fallback strategy that redacts entire SQL statements but keeps the SQL command visible.
   - Example: `INSERT INTO users VALUES (...)` → `INSERT INTO users VALUES [VALUES_REDACTED]`

This analysis provides the foundation for implementing enhanced SQL redaction logic in T012.
