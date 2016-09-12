// Package ctx defines the context types, which carry information defined
// for a specific scope (application, request, ...)
// A context can be passed across API boundaries and between processes.
//
// Incoming requests to a server should create a Context, and outgoing calls to
// servers should accept a Context.  The chain of function calls between must
// propagate the Context, optionally replacing it with a modified copy.
//
// Programs that use Contexts should follow these rules to keep interfaces
// consistent across packages and enable static analysis tools to check context
// propagation
package ctx
