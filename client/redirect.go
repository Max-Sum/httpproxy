// +build linux

package client

import (
	"fmt"
	"net"
	"errors"
	"syscall"
)

// EntryRedirectServer is an entrypoint for Linux firewall redirection.
type EntryRedirectServer struct {
	Addr  string
	Tr    *HTTPProxyClient
	ln    net.Listener
	conns chan net.Conn
}

// NewEntryRedirectServer returns a new proxyserver.
func NewEntryRedirectServer(addr string, client *HTTPProxyClient) *EntryRedirectServer {
	return &EntryRedirectServer{
		Addr: addr,
		Tr: client,
	}
}

// ListenAndServe on the address specified in config
func (s *EntryRedirectServer) ListenAndServe() error {
	addr := s.Addr
	if addr == "" {
		addr = ":3333"
	}
	ln, err := net.Listen("tcp", s.Addr)
	defer ln.Close()
	if err != nil {
		return err
	}
	return s.Serve(ln)
}

// Serve on the listener
func (s *EntryRedirectServer) Serve(l net.Listener) error {
	// Save the listener
	s.ln = l
	for {
		conn, err := l.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				log.Error("Redirect Entry: Temporary error while accepting connection")
				log.Error(netErr)
			}
			log.Fatal(err)
			return err
		}
		go s.handleTCPConn(conn.(*net.TCPConn))
	}
}

// Shutdown the server gracefully
func (s *EntryRedirectServer) Shutdown() error {
	err := s.ln.Close()
	if err != nil {
		return err
	}
	return nil
}

func (s *EntryRedirectServer) handleTCPConn(conn *net.TCPConn) {
	clientAddr := conn.RemoteAddr().String()
	remoteAddr, err := s.getRemoteAddr(conn)
	if err != nil {
		log.Error("Redir: Failed to get remote address", err)
	}
	addr := net.JoinHostPort(remoteAddr.IP.String(), fmt.Sprint(remoteAddr.Port))
	log.Infof("Redir: Real remote addr %s", addr)
	log.Debugf("Accepting TCP connection from %s with destination of %s", clientAddr, addr)
	err = s.Tr.Redirect(conn, addr)
	if err != nil {
		log.Errorf("Failed to connect to original destination [%s]: %s", addr, err)
		conn.Close()
	}
}

func (s *EntryRedirectServer) getRemoteAddr(c *net.TCPConn) (*net.TCPAddr, error) {
	// test if the underlying fd is nil
	remoteAddr := c.RemoteAddr()
	if remoteAddr == nil {
		return nil, errors.New("getRemoteAddr: Underlying FileDescriptor is nil")
	}
	fc, err := c.File()
	if err != nil {
		return nil, err
	}
	defer fc.Close()
	fd := fc.Fd()
	// The File() call above puts both the original socket c and the file fd in blocking mode.
	// Set the file fd back to non-blocking mode and the original socket c will become non-blocking as well.
	// Otherwise blocking I/O will waste OS threads.
	if err := syscall.SetNonblock(int(fd), true); err != nil {
		return nil, err
	}
	// Get Actual Address
	mreq, err := syscall.GetsockoptIPv6Mreq(int(fd), syscall.IPPROTO_IP, 80)
	if err != nil {
		return nil, err
	}
	// TCPv4
	ip := net.IPv4(mreq.Multiaddr[4], mreq.Multiaddr[5], mreq.Multiaddr[6], mreq.Multiaddr[7])
	port := uint16(mreq.Multiaddr[2])<<8 + uint16(mreq.Multiaddr[3])
	
	return &net.TCPAddr{IP:ip, Port:int(port)}, nil
}