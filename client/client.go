package client

import (
	"bufio"
	"context"
	"crypto/tls"
	"strings"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
)

// HTTPProxyClient is a backend client.
// It could be used to construct a new client with various backend.
type HTTPProxyClient struct {
	Tr     http.Transport
	ctx    context.Context
	Cancel context.CancelFunc
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
	return &HTTPProxyClient {
		Tr: http.Transport{
			TLSClientConfig:    TLSConfig,
			Proxy:				http.ProxyURL(proxyURL),
			ProxyConnectHeader: header,
		},
		ctx: ctx,
		Cancel: cancelFn,
	}
}

// SetBasicAuth sets username and password for a HTTPProxyClient object.
func (p *HTTPProxyClient) SetBasicAuth(username, password string) error {
	proxyURL, _ := p.Tr.Proxy(nil)
	if len(username) + len(password) == 0 {
		p.Tr.ProxyConnectHeader.Del("Proxy-Authorization")
		proxyURL.User = nil
	} else {
		var user = url.UserPassword(username, password)
		if user == nil {
			return fmt.Errorf("invalid username or password inserted")
		}
		p.Tr.ProxyConnectHeader.Set("Proxy-Authorization",
			"Basic "+base64.StdEncoding.EncodeToString([]byte(user.String())))
		proxyURL.User = user
	}
	p.Tr.Proxy = http.ProxyURL(proxyURL)
	return nil
}

func (p *HTTPProxyClient) connect(targetAddr string) (*net.TCPConn, error) {
	var conn net.Conn
	var err error
	// Only Do CONNECT Method
	req := &http.Request{
		Method: http.MethodConnect,
		URL:    &url.URL{Opaque: targetAddr},
		Host:   targetAddr,
		Header: p.Tr.ProxyConnectHeader,
	}
	// Address
	proxyURL, _ := p.Tr.Proxy(req)
	proxyAddr := canonicalAddr(proxyURL)
	switch scheme := proxyURL.Scheme; scheme {
	case "http":
		conn, err = (&net.Dialer{}).DialContext(p.ctx, "tcp", proxyAddr)
	case "https":
		conn, err = tls.DialWithDialer((&net.Dialer{}), "tcp", proxyAddr, p.Tr.TLSClientConfig)
	default:
		err = fmt.Errorf("unsupported Proxy scheme: %s", scheme)
	}
	if err != nil {
		return nil, err
	}
	// Send Request to the Connection
	err = req.WriteProxy(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return conn.(*net.TCPConn), nil
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
	if err != nil {
		return nil, err
	}
	resp, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: http.MethodConnect})
	if err != nil {
		conn.Close()
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		f := strings.SplitN(resp.Status, " ", 2)
		conn.Close()
		return nil, fmt.Errorf(f[1])
	}
	return conn, nil
}

// Redirect Connects to the proxy and Copy from the given connection.
// Using Redirect rather than Dial to save one RTT.
func (p *HTTPProxyClient) Redirect(srcConn *net.TCPConn, targetAddr string) error {
	targetAddr, err := santinizeAddr(targetAddr)
	if err != nil {
		return err
	}
	conn, err := p.connect(targetAddr)
	if err != nil {
		return err
	}
	// Copy Request IO before read 200 OK.
	// So that the proxy server could start transmission faster.
	go CopyIO(conn, srcConn)
	resp, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: http.MethodConnect})
	if err != nil {
		conn.Close()
		return err
	}
	if resp.StatusCode != http.StatusOK {
		f := strings.SplitN(resp.Status, " ", 2)
		conn.Close()
		return fmt.Errorf(f[1])
	}
	// Start to copy other
	go CopyIO(srcConn, conn)
	return nil
}

// RoundTrip request normal http request
func (p *HTTPProxyClient) RoundTrip(req *http.Request) (*http.Response, error) {
	return p.Tr.RoundTrip(req)
}
