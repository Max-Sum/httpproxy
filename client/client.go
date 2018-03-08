package client

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/silenceper/pool"
)

// HTTPProxyClient is a backend client.
// It could be used to construct a new client with various backend.
type HTTPProxyClient struct {
	Cancel        context.CancelFunc
	ProxyURL      url.URL
	ConnectHeader http.Header
	TLSConfig     *tls.Config
	Pool          pool.Pool
	ctx           context.Context
	bogusDNS      *BogusDNS
}

// NewHTTPProxyClient creates a new HTTPProxyClient object.
func NewHTTPProxyClient(proxyURL *url.URL, TLSConfig *tls.Config, bogus *BogusDNS) *HTTPProxyClient {
	header := make(http.Header)
	header.Set("Proxy-Connection", "keep-alive")
	header.Set("User-Agent", "HTTPProxy/1.0")
	header.Set("X-Proxy-Boost", "boosted")
	// Cache the base64 result
	if proxyURL.User != nil && len(proxyURL.User.String()) > 0 {
		header.Set("Proxy-Authorization",
			"Basic "+base64.StdEncoding.EncodeToString([]byte(proxyURL.User.String())))
	}
	// Set default context
	ctx, cancelFn := context.WithCancel(context.Background())
	client := &HTTPProxyClient{
		ProxyURL:      *proxyURL,
		ConnectHeader: header,
		TLSConfig:     TLSConfig,
		ctx:           ctx,
		Cancel:        cancelFn,
		bogusDNS:      bogus,
	}
	// Create connection pool
	poolConfig := &pool.PoolConfig{
		InitialCap:  0,
		MaxCap:      cnfg.MaxIdleConnections,
		Factory:     client.factory,
		Close:       func(c interface{}) error { return c.(net.Conn).Close() },
		IdleTimeout: cnfg.IdleTime * time.Second,
	}
	pool, err := pool.NewChannelPool(poolConfig)
	if err != nil {
		log.Fatalf("HTTP Proxy: Error when creating connection pool: %s", err)
		return nil
	}
	client.Pool = pool
	return client
}

// SetBasicAuth sets username and password for a HTTPProxyClient object.
func (p *HTTPProxyClient) SetBasicAuth(username, password string) error {
	if len(username)+len(password) == 0 {
		p.ConnectHeader.Del("Proxy-Authorization")
	} else {
		var user = url.UserPassword(username, password)
		if user == nil {
			return fmt.Errorf("invalid username or password inserted")
		}
		p.ConnectHeader.Set("Proxy-Authorization",
			"Basic "+base64.StdEncoding.EncodeToString([]byte(user.String())))
	}
	return nil
}

func (p *HTTPProxyClient) factory() (interface{}, error) {
	var conn net.Conn
	var err error
	switch scheme := p.ProxyURL.Scheme; scheme {
	case "http":
		conn, err = (&net.Dialer{}).DialContext(p.ctx, "tcp", p.ProxyURL.Host)
	case "https":
		conn, err = tls.DialWithDialer((&net.Dialer{}), "tcp", p.ProxyURL.Host, p.TLSConfig)
	default:
		err = fmt.Errorf("unsupported Proxy scheme: %s", scheme)
	}
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// get a connection and check if it's still usable.
func (p *HTTPProxyClient) getConn() (net.Conn, error) {
	v, err := p.Pool.Get()
	if err != nil {
		return nil, err
	}
	c, ok := v.(net.Conn)
	if !ok {
		return nil, fmt.Errorf("HTTP Proxy: Cannot get netConn")
	}
	one := []byte{0}
	c.SetReadDeadline(time.Now().Add(10 * time.Microsecond))
	if _, err := c.Read(one); err == io.EOF {
		// Abandon expired connections
		// and get a new one.
		log.Info("HTTP Proxy: Connection closed, abandon")
		c.Close()
		return p.getConn()
	}
	c.SetReadDeadline(time.Time{})
	return c, nil
}

// probeAddress check if the address is a bogus IP,
// and try to translate it back to original address
func (p *HTTPProxyClient) probeAddress(host string) string {
	if p.bogusDNS == nil {
		return host
	}
	ip := net.ParseIP(host)
	orig, err := p.bogusDNS.GetAddress(ip)
	if err != nil {
		log.Debug(err)
		return host
	}
	log.Debugf("HTTP Proxy: got generic address %s from bogus IP %s", orig, host)
	return orig
}

func (p *HTTPProxyClient) connect(targetAddr string) (net.Conn, error) {
	// Parse Host
	host, port, err := net.SplitHostPort(targetAddr)
	if err != nil {
		return nil, err
	}
	host = p.probeAddress(host)
	addr := net.JoinHostPort(host, port)
	// Only Do CONNECT Method
	req := &http.Request{
		Method: http.MethodConnect,
		URL:    &url.URL{Opaque: addr, Path: addr},
		Host:   cnfg.Hostname,
		Header: p.ConnectHeader,
	}
	// Get connection from pool
	pc, err := p.getConn()
	if err != nil {
		log.Errorf("HTTP Proxy: Cannot get connection from pool: %s", err)
	}
	// Send Request to the Connection
	err = req.WriteProxy(pc)
	if err != nil {
		pc.Close()
		return nil, err
	}
	return pc, nil
}

// Dial is just a function to perform Connection to the Proxy.
// It returns after receiving 200 Connection Established.
// It will not send out any message.
func (p *HTTPProxyClient) Dial(targetAddr string) (net.Conn, error) {
	conn, err := p.connect(targetAddr)
	if err != nil {
		return nil, err
	}
	resp, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: http.MethodConnect})
	if err != nil {
		p.Pool.Put(conn)
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		f := strings.SplitN(resp.Status, " ", 2)
		p.Pool.Put(conn)
		return nil, fmt.Errorf(f[1])
	}
	return conn, nil
}

// Redirect Connects to the proxy and Copy from the given connection.
// Using Redirect rather than Dial to save one RTT.
func (p *HTTPProxyClient) Redirect(srcConn net.Conn, targetAddr string) error {
	conn, err := p.connect(targetAddr)
	if err != nil {
		return err
	}
	// Copy Request IO before read 200 OK.
	// So that the proxy server could start transmission faster.
	term := make(chan bool, 1)
	term <- false
	go CopyIO(conn, srcConn, term)
	resp, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: http.MethodConnect})
	if err != nil {
		p.Pool.Put(conn)
		return err
	}
	if resp.StatusCode != http.StatusOK {
		f := strings.SplitN(resp.Status, " ", 2)
		p.Pool.Put(conn)
		return fmt.Errorf(f[1])
	}
	// Start to copy other
	go CopyIO(srcConn, conn, term)
	// Send Signal to the first go routine.
	return nil
}

// A custom body contains a afterClose hook
type body struct {
	src        io.ReadCloser
	afterClose func() error
}

func (b *body) Read(p []byte) (n int, err error) {
	return b.src.Read(p)
}

// before closing the body, trigger afterClose hook
func (b *body) Close() (err error) {
	if err := b.src.Close(); err != nil {
		return err
	}
	return b.afterClose()
}

// RoundTrip request normal http request
func (p *HTTPProxyClient) RoundTrip(req *http.Request) (*http.Response, error) {
	// Get connection from pool
	c, err := p.getConn()
	if err != nil {
		log.Errorf("HTTP Proxy: Cannot get connection from pool: %s", err)
	}
	// Probe the address
	host := p.probeAddress(req.URL.Hostname())
	port := req.URL.Port()
	if port != "" {
		host = net.JoinHostPort(host, port)
	}
	req.URL.Host = host
	req.Host = host
	// Add Password
	req.Header.Set("Proxy-Authorization", p.ConnectHeader.Get("Proxy-Authorization"))

	// Send Request to the Connection
	log.Debugf("HTTP Proxy: sending a request to %v", req.URL)
	err = req.WriteProxy(c)
	if err != nil {
		return nil, err
	}
	// Read reasponse
	reader := bufio.NewReader(c)
	resp, err := http.ReadResponse(reader, req)
	// New body
	b := &body{
		src: resp.Body,
		afterClose: func() error {
			log.Info("body hook: put back connection")
			p.Pool.Put(c)
			return nil
		},
	}
	resp.Body = b
	if err != nil {
		return nil, err
	}
	return resp, nil
}
