package client

import (
    socks "github.com/fangdingjun/socks-go"
    "log"
    "net"
    "time"
)

type EntrySocksServer struct {
	Addr string
	Tr *HTTPProxyClient
}
// NewEntryServer returns a new proxyserver.
func NewEntryServer(addr string, client *HTTPProxyClient) *EntrySocksServer {
	return &EntrySocksServer{
		Addr: addr
		Tr: client
	}
}

func (s *EntrySocksServer) ListenAndServe() error
	addr := s.Addr
	if addr == "" {
		addr = ":1080"
	}
	l, err := tproxy.listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	return s.Serve(l)
}

func (s *EntrySocksServer) Serve(l net.Listener) error
	defer l.Close()

    for {
        conn, err := l.Accept()
        if err != nil {
            log.Println(err)
            continue
        }

        log.Debug("connected from %s", c.RemoteAddr())

        c := socks.Conn{Conn: conn, Dial: s.dial}
        go c.Serve()
    }
}

func (s *EntrySocksServer) dial(network, addr string) (net.Conn, error) {
	if network == "tcp" || network == "tcp4" || network == "tcp6" {
		return s.Tr.Dial(addr)
	}
}