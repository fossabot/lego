package grpc

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"

	"github.com/pkg/errors"
	"github.com/stairlin/lego/ctx/app"
	netcontext "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Client is a wrapper for the grpc client.
type Client struct {
	unaryMiddlewares []UnaryClientMiddleware

	// HTTP is the standard net/http client
	GRPC *grpc.ClientConn
	// PropagateContext tells whether the journey should be propagated upstream
	//
	// This should be activated when the upstream endpoint is a LEGO service
	// or another LEGO-compatible service. The context can potentially leak
	// sensitive information, so do not activate it for services that you
	// don't trust.
	PropagateContext bool
}

func NewClient(
	appCtx app.Ctx, target string, opts ...grpc.DialOption,
) (*Client, error) {
	client := &Client{}

	// Add default dial options
	opts = append(opts,
		grpc.WithUnaryInterceptor(client.unaryInterceptor),
		grpc.WithBlock(),
	)

	// Dial GRPC connection
	conn, err := grpc.DialContext(appCtx, target, opts...)
	if err != nil {
		return nil, err
	}
	client.GRPC = conn
	return client, nil
}

func (c *Client) AppendUnaryMiddleware(m UnaryClientMiddleware) {
	c.unaryMiddlewares = append(c.unaryMiddlewares, m)
}

func (c *Client) Close() error {
	return c.GRPC.Close()
}

// WithTLS returns a dial option for the GRPC client that activates
// TLS. This must be used when the server has TLS activated.
func WithTLS(
	certFile, serverNameOverride string,
) (grpc.DialOption, error) {
	creds, err := credentials.NewClientTLSFromFile(certFile, serverNameOverride)
	if err != nil {
		return nil, errors.Wrap(err, "could not load certificate")
	}
	return grpc.WithTransportCredentials(creds), nil
}

// WithMutualTLS returns a dial option for the GRPC client that activates
// a mutual TLS authentication between the server and the client.
func WithMutualTLS(
	serverName, certFile, keyFile, caFile string,
) (grpc.DialOption, error) {
	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, errors.Wrap(err, "could not load client key pair")
	}

	// Create a certificate pool from the certificate authority
	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(caFile)
	if err != nil {
		return nil, errors.Wrap(err, "could not read ca certificate")
	}
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		return nil, errors.Wrap(err, "failed to append ca certs")
	}

	creds := credentials.NewTLS(&tls.Config{
		ServerName:   serverName,
		Certificates: []tls.Certificate{certificate},
		RootCAs:      certPool,
	})
	return grpc.WithTransportCredentials(creds), nil
}

// MustDialOption panics if it receives an error
func MustDialOption(opt grpc.DialOption, err error) grpc.DialOption {
	if err != nil {
		panic(err)
	}
	return opt
}

// unaryInterceptor intercepts the execution of a unary RPC on the client.
// invoker is the handler to complete the RPC and it is the responsibility of
// the interceptor to call it. This is an EXPERIMENTAL API.
func (c *Client) unaryInterceptor(
	ctx netcontext.Context,
	method string,
	req, reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	if c.PropagateContext {
		var err error
		ctx, err = EmbedContext(ctx)
		if err != nil {
			return err
		}
	}

	// Build middleware chain and then call it
	next := invoker
	for i := len(c.unaryMiddlewares) - 1; i >= 0; i-- {
		next = c.unaryMiddlewares[i](next)
	}
	return next(ctx, method, req, reply, cc, opts...)
}

type UnaryClientMiddleware func(grpc.UnaryInvoker) grpc.UnaryInvoker
