// +build linux
package client

import (
	"context"
    "log"
    "net"
    "time"
    tproxy "github.com/LiamHaworth/go-tproxy"
)

type EntryTproxyServer struct {
	Addr string
	Tr   *HTTPProxyClient
}
// NewEntryServer returns a new proxyserver.
func NewEntryTProxyServer(addr string, client *HTTPProxyClient) *EntryTproxyServer {
	return &EntryTproxyServer{
		Addr: addr,
		Tr: client
	}
}

func (s *EntryTproxyServer) ListenAndServe() error
	addr := s.Addr
	if addr == "" {
		addr = ":3334"
	}
	l, err := tproxy.listen("tcp", s.Addr)
	defer l.Close()
	if err != nil {
		return err
	}
	return s.Serve(l)
}

func (s *EntryTproxyServer) listen(network, address string) (Listener, error) {
	DefaultResolver := &net.Resolver{}
	addrs, err := DefaultResolver.resolveAddrList(context.Background(), "listen", network, address, nil)
	if err != nil {
		return nil, &OpError{Op: "listen", Net: network, Source: nil, Addr: nil, Err: err}
	}
	var l Listener
	switch la := addrs.first(isIPv4).(type) {
	case *TCPAddr:
		l, err = ListenTCP(network, la)
	default:
		return nil, &OpError{Op: "listen", Net: network, Source: nil, Addr: la, Err: &AddrError{Err: "unexpected address type", Addr: address}}
	}
	if err != nil {
		return nil, err
	}
	return l, nil
}

func (s *EntryTproxyServer) Serve(l net.Listener) error {
	for {
		conn, err := l.Accept()
		defer conn.Close()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				log.Errorf("Temporary error while accepting connection: %s", netErr)
			}
			log.Fatalf("Unrecoverable error while accepting connection: %s", err)
			return err
		}
		go handleTCPConn(conn)
	}
}

func (s *EntryTproxyServer) handleTCPConn(conn net.Conn) {
	log.Debugf("Accepting TCP connection from %s with destination of %s", conn.RemoteAddr().String(), conn.LocalAddr().String())
	// LocalAddr is the real remote address.
	// Think about it.
	err := s.Tr.Redirect(conn, conn.LocalAddr().String)
	if err != nil {
		log.Errorf("Failed to connect to original destination [%s]: %s", conn.LocalAddr().String(), err)
		conn.Close()
	}
}