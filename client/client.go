package client

import (
	"bufio"
	"context"
	"crypto/tls"
	"strings"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
	"github.com/fatih/pool"
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
}

// NewHTTPProxyClient creates a new HTTPProxyClient object.
func NewHTTPProxyClient(proxyURL *url.URL, TLSConfig *tls.Config) *HTTPProxyClient {
	header := make(http.Header)
	header.Set("Proxy-Connection", "keep-alive")
	header.Set("User-Agent", "HTTPProxy/1.0")
	header.Set("X-Proxy-Boost", "boosted")
	// Cache the base64 result
	if proxyURL.User != nil && len(proxyURL.User.String()) > 0 {
		header.Set("Proxy-Authorization",
			"Basic " + base64.StdEncoding.EncodeToString([]byte(proxyURL.User.String())))
	}
	// Set default context
	ctx, cancelFn := context.WithCancel(context.Background())
	client := &HTTPProxyClient {
		ProxyURL:      *proxyURL,
		ConnectHeader: header,
		TLSConfig:     TLSConfig,
		ctx:           ctx,
		Cancel:        cancelFn,
	}
	// Create connection pool
	pool, err := pool.NewChannelPool(0, 30, client.factory)
	if err != nil {
		log.Fatalf("HTTP Proxy: Error when creating connection pool: %s", err)
		return nil
	}
	client.Pool = pool
	return client
}

// SetBasicAuth sets username and password for a HTTPProxyClient object.
func (p *HTTPProxyClient) SetBasicAuth(username, password string) error {
	if len(username) + len(password) == 0 {
		p.ConnectHeader.Del("Proxy-Authorization")
	} else {
		var user = url.UserPassword(username, password)
		if user == nil {
			return fmt.Errorf("invalid username or password inserted")
		}
		p.ConnectHeader.Set("Proxy-Authorization",
			"Basic " + base64.StdEncoding.EncodeToString([]byte(user.String())))
	}
	return nil
}

func (p *HTTPProxyClient) factory() (net.Conn, error) {
	var conn net.Conn
	var err  error
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
func (p *HTTPProxyClient) getConn() (*pool.PoolConn, error) {
	c, err := p.Pool.Get()
	if err != nil {
		return nil, err
	}
	pc, ok := c.(*pool.PoolConn);
	if !ok {
		return nil, fmt.Errorf("HTTP Proxy: Cannot cast connection into PoolConn")
	}
	one := []byte{}
	pc.SetReadDeadline(time.Now())
	if _, err := pc.Read(one); err == io.EOF || pc.Conn == nil {
		// Abandon expired connections
		// and get a new one.
		log.Debug("HTTP Proxy: Connection expired, abandon")
		pc.MarkUnusable()
		pc.Close()
		// get a new one
		return p.getConn()
	}
	pc.SetReadDeadline(time.Time{})
	return pc, nil
}

func (p *HTTPProxyClient) connect(targetAddr string) (net.Conn, error) {
	// Only Do CONNECT Method
	req := &http.Request{
		Method: http.MethodConnect,
		URL:    &url.URL{Opaque: targetAddr, Path: targetAddr},
		Host:   p.ProxyURL.Hostname(),
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
	targetAddr, err := santinizeAddr(targetAddr)
	if err != nil {
		return nil, err
	}
	conn, err := p.connect(targetAddr)
	pc := conn.(*pool.PoolConn)
	if err != nil {
		return nil, err
	}
	resp, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: http.MethodConnect})
	if err != nil {
		pc.Close()
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		f := strings.SplitN(resp.Status, " ", 2)
		pc.Close()
		return nil, fmt.Errorf(f[1])
	}
	pc.MarkUnusable()
	return conn, nil
}

// Redirect Connects to the proxy and Copy from the given connection.
// Using Redirect rather than Dial to save one RTT.
func (p *HTTPProxyClient) Redirect(srcConn net.Conn, targetAddr string) error {
	targetAddr, err := santinizeAddr(targetAddr)
	if err != nil {
		return err
	}
	conn, err := p.connect(targetAddr)
	pc := conn.(*pool.PoolConn)
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
		pc.Close()
		return err
	}
	if resp.StatusCode != http.StatusOK {
		f := strings.SplitN(resp.Status, " ", 2)
		pc.Close()
		return fmt.Errorf(f[1])
	}
	pc.MarkUnusable()
	// Start to copy other
	go CopyIO(srcConn, conn, term)
	// Send Signal to the first go routine.
	return nil
}

// RoundTrip request normal http request
func (p *HTTPProxyClient) RoundTrip(req *http.Request) (*http.Response, error) {
	// Get connection from pool
	pc, err := p.getConn()
	defer pc.Close()
	if err != nil {
		log.Errorf("HTTP Proxy: Cannot get connection from pool: %s", err)
	}
	// Add Password
	req.Header.Set("Proxy-Authorization", p.ConnectHeader.Get("Proxy-Authorization"))
	// Send Request to the Connection
	err = req.WriteProxy(pc)
	if err != nil {
		return nil, err
	}
	// Read reasponse
	reader := bufio.NewReader(pc)
	resp, err := http.ReadResponse(reader, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
