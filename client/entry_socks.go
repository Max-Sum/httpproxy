package client

import (
    socks "github.com/fangdingjun/socks-go"
    "log"
    "net"
    "time"
)

type EntrySocksServer struct {
	Tr *HTTPProxyClient
}
// NewEntryServer returns a new proxyserver.
func NewEntryServer(client *HTTPProxyClient) *EntrySocksServer {
	return &EntrySocksServer{
		Tr: client
	}
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

        d := net.Dialer{Timeout: 10 * time.Second}
        c := socks.Conn{Conn: conn, Dial: s.dial}
        go c.Serve()
    }
}

func (s *EntrySocksServer) dial(network, addr string) (net.Conn, error) {
	if network == "tcp" || network == "tcp4" || network == "tcp6" {
		return s.Tr.Dial(addr)
	}
}