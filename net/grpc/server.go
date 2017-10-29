package grpc

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/stairlin/lego/ctx/app"
	"github.com/stairlin/lego/ctx/journey"
	"github.com/stairlin/lego/log"
	netcontext "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

const (
	// down mode is the default state. The handler is not ready to accept
	// new connections
	down uint32 = 0
	// up mode is when a handler accepts connections
	up uint32 = 1
	// drain mode is when a handler stops accepting new connection, but wait
	// for all existing in-flight requests to complete
	drain uint32 = 2
)

// A Server defines parameters for running a lego compatible GRPC server
type Server struct {
	mode uint32
	addr string

	opts             []grpc.ServerOption
	registrations    []registration
	services         []service
	unaryMiddlewares []UnaryServerMiddleware

	creds grpc.ServerOption

	app app.Ctx

	GRPC *grpc.Server
}

// NewServer creates a new GRPC server
func NewServer() *Server {
	return &Server{
		GRPC: grpc.NewServer(),
		unaryMiddlewares: []UnaryServerMiddleware{
			mwServerLogging,
			mwServerStats,
		},
	}
}

// Handle just injects the GRPC server to register a service. The function
// is called back only when Serve is called. This must be called before
// invoking Serve.
func (s *Server) Handle(f func(*grpc.Server)) {
	s.registrations = append(s.registrations, f)
}

// RegisterService register a service and its implementation to the gRPC
// server. Called from the IDL generated code.
//
// The function is called back only when Serve is called. This must be called
// before invoking Serve.
func (s *Server) RegisterService(sd *grpc.ServiceDesc, ss interface{}) {
	s.services = append(s.services, service{sd: sd, ss: ss})
}

// SetOptions changes the handler options
func (s *Server) SetOptions(opts ...grpc.ServerOption) {
	s.opts = append(s.opts, opts...)
}

// AppendUnaryMiddleware appends an unary middleware to the call chain
func (s *Server) AppendUnaryMiddleware(m UnaryServerMiddleware) {
	s.unaryMiddlewares = append(s.unaryMiddlewares, m)
}

// ActivateTLS activates TLS on this handler. That means only incoming TLS
// connections are allowed.
//
// If the certificate is signed by a certificate authority, the certFile should
// be the concatenation of the server's certificate, any intermediates,
// and the CA's certificate.
//
// Clients are not authenticated.
func (s *Server) ActivateTLS(certFile, keyFile string) {
	// Create the TLS credentials
	creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
	if err != nil {
		panic(err)
	}
	s.creds = grpc.Creds(creds)
}

// ActivateMutualTLS activates TLS on this handler. That means only incoming TLS
// connections are allowed and clients must authenticate themselves to the
// server.
//
// If the certificate is signed by a certificate authority, the certFile should
// be the concatenation of the server's certificate, any intermediates,
// and the CA's certificate.
func (s *Server) ActivateMutualTLS(certFile, keyFile, caFile string) {
	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		panic(errors.Wrap(err, "could not load server key pair"))
	}

	// Create a certificate pool from the certificate authority
	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(caFile)
	if err != nil {
		panic(errors.Wrap(err, "could not read ca certificate"))
	}
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		panic(errors.Wrap(err, "failed to append client certs"))
	}

	creds := credentials.NewTLS(&tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{certificate},
		ClientCAs:    certPool,
	})
	s.creds = grpc.Creds(creds)
}

// Serve starts serving HTTP requests (blocking call)
func (s *Server) Serve(addr string, ctx app.Ctx) error {
	s.app = ctx
	defer atomic.StoreUint32(&s.mode, down)

	// Register interceptor
	s.opts = append(s.opts, grpc.UnaryInterceptor(s.unaryInterceptor))

	tlsEnabled := s.creds != nil
	if tlsEnabled {
		s.SetOptions(s.creds)
	}

	s.GRPC = grpc.NewServer(s.opts...)
	s.addr = addr

	// Register endpoints/services
	for _, registration := range s.registrations {
		registration(s.GRPC)
	}
	for _, service := range s.services {
		s.GRPC.RegisterService(service.sd, service.ss)
	}

	// Register reflection service on gRPC server
	reflection.Register(s.GRPC)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	ctx.Trace("s.grpc.listen", "Listening...",
		log.String("addr", addr),
		log.Bool("tls", tlsEnabled),
	)
	atomic.StoreUint32(&s.mode, up)
	err = s.GRPC.Serve(lis)
	switch err := err.(type) {
	case *net.OpError:
		if err.Op == "accept" && s.isDraining() {
			return nil
		}
	}
	return err
}

// Drain puts the handler into drain mode.
func (s *Server) Drain() {
	atomic.StoreUint32(&s.mode, drain)
	s.GRPC.GracefulStop()
}

// isDraining checks whether the handler is draining
func (s *Server) isDraining() bool {
	return atomic.LoadUint32(&s.mode) == drain
}

func (s *Server) unaryInterceptor(
	context netcontext.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp interface{}, err error) {
	if s.app.Config().Request.AllowContext {
		var err error
		context, err = ExtractContext(context, s.app)
		if err != nil {
			return nil, err
		}
	} else {
		// TODO: Create journey from generic context
		context = journey.New(s.app)
	}
	ctx := context.(journey.Ctx)
	ctx.Store("Start-Time", time.Now())

	// Build middleware chain and then call it
	next := func(ctx journey.Ctx, req interface{}) (interface{}, error) {
		return handler(ctx, req)
	}
	for i := len(s.unaryMiddlewares) - 1; i >= 0; i-- {
		next = s.unaryMiddlewares[i](next)
	}
	return next(ctx, req)
}

type UnaryHandler func(ctx journey.Ctx, req interface{}) (interface{}, error)
type UnaryServerMiddleware func(next UnaryHandler) UnaryHandler

type registration func(s *grpc.Server)

type service struct {
	sd *grpc.ServiceDesc
	ss interface{}
}

// mwServerLogging logs information about HTTP requests/responses
func mwServerLogging(next UnaryHandler) UnaryHandler {
	return func(ctx journey.Ctx, req interface{}) (interface{}, error) {
		ctx.Trace("h.grpc.req.start", "Request start",
			log.Type("req", req),
		)

		// Next middleware
		res, err := next(ctx, req)

		startTime := ctx.Load("Start-Time").(time.Time)
		ctx.Trace("h.grpc.req.end", "Request end",
			log.Duration("duration", time.Since(startTime)),
			log.Error(err),
		)
		return res, err
	}
}

// mwServerStats sends the request/response stats
func mwServerStats(next UnaryHandler) UnaryHandler {
	return func(ctx journey.Ctx, req interface{}) (interface{}, error) {
		tags := map[string]string{
			"req": fmt.Sprintf("%T", req),
		}
		ctx.Stats().Inc("grpc.conc", tags)

		// Next middleware
		res, err := next(ctx, req)

		startTime := ctx.Load("Start-Time").(time.Time)
		ctx.Stats().Histogram("grpc.call", 1, tags)
		ctx.Stats().Timing("grpc.time", time.Since(startTime), tags)
		ctx.Stats().Dec("grpc.conc", tags)
		return res, err
	}
}
