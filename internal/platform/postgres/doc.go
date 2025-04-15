// Package postgres provides PostgreSQL-specific implementations for the data
// storage interfaces (repositories) defined in the internal/store package.
// It handles the details of database connections, query execution, and data
// mapping between domain entities and database records.
//
// The postgres package is an infrastructure adapter in the hexagonal architecture,
// translating between the application's domain models and the PostgreSQL database.
// It implements the repository interfaces defined in the store package, allowing
// the application core to remain agnostic about the specific database technology.
//
// Key components:
//
// 1. Store Implementations:
//   - PostgresUserStore: Implements store.UserStore for user persistence
//   - Additional stores for Memo, Card, and UserCardStats (upcoming)
//
// 2. Database Connectivity:
//   - Connection pool management
//   - Transaction handling
//   - Connection configuration
//
// 3. Data Mapping:
//   - Translates between domain entities and database rows
//   - Handles conversion of custom types (UUIDs, enums, JSON, etc.)
//
// 4. SQL Query Execution:
//   - Prepared statements for SQL injection prevention
//   - Parameter binding
//   - Result scanning into domain entities
//
// The DBTX interface is used throughout to support both direct database
// connections (*sql.DB) and transactions (*sql.Tx), enabling transaction-based
// test isolation and proper transaction management in production code.
package postgres
