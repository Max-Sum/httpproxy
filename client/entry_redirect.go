// +build !windows

package client

import (
	"fmt"
	"net"
)

type EntryRedirectServer struct {
	Addr string
	Tr *HTTPProxyClient
}

// NewEntryServer returns a new proxyserver.
func NewEntryRedirectServer(addr string, client *HTTPProxyClient) *EntryRedirectServer {
	return &EntryRedirectServer{
		Addr: addr
		Tr: client
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

func (s *EntryRedirectServer) Serve(l net.Listener) error {
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

func (s *EntryRedirectServer) handleTCPConn(conn net.Conn) {
	clientAddr := conn.RemoteAddr().String()
	remoteAddr := s.getRemoteAddr(conn)
	log.Debugf("Accepting TCP connection from %s with destination of %s", clientAddr, remoteAddr)
	err := s.Tr.Redirect(conn, remoteAddr)
	if err != nil {
		log.Errorf("Failed to connect to original destination [%s]: %s", remoteAddr, err)
		conn.Close()
	}
}

func (s *EntryRedirectServer) getRemoteAddr(c net.Conn) string {
	// test if the underlying fd is nil
	remoteAddr := c.RemoteAddr()
	if remoteAddr == nil {
		//err = errors.New("getOriginalDstAddr: conn.fd is nil")
		return nil
	}
	// net.TCPConn.File() will cause the receiver's (clientConn) socket to be placed in blocking mode.
	// The workaround is to take the File returned by .File(), do getsockopt() to get the original
	// destination, then create a new *net.TCPConn by calling net.Conn.FileConn().  The new TCPConn
	// will be in non-blocking mode. What a pain.
	fc, err := c.File()
	defer fc.Close()
	if err != nil {
		return nil
	} else {
		c.Close()
	}
	// Recreate TCPConn
	cc, err := net.FileConn(fc)
	if err != nil {
		return nil
	}
	conn, ok := cc.(*net.TCPConn)
	if !ok {
		//err = errors.New("not a TCP connection")
		return nil
	} else {
		c = conn
	}
	// Get Actual Address
	mreq, err := syscall.GetsockoptIPv6Mreq(int(fc.Fd()), syscall.IPPROTO_IP, SO_ORIGINAL_DST)
	if err != nil {
		return nil
	}
	var addr string
	// Simple comparasion to get network type.
	if Equal(mreq.Multiaddr[8:], [8]byte{}) {
		// TCPv4
		ip := net.IPv4(mreq.Multiaddr[4], mreq.Multiaddr[5], mreq.Multiaddr[6], mreq.Multiaddr[7])
		port := uint16(mreq.Multiaddr[2])<<8 + uint16(mreq.Multiaddr[3])
		addr = net.JoinHostPort(ip.String(), port)
	} else {
		// TCPv6
		ip := (net.IP)(mreq.Multiaddr[4:])
		port := uint16(mreq.Multiaddr[2])<<8 + uint16(mreq.Multiaddr[3])
		addr = fmt.JoinHostPort(ip.String(), port)
	}
	
	return addr
}