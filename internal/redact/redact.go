// Package redact provides utilities for redacting sensitive information from strings
// before they are logged or returned in error responses. This package helps prevent
// the accidental leakage of credentials, connection strings, file paths, and other
// sensitive data that might be included in error messages.
package redact

import (
	"regexp"
	"sync"
)

// Constants for redaction placeholders
const (
	RedactionPlaceholder          = "[REDACTED]"
	RedactedPathPlaceholder       = "[REDACTED_PATH]"
	RedactedCredentialPlaceholder = "[REDACTED_CREDENTIAL]"
	RedactedKeyPlaceholder        = "[REDACTED_KEY]"
)

// Precompiled regex patterns
var (
	// Database connection strings
	dbConnRegex = regexp.MustCompile(`(?i)(postgres|mysql|mongodb|db|database|connection)://[^@]+@`)

	// Credentials and tokens
	passwordRegex = regexp.MustCompile(`(?i)(password|passwd|pwd)([=:\s]?['"]?)[^'"&\s]{3,}`)
	apiKeyRegex   = regexp.MustCompile(
		`(?i)(api[_-]?key|token|secret|key|access|auth)(['"\s:=]+)[A-Za-z0-9_\-.~+/]{8,}`,
	)
	awsKeyRegex = regexp.MustCompile(`(AKIA|AccessKey(Id)?)([^a-zA-Z0-9])?[A-Z0-9]{8,}`)
	// JWT token pattern - matches the standard three-part base64url-encoded JWT token format
	jwtTokenRegex = regexp.MustCompile(`eyJ[a-zA-Z0-9_-]+\.eyJ[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+`)

	// File paths
	unixPathRegex = regexp.MustCompile(`(/[\w.-]+){2,}`)
	winPathRegex  = regexp.MustCompile(`[A-Za-z]:\\[^\\]+(\\[^\\]+)+`)

	// Stack trace fragments
	stackTraceRegex = regexp.MustCompile(`(?:goroutine \d+|panic:)[\s\S]*?(\n\t.*)+`)

	// Email addresses
	emailRegex = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`)

	// SQL queries and fragments - enhanced patterns for more reliable redaction

	// General SQL query pattern - catches any SQL query regardless of type
	generalSQLRegex = regexp.MustCompile(
		`(?i)(SELECT|INSERT\s+INTO|UPDATE|DELETE\s+FROM|CREATE|ALTER|DROP|TRUNCATE)(\s+)([^;]{3,})`,
	)

	// SQL SELECT query pattern with improved matching
	sqlSelectRegex = regexp.MustCompile(
		`(?i)(SELECT\s+)(.+?)(\s+FROM\s+[\w_.]+)(.*)`,
	)

	// SQL INSERT query pattern with improved matching for different formats
	sqlInsertRegex = regexp.MustCompile(
		`(?i)(INSERT\s+INTO\s+[\w_.]+)(\s*(?:\([^)]*\)\s*)?(?:VALUES|SELECT))(.*)`,
	)

	// SQL UPDATE query pattern with better clause matching
	sqlUpdateRegex = regexp.MustCompile(
		`(?i)(UPDATE\s+[\w_.]+\s+SET)([^;]*)`,
	)

	// SQL DELETE query pattern with improved WHERE clause handling
	sqlDeleteRegex = regexp.MustCompile(
		`(?i)(DELETE\s+FROM\s+[\w_.]+)(\s+WHERE.*)`,
	)

	// Additional SQL patterns for better coverage
	sqlCreateRegex = regexp.MustCompile(
		`(?i)(CREATE\s+(?:TABLE|INDEX|VIEW|FUNCTION|PROCEDURE|TRIGGER)\s+[\w_.]+)(.*)`,
	)

	sqlAlterRegex = regexp.MustCompile(
		`(?i)(ALTER\s+(?:TABLE|INDEX|VIEW|FUNCTION|PROCEDURE|TRIGGER)\s+[\w_.]+)(.*)`,
	)

	// UUID pattern - specifically targets UUIDs that might appear in queries
	uuidRegex = regexp.MustCompile(
		`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`,
	)

	// PostgreSQL error message patterns
	pgErrorRegex = regexp.MustCompile(
		`(?i)(ERROR|SQLSTATE|DETAIL|HINT|CONTEXT|WHERE):?\s+([^:]+)`,
	)

	// SQL query detection pattern - useful for detecting SQL in general text
	sqlQueryDetectionRegex = regexp.MustCompile(
		`(?i)(?:SELECT|INSERT\s+INTO|UPDATE|DELETE\s+FROM|CREATE|ALTER|DROP)\s+[\w_.]+`,
	)

	// Additional sensitive patterns
	lineNumberRegex  = regexp.MustCompile(`(?:at )?line ?\d+`)
	syntaxErrorRegex = regexp.MustCompile(`(?i)syntax error|syntax problem|parse error`)
	hostPortRegex    = regexp.MustCompile(
		`\b(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}(?::\d{1,5})?\b`,
	)
	fileErrorRegex = regexp.MustCompile(
		`(?i)(?:no such file|file not found|can't open|cannot open|file error)`,
	)

	// All patterns and their placeholders
	patterns = []*regexp.Regexp{
		dbConnRegex, passwordRegex, apiKeyRegex, awsKeyRegex, jwtTokenRegex,
		unixPathRegex, winPathRegex, stackTraceRegex, emailRegex,
		// SQL patterns in specific order (most specific first)
		sqlInsertRegex, sqlUpdateRegex, sqlDeleteRegex, sqlSelectRegex,
		sqlCreateRegex, sqlAlterRegex, generalSQLRegex,
		pgErrorRegex, sqlQueryDetectionRegex,
		// Other patterns
		uuidRegex, lineNumberRegex, syntaxErrorRegex, hostPortRegex, fileErrorRegex,
	}

	patternPlaceholders = map[*regexp.Regexp]string{
		dbConnRegex:     RedactedCredentialPlaceholder,
		passwordRegex:   RedactedCredentialPlaceholder,
		apiKeyRegex:     RedactedKeyPlaceholder,
		awsKeyRegex:     RedactedKeyPlaceholder,
		jwtTokenRegex:   "[REDACTED_JWT]",
		unixPathRegex:   RedactedPathPlaceholder,
		winPathRegex:    RedactedPathPlaceholder,
		stackTraceRegex: "[STACK_TRACE_REDACTED]",
		emailRegex:      "[REDACTED_EMAIL]",
		// SQL patterns with improved redaction
		generalSQLRegex:        "$1 [REDACTED_SQL]",                     // General SQL redaction
		sqlSelectRegex:         "$1[REDACTED_COLUMNS]$3 [REDACTED_SQL]", // Preserve SELECT and FROM parts
		sqlInsertRegex:         "$1$2 [REDACTED_SQL]",                   // Preserve INSERT INTO and structure
		sqlUpdateRegex:         "$1 [REDACTED_SQL]",                     // Preserve UPDATE command
		sqlDeleteRegex:         "$1 [REDACTED_SQL]",                     // Preserve DELETE command
		sqlCreateRegex:         "$1 [REDACTED_SQL]",                     // Preserve CREATE command
		sqlAlterRegex:          "$1 [REDACTED_SQL]",                     // Preserve ALTER command
		pgErrorRegex:           "$1: [REDACTED_SQL_ERROR]",              // Preserve error type
		sqlQueryDetectionRegex: "[REDACTED_SQL]",                        // Full redaction for detected SQL
		uuidRegex:              "[REDACTED_UUID]",
		lineNumberRegex:        "[REDACTED_LINE_NUMBER]",
		syntaxErrorRegex:       "[REDACTED_SYNTAX_ERROR]",
		hostPortRegex:          "[REDACTED_HOST]",
		fileErrorRegex:         "[REDACTED_FILE_ERROR]",
	}

	mu sync.RWMutex
)

// String redacts sensitive information from the input string
func String(input string) string {
	if input == "" {
		return input
	}

	mu.RLock()
	defer mu.RUnlock()

	result := input
	for _, pattern := range patterns {
		placeholder := RedactionPlaceholder
		if ph, ok := patternPlaceholders[pattern]; ok {
			placeholder = ph
		}
		result = pattern.ReplaceAllString(result, placeholder)
	}

	return result
}

// Error redacts sensitive information from an error's Error() output
func Error(err error) string {
	if err == nil {
		return ""
	}

	return String(err.Error())
}
