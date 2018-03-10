package client

import (
	"io"
	"net"
	"net/url"
	"net/http"
	"time"
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

type closeRead interface {
	CloseRead() error
}
type closeWrite interface {
	CloseWrite() error
}

// CopyIO copies from connection to another
// And Close them when things ends.
func CopyIO(dst, src net.Conn, terminate chan bool) {
	defer func() {
		// The first goroutine will only try to half close
		// The second goroutine close both forcefully.
		select {
		// lock
		case terminate <- true:
			log.Debugf("CopyIO: HalfClose %s <-> %s", dst.RemoteAddr(), src.RemoteAddr())
			if cw, ok := dst.(closeWrite); ok {
				cw.CloseWrite()
			}
			if cr, ok := src.(closeRead); ok {
				cr.CloseRead()
			}
		default:
			log.Debugf("CopyIO: Close %s <-> %s", dst.RemoteAddr(), src.RemoteAddr())
			dst.Close()
			src.Close()
			// unlock
			<- terminate
			close(terminate)
			return
		}
		// Wait for the second goroutine to close the channel
		select {
		case terminate <- true:
		case <- time.After(5 * time.Second):
			log.Debugf("CopyIO: Force Close %s <-> %s", dst.RemoteAddr(), src.RemoteAddr())
			dst.Close()
			src.Close()
		}
	}()
	bytes, err := io.Copy(dst, src)
	if err != nil && err != io.EOF {
		log.Warningf("Got an error when copying %v", err)
		return
	}
	log.Infof("CopyIO: copied %d bytes %s to %s", bytes, src.RemoteAddr(), dst.RemoteAddr())
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
	req.Header.Del("X-Forwarded-For")
	req.Header.Del("X-Forwarded-By")
	req.Header.Del("X-Forwarded-Proto")
	req.Header.Del("X-Forwarded-X")
	req.Header.Del("Forwarded")
	req.Header.Del("Forwarded-For")
	req.Header.Del("Forwarded-By")
	req.Header.Del("Keep-Alive")
	req.Header.Del("Proxy-Authenticate")
	req.Header.Del("Proxy-Authorization")
	req.Header.Del("TE")
	req.Header.Del("Trailers")
	req.Header.Del("Transfer-Encoding")
	req.Header.Del("Upgrade")
}
