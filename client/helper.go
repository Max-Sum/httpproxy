package client

import (
	"fmt"
	"io"
	"net"
	"net/url"
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
		addrRegexp, err = regexp.Compile("^([^:]*):(\\d(1-5))$")
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
