package http

// HTTP status codes, defined in RFC 2616.
const (
	StatusContinue           = 100
	StatusSwitchingProtocols = 101

	StatusOK                   = 200
	StatusCreated              = 201
	StatusAccepted             = 202
	StatusNonAuthoritativeInfo = 203
	StatusNoContent            = 204
	StatusResetContent         = 205
	StatusPartialContent       = 206

	StatusMultipleChoices   = 300
	StatusMovedPermanently  = 301
	StatusFound             = 302
	StatusSeeOther          = 303
	StatusNotModified       = 304
	StatusUseProxy          = 305
	StatusTemporaryRedirect = 307

	StatusBadRequest                   = 400
	StatusUnauthorized                 = 401
	StatusPaymentRequired              = 402
	StatusForbidden                    = 403
	StatusNotFound                     = 404
	StatusMethodNotAllowed             = 405
	StatusNotAcceptable                = 406
	StatusProxyAuthRequired            = 407
	StatusRequestTimeout               = 408
	StatusConflict                     = 409
	StatusGone                         = 410
	StatusLengthRequired               = 411
	StatusPreconditionFailed           = 412
	StatusRequestEntityTooLarge        = 413
	StatusRequestURITooLong            = 414
	StatusUnsupportedMediaType         = 415
	StatusRequestedRangeNotSatisfiable = 416
	StatusExpectationFailed            = 417
	StatusTeapot                       = 418

	StatusInternalServerError     = 500
	StatusNotImplemented          = 501
	StatusBadGateway              = 502
	StatusServiceUnavailable      = 503
	StatusGatewayTimeout          = 504
	StatusHTTPVersionNotSupported = 505
)

const (
	// OPTIONS method represents a request for information about the communication options available
	// on the request/response chain identified by the Request-URI.
	OPTIONS = "OPTIONS"
	// GET method means retrieve whatever information (in the form of an entity) is identified by the Request-URI.
	GET = "GET"
	// HEAD method is identical to GET except that the server MUST NOT return a message-body in the response.
	// The metainformation contained in the HTTP headers in response to a HEAD request SHOULD be identical
	// to the information sent in response to a GET request.
	HEAD = "HEAD"
	// POST method is used to request that the origin server accept the entity enclosed in the request as
	// a new subordinate of the resource identified by the Request-URI in the Request-Line.
	POST = "POST"
	// PUT method requests that the enclosed entity be stored under the supplied Request-URI.
	// If the Request-URI refers to an already existing resource, the enclosed entity SHOULD be considered
	// as a modified version of the one residing on the origin server.
	// If the Request-URI does not point to an existing resource, and that URI is capable of being defined
	// as a new resource by the requesting user agent, the origin server can create the resource with that URI.
	PUT = "PUT"
	// DELETE method requests that the origin server delete the resource identified by the Request-URI.
	DELETE = "DELETE"
	// TRACE method is used to invoke a remote, application-layer loop-back of the request message.
	TRACE = "TRACE"

	// RFC 5789 (HTTP PATCH)
	// https://tools.ietf.org/html/rfc5789
	// The PATCH method is used to do partial resource modification.

	// PATCH is similar to PUT, but unlike PUT which proceeds to a complete replacement of a document,
	// PATCH to modifies partially an existing document.
	PATCH = "PATCH"

	// draft-snell-link-method-12 (HTTP LINK/UNLINK)
	// https://tools.ietf.org/html/draft-snell-link-method-12
	// The LINK and UNLINK methods are used to manage relationships between resources.

	// LINK is used to establish one or more relationships between the resource identified by the effective
	// request URI and one or more other resources.
	LINK = "LINK"
	// UNLINK is used to remove one or more relationships between the resource identified by the effective
	// request URI and other resources.
	UNLINK = "UNLINK"
)
