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
	Addr string
	Tr *HTTPProxyClient
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
	for {
		conn, err := l.Accept()
		defer conn.Close()
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

func (s *EntryRedirectServer) handleTCPConn(conn *net.TCPConn) {
	clientAddr := conn.RemoteAddr().String()
	remoteAddr, err := s.getRemoteAddr(conn)
	log.Debugf("Accepting TCP connection from %s with destination of %s", clientAddr, remoteAddr)
	err = s.Tr.Redirect(conn, remoteAddr)
	if err != nil {
		log.Errorf("Failed to connect to original destination [%s]: %s", remoteAddr, err)
		conn.Close()
	}
}

func (s *EntryRedirectServer) getRemoteAddr(c *net.TCPConn) (string, error) {
	// test if the underlying fd is nil
	remoteAddr := c.RemoteAddr()
	if remoteAddr == nil {
		return "", errors.New("getRemoteAddr: Underlying FileDescriptor is nil")
	}
	// net.TCPConn.File() will cause the receiver's (clientConn) socket to be placed in blocking mode.
	// The workaround is to take the File returned by .File(), do getsockopt() to get the original
	// destination, then create a new *net.TCPConn by calling net.Conn.FileConn().  The new TCPConn
	// will be in non-blocking mode. What a pain.
	fc, err := c.File()
	defer fc.Close()
	if err != nil {
		return "", err
	}
	c.Close()
	// Recreate TCPConn
	cc, err := net.FileConn(fc)
	if err != nil {
		return "", err
	}
	conn, ok := cc.(*net.TCPConn)
	if !ok {
		err = errors.New("getRemoteAddr: not a TCP connection")
		return "", err
	}
	c = conn
	// Get Actual Address
	mreq, err := syscall.GetsockoptIPv6Mreq(int(fc.Fd()), syscall.IPPROTO_IP, 80)
	if err != nil {
		return "", err
	}
	var addr string
	// TCPv4
	ip := net.IPv4(mreq.Multiaddr[4], mreq.Multiaddr[5], mreq.Multiaddr[6], mreq.Multiaddr[7])
	port := uint16(mreq.Multiaddr[2])<<8 + uint16(mreq.Multiaddr[3])
	addr = net.JoinHostPort(ip.String(), fmt.Sprint(port))
	
	return addr, nil
}