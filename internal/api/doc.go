// Package api handles incoming HTTP requests, routing, request validation,
// and response formatting. It acts as an adapter between external clients
// and the internal application services, translating HTTP concerns to
// business operations.
//
// The api package is organized as an adapter layer in the hexagonal architecture,
// connecting external HTTP clients to the core application domain without
// leaking HTTP-specific concerns into the domain. It has several key responsibilities:
//
// Key responsibilities:
//
// 1. HTTP Routing: Defining and configuring API endpoints and their HTTP methods.
//
//  2. Request Validation: Ensuring incoming requests contain valid data before
//     passing them to the application services.
//
// 3. Authentication: Verifying user identity through JWT tokens for protected routes.
//
//  4. Response Formatting: Converting domain objects and errors into appropriate
//     HTTP responses with proper status codes and JSON formatting.
//
//  5. Error Handling: Transforming domain and application errors into meaningful
//     HTTP error responses with appropriate status codes.
//
// The api package depends on the service package for business logic and the
// domain package for entity definitions, but other packages should not depend
// on api, maintaining the dependency inversion principle.
package api
