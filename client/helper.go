package client

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"net/http"
	"regexp"
)

var portMap = map[string]string{
	"http":   "80",
	"https":  "443",
	"socks5": "1080",
}


// canonicalAddr returns url.Host but always with a ":port" suffix
func canonicalAddr(url *url.URL) string {
	addr := url.Hostname()
	port := url.Port()
	if port == "" {
		port = portMap[url.Scheme]
	}
	return net.JoinHostPort(addr, port)
}

var addrRegexp *regexp.Regexp
func santinizeAddr(addr string) (string, error) {
	var err error
	if addrRegexp == nil {
		addrRegexp, err = regexp.Compile("^(([^:/\\\\]*)|(\\[[1-9a-f:]*\\]))(\\d(1-5))$")
		if err != nil {
			return "", err
		}
	}
	if !addrRegexp.MatchString(addr) {
		return "", fmt.Errorf("invalid address")
	}
	return addr, nil
}

// CopyIO copies from connection to another
func CopyIO(dst, src net.Conn) {
	defer func() {
		dst.Close()
		src.Close()
	}()

	_, err := io.Copy(dst, src)
	if err != nil && err != io.EOF {
		//log.Error("%v got an error when handles CONNECT %v\n", User, err)
		return
	}
}

// CopyHeaders copy headers from source to destination.
// Nothing would be returned.
func CopyHeaders(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}


func SanitizeRequest(req *http.Request) {
	if req.URL.Scheme == "" {
		req.URL.Scheme = "http"
	}
	if req.URL.Host == "" {
		req.URL.Host = req.Host
	}
}

// ClearHeaders clear headers.
func ClearHeaders(headers http.Header) {
	for key := range headers {
		headers.Del(key)
	}
}

// RmProxyHeaders remove Hop-by-hop headers.
func RmProxyHeaders(req *http.Request) {
	req.RequestURI = ""
	req.Header.Del("Connection")
	req.Header.Del("Proxy-Authenticate")
	req.Header.Del("Proxy-Authorization")
	req.Header.Del("TE")
	req.Header.Del("Trailers")
	req.Header.Del("Transfer-Encoding")
	req.Header.Del("Upgrade")
}
