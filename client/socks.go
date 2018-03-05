// +build !windows

package client

import (
	"errors"
    "net"
	socks "github.com/fangdingjun/socks-go"
)

type EntrySocksServer struct {
	Addr string
	Tr *HTTPProxyClient
	ln net.Listener
}
// NewEntryServer returns a new proxyserver.
func NewEntrySocksServer(addr string, client *HTTPProxyClient) *EntrySocksServer {
	return &EntrySocksServer{
		Addr: addr,
		Tr: client,
	}
}

func (s *EntrySocksServer) ListenAndServe() error {
	addr := s.Addr
	if addr == "" {
		addr = ":1080"
	}
	l, err := net.Listen("tcp", s.Addr)
	defer l.Close()
	if err != nil {
		return err
	}
	return s.Serve(l)
}

// Serve on the Listener
func (s *EntrySocksServer) Serve(l net.Listener) error {
	// Save the listener
	s.ln = l
    for {
		conn, err := l.Accept()
        if err != nil {
            log.Error(err)
            continue
        }
        log.Debug("connected from %s", conn.RemoteAddr())
        c := socks.Conn{Conn: conn, Dial: s.dial}
        go c.Serve()
    }
}

// Shutdown the redirect server gracefully
func (s *EntrySocksServer) Shutdown() error {
	err := s.ln.Close()
	if err != nil {
		return err
	}
	return nil
}

func (s *EntrySocksServer) dial(network, addr string) (net.Conn, error) {
	if (network == "tcp" || network == "tcp4" || network == "tcp6") {
		return s.Tr.Dial(addr)
	}
	return nil, errors.New("Socks Entry: Not a TCP connection")
}