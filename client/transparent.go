// +build linux

package client

import (
	"fmt"
    "net"
	tproxy "github.com/LiamHaworth/go-tproxy"
)

type EntryTproxyServer struct {
	Addr string
	Tr   *HTTPProxyClient
	ln   net.Listener
}
// NewEntryTProxyServer returns a new proxyserver.
func NewEntryTProxyServer(addr string, client *HTTPProxyClient) *EntryTproxyServer {
	return &EntryTproxyServer{
		Addr: addr,
		Tr: client,
	}
}

func (s *EntryTproxyServer) ListenAndServe() error {
	addr := s.Addr
	if addr == "" {
		addr = ":3334"
	}
	l, err := s.listen("tcp", s.Addr)
	defer l.Close()
	if err != nil {
		return err
	}
	return s.Serve(l)
}

func (s *EntryTproxyServer) listen(network, address string) (net.Listener, error) {
	if (network == "tcp" || network == "tcp4" || network == "tcp6") {
		laddr, err := net.ResolveTCPAddr(network, address)
		if err != nil {
			return nil, err
		}
		return tproxy.ListenTCP(network, laddr)
	}
	return nil, fmt.Errorf("TProxy Entry: Not supported network: %s", network)
}

// Serve at the listener.
func (s *EntryTproxyServer) Serve(l net.Listener) error {
	// Save the listener
	s.ln = l
	for {
		conn, err := l.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				log.Errorf("Temporary error while accepting connection: %s", netErr)
			}
			log.Fatalf("Unrecoverable error while accepting connection: %s", err)
			return err
		}
		go s.handleTCPConn(conn)
	}
}

// Shutdown the redirect server gracefully
func (s *EntryTproxyServer) Shutdown() error {
	err := s.ln.Close()
	if err != nil {
		return err
	}
	return nil
}

func (s *EntryTproxyServer) handleTCPConn(conn net.Conn) {
	log.Debugf("Accepting TCP connection from %s with destination of %s", conn.RemoteAddr().String(), conn.LocalAddr().String())
	// LocalAddr is the real remote address.
	// Think about it.
	err := s.Tr.Redirect(conn.(*net.TCPConn), conn.LocalAddr().String())
	if err != nil {
		log.Errorf("Failed to connect to original destination [%s]: %s", conn.LocalAddr().String(), err)
		conn.Close()
	}
}