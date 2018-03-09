// +build !linux

package client

import (
	"fmt"
	"net"
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
	return nil
}

func (s *EntryRedirectServer) Deploy(blacklist, whitelist []string) error {
	return nil
}

func (s *EntryRedirectServer) Undeploy() error {
	return nil
}
