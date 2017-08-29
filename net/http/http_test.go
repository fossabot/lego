package http_test

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"math"
	netHttp "net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stairlin/lego/ctx/journey"
	"github.com/stairlin/lego/net/http"
	lt "github.com/stairlin/lego/testing"
)

// portSequence returns a sequence of port numbers. It should be used
// for test handlers in order to avoid port clashes
var portSequence = &sequencer{n: 9900}

type sequencer struct {
	mu sync.Mutex
	n  int
}

func (s *sequencer) next() int {
	s.mu.Lock()
	p := s.n
	s.n++
	s.mu.Unlock()
	return p
}

func TestHTTP(t *testing.T) {
	tt := lt.New(t)
	appCtx := tt.NewAppCtx("test-http")

	// Build handler
	h := http.NewServer()
	addr := fmt.Sprintf("127.0.0.1:%d", portSequence.next())
	h.HandleFunc("/preflight", http.GET, func(
		ctx journey.Ctx, w http.ResponseWriter, r *http.Request,
	) {
		w.Head(http.StatusOK)
	})

	// Start serving requests
	go func() {
		err := h.Serve(addr, appCtx)
		if err != nil {
			panic(err)
		}
	}()
	// Ensure HTTP handler is ready to serve requests
	var lastRes *netHttp.Response
	for attempt := 1; attempt <= 10; attempt++ {
		ctx := journey.New(appCtx)
		res, err := http.Get(ctx, fmt.Sprintf("http://%s/preflight", addr))
		lastRes = res
		if err == nil && res.StatusCode == http.StatusOK {
			break
		}
		backoff := math.Pow(2, float64(attempt))
		time.Sleep(time.Millisecond * time.Duration(backoff))
	}

	if http.StatusOK != lastRes.StatusCode {
		t.Errorf("expect to reach endpoint, but got code %d", lastRes.StatusCode)
	}
}

func TestHTTPS(t *testing.T) {
	tt := lt.New(t)
	appCtx := tt.NewAppCtx("test-https")

	// Build handler
	h := http.NewServer()
	addr := fmt.Sprintf("127.0.0.1:%d", portSequence.next())
	h.HandleFunc("/preflight", http.GET, func(
		ctx journey.Ctx, w http.ResponseWriter, r *http.Request,
	) {
		w.Head(http.StatusOK)
	})

	// Activate TLS
	h.ActivateTLS("./test_cert.pem", "./test_key.pem")

	// Start serving requests
	go func() {
		err := h.Serve(addr, appCtx)
		if err != nil {
			panic(err)
		}
	}()
	// Ensure HTTP handler is ready to serve requests
	var lastRes *netHttp.Response
	client := http.Client{
		HTTP: netHttp.Client{
			Transport: &netHttp.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
	}
	for attempt := 1; attempt <= 10; attempt++ {
		ctx := journey.New(appCtx)
		res, err := client.Get(ctx, fmt.Sprintf("https://%s/preflight", addr))
		if err == nil {
			lastRes = res
			break
		}
		switch {
		case strings.Contains(err.Error(), "cannot validate certificate"):
			t.Fatal("got a bad certificate")
		case strings.HasSuffix(err.Error(), "getsockopt: connection refused"):
			t.Log("handler not ready")
		default:
			t.Fatalf("got unexpected error %s", err.Error())
		}
		backoff := math.Pow(2, float64(attempt))
		time.Sleep(time.Millisecond * time.Duration(backoff))
	}

	if lastRes == nil {
		t.Fatal("expect to reach endpoint")
	}
	if http.StatusOK != lastRes.StatusCode {
		t.Errorf("expect status OK, but got code %d", lastRes.StatusCode)
	}
}

func TestHTTPS_WithConfig(t *testing.T) {
	tt := lt.New(t)
	appCtx := tt.NewAppCtx("test-https")

	// Build handler
	h := http.NewServer()
	addr := fmt.Sprintf("127.0.0.1:%d", portSequence.next())
	h.HandleFunc("/preflight", http.GET, func(
		ctx journey.Ctx, w http.ResponseWriter, r *http.Request,
	) {
		w.Head(http.StatusOK)
	})

	// Set TLS
	tlsConfig := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}
	h.ActivateTLS("./test_cert.pem", "./test_key.pem")
	h.SetOptions(http.OptTLS(tlsConfig))

	// Start serving requests
	go func() {
		err := h.Serve(addr, appCtx)
		if err != nil {
			panic(err)
		}
	}()
	// Ensure HTTP handler is ready to serve requests
	var lastRes *netHttp.Response
	client := http.Client{
		HTTP: netHttp.Client{
			Transport: &netHttp.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
	}
	for attempt := 1; attempt <= 10; attempt++ {
		ctx := journey.New(appCtx)
		res, err := client.Get(ctx, fmt.Sprintf("https://%s/preflight", addr))
		if err == nil {
			lastRes = res
			break
		}
		switch {
		case strings.Contains(err.Error(), "cannot validate certificate"):
			t.Fatal("got a bad certificate")
		case strings.HasSuffix(err.Error(), "getsockopt: connection refused"):
			t.Log("handler not ready")
		default:
			t.Fatalf("got unexpected error %s", err.Error())
		}
		backoff := math.Pow(2, float64(attempt))
		time.Sleep(time.Millisecond * time.Duration(backoff))
	}

	if lastRes == nil {
		t.Fatal("expect to reach endpoint")
	}
	if http.StatusOK != lastRes.StatusCode {
		t.Errorf("expect status OK, but got code %d", lastRes.StatusCode)
	}
}

func TestHTTP_Static(t *testing.T) {
	tt := lt.New(t)
	appCtx := tt.NewAppCtx("test-static-http")

	// Build handler
	h := http.NewServer()
	addr := fmt.Sprintf("127.0.0.1:%d", portSequence.next())
	h.HandleFunc("/preflight", http.GET, func(
		ctx journey.Ctx, w http.ResponseWriter, r *http.Request,
	) {
		w.Head(http.StatusOK)
	})
	h.HandleStatic("/assets", "./")

	// Start serving requests
	go func() {
		err := h.Serve(addr, appCtx)
		if err != nil {
			panic(err)
		}
	}()
	// Ensure HTTP handler is ready to serve requests
	var lastRes *netHttp.Response
	for attempt := 1; attempt <= 10; attempt++ {
		ctx := journey.New(appCtx)
		res, err := http.Get(ctx, fmt.Sprintf("http://%s/preflight", addr))
		lastRes = res
		if err == nil && res.StatusCode == http.StatusOK {
			break
		}
		backoff := math.Pow(2, float64(attempt))
		time.Sleep(time.Millisecond * time.Duration(backoff))
	}

	if http.StatusOK != lastRes.StatusCode {
		t.Errorf("expect to reach preflight endpoint, but got code %d", lastRes.StatusCode)
	}

	ctx := journey.New(appCtx)
	res, err := http.Get(ctx, fmt.Sprintf("http://%s/assets/test_file.txt", addr))
	if err != nil {
		t.Fatalf("unexpected error on assets endpoint %s", err)
	}
	if http.StatusOK != res.StatusCode {
		t.Fatalf("expect code %d, but got %d", http.StatusOK, res.StatusCode)
	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("unexpected error when reading response body %s", err)
	}
	defer res.Body.Close()
	expectData := "hello from a static endpoint"
	if expectData != string(data) {
		t.Errorf("expect code %s, but got %s", expectData, string(data))
	}
}
