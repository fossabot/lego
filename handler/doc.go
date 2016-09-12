// Package handler contains the main logic to accept external requests.
//
// I defines interfaces shared by other packages that accept
// external requests. Packages that implement these interfaces include:
//  * handler/http
//
// It also manages the lifecycle of those handlers. All handlers should be
// registered to this package in order to be gracefuly stopped (drained)
// when the application shuts down.
package handler
