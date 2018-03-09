// +build linux

package client

import (
	"fmt"
	"net"
	"errors"
	"syscall"

	"github.com/coreos/go-iptables/iptables"
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
	s.Undeploy()
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

// Deploy the iptables rules to
// redirect connections to redirect server
func (s *EntryRedirectServer) Deploy(blacklist, whitelist []string) error {
	CHAIN := "HTTPPROXY-REDIR"
	_, p, err := net.SplitHostPort(s.Addr)
	if err != nil {
		return err
	}
	ipt, err := iptables.New()
	if err != nil {
		return err
	}
	// Check if any left chains
	chains, err := ipt.ListChains("nat")
	if err != nil {
		return err
	}
	for _, c := range chains {
		if c == CHAIN {
			s.Undeploy()
		}
	}
	if err = ipt.NewChain("nat", CHAIN); err != nil {
		return err
	}
	// whitelist ips will returns
	for _, rule := range whitelist {
		if err = ipt.AppendUnique("nat", CHAIN, "-d", rule, "-j", "RETURN"); err != nil {
			return err
		}
	}
	// blacklist ips will be redirected
	for _, rule := range blacklist {
		if err = ipt.AppendUnique("nat", CHAIN, "-p", "tcp", "-d", rule, "-j", "REDIRECT", "--to-ports", p); err != nil {
			return err
		}
	}
	// append chain into prerouting and output
	if err = ipt.AppendUnique("nat", "PREROUTING", "-p", "tcp", "-j", CHAIN); err != nil {
		return err
	}
	if err = ipt.AppendUnique("nat", "OUTPUT", "-p", "tcp", "-j", CHAIN); err != nil {
		return err
	}
	return nil
}

// Undeploy removes the iptables rules set using Deploy
// It tolerates errors
func (s *EntryRedirectServer) Undeploy() error {
	CHAIN := "HTTPPROXY-REDIR"
	ipt, err := iptables.New()
	if err != nil {
		return err
	}
	if err = ipt.ClearChain("nat", CHAIN); err != nil {
		log.Error(err)
	}
	if err = ipt.Delete("nat", "PREROUTING", "-p", "tcp", "-j", CHAIN); err != nil {
		log.Error(err)
	}
	if err = ipt.Delete("nat", "OUTPUT", "-p", "tcp", "-j", CHAIN); err != nil {
		log.Error(err)
	}
	if err = ipt.DeleteChain("nat", CHAIN); err != nil {
		log.Error(err)
	}
	return err
}