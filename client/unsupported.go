// +build !linux

package client

import (
	"fmt"
	"net"
	"time"
	"io"
)

// EntrypTproxyServer is a null struct on non-Linux system
type EntryTproxyServer struct {
	Addr string
	Tr   *HTTPProxyClient
}

// EntrypTproxyServer is a null struct on non-Linux system
type EntryRedirectServer struct {
	Addr string
	Tr   *HTTPProxyClient
}

type BogusDNS struct {
	IPPrefix    net.IP
	DNSTTL      time.Duration // TTL to reply the request
	TTL         time.Duration // TTL to stay in the map
	IPIndex     [0]*struct{}
}

func NewEntryTProxyServer(addr string, client *HTTPProxyClient) *EntryTproxyServer {
	return &EntryTproxyServer{}
}

func (s *EntryTproxyServer) ListenAndServe() error {
	return fmt.Errorf("TProxy Server is not supported on this platform.")
}

func (s *EntryTproxyServer) Serve(l net.Listener) error {
	return fmt.Errorf("TProxy Server is not supported on this platform.")
}

func (s *EntryTproxyServer) Shutdown() error {
	return nil
}

func NewEntryRedirectServer(addr string, client *HTTPProxyClient) *EntryRedirectServer {
	return &EntryRedirectServer{}
}

func (s *EntryRedirectServer) ListenAndServe() error {
	return fmt.Errorf("Redirect Server is not supported on this platform.")
}

func (s *EntryRedirectServer) Serve(l net.Listener) error {
	return fmt.Errorf("Redirect Server is not supported on this platform.")
}

func (s *EntryRedirectServer) Shutdown() error {
	return fmt.Errorf("Redirect Server is not supported on this platform.")
}

func (s *EntryRedirectServer) Deploy(blacklist, whitelist []string) error {
	return fmt.Errorf("Redirect Server is not supported on this platform.")
}

func (s *EntryRedirectServer) Undeploy() error {
	return fmt.Errorf("Redirect Server is not supported on this platform.")
}

func NewBogusDNS(addr string, prefix net.IP, ttl time.Duration) *BogusDNS {
	return nil
}

func (s *BogusDNS) ListenAndServe() error {
	return fmt.Errorf("BogusDNS is not supported on this platform")
}

// Shutdown the server gracefully
func (s *BogusDNS) Shutdown() error {
	return fmt.Errorf("BogusDNS is not supported on this platform")
}

// Close the server forcefully
func (s *BogusDNS) Close() error {
	return fmt.Errorf("BogusDNS is not supported on this platform")
}

// GetIP address by the given address
func (s *BogusDNS) GetIP(domain string) (net.IP, error) {
	return nil, fmt.Errorf("BogusDNS is not supported on this platform")
}

// GetAddress from the IP address given
// It can be void
func (s *BogusDNS) GetAddress(ip net.IP) (string, error) {
	return "", fmt.Errorf("BogusDNS is not supported on this platform")
}

// WriteDNSMasqConfig gives a dnsmasq config that will
// make dnsmasq to request Bogus DNS Server in case of
// gfwlist matches.
// It needs GFWList to be established.
func (s *BogusDNS) WriteDNSMasqConfig(w io.Writer, blacklist []string) error {
	return fmt.Errorf("BogusDNS is not supported on this platform")
}
