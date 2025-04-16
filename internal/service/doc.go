// Package service contains the application-specific use cases and business
// logic. It orchestrates interactions between domain objects and repositories
// (defined in internal/store) to fulfill application features.
//
// The service package implements the application layer in the clean architecture,
// containing use cases that coordinate the flow of data between external interfaces
// (API, message queues, etc.) and the domain layer. It abstracts away infrastructure
// details while orchestrating domain entities to fulfill business requirements.
//
// Key components:
//
// 1. Service Interfaces:
//   - Define application-specific operations available to the delivery mechanisms
//   - Each service focuses on a specific domain area (authentication, card management, etc.)
//
// 2. Use Case Implementations:
//   - Coordinate between multiple repositories and domain services
//   - Apply transactional boundaries when operations span multiple repositories
//   - Enforce application-level business rules that span multiple domain entities
//
// 3. Dependency Management:
//   - Services receive dependencies through constructor injection
//   - Core dependencies include repositories, domain services, and cross-cutting concerns
//
// 4. Error Handling:
//   - Translate domain-specific errors to application-level errors
//   - Provide meaningful error context for API responses
//
// The service layer depends on domain entities and repository interfaces (from store),
// but never on specific infrastructure implementations, maintaining the Dependency
// Inversion Principle of clean architecture.
package service
