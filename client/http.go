package client

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

type entryHTTPHandler struct {
	Tr   *HTTPProxyClient
}

// EntryHTTPServer is normal HTTP entrypoint.
type EntryHTTPServer struct {
	*http.Server
}

// NewEntryHTTPServer returns a new proxyserver.
func NewEntryHTTPServer(addr string, client *HTTPProxyClient) *EntryHTTPServer {
	return &EntryHTTPServer{&http.Server{
		Addr:           addr,
		Handler:        &entryHTTPHandler{Tr: client},
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}}
}

//ServeHTTP will be automatically called by system.
//ProxyServer implements the Handler interface which need ServeHTTP.
func (proxy *entryHTTPHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			log.Debugf("HTTP Entry: %v", err)
			fmt.Fprintf(rw, fmt.Sprintln(err))
		}
	}()

	// log.Debug("Host := %v", req.URL.Host)
	//if proxy.Ban(rw, req) {
	//	return
	//}

	if req.Method == "CONNECT" {
		proxy.HTTPSHandler(rw, req)
	} else {
		proxy.HTTPHandler(rw, req)
	}
}

//HttpHandler handles http connections.
//处理普通的http请求
func (proxy *entryHTTPHandler) HTTPHandler(rw http.ResponseWriter, req *http.Request) {
	log.Debugf("HTTP Entry: Sending request %s %s", req.Method, req.Host)
	SanitizeRequest(req)
	RmProxyHeaders(req)

	resp, err := proxy.Tr.RoundTrip(req)
	if err != nil {
		log.Errorf("HTTP Entry: %v", err)
		http.Error(rw, err.Error(), 500)
		return
	}
	defer resp.Body.Close()

	ClearHeaders(rw.Header())
	CopyHeaders(rw.Header(), resp.Header)

	rw.WriteHeader(resp.StatusCode) //写入响应状态

	nr, err := io.Copy(rw, resp.Body)
	if err != nil && err != io.EOF {
		log.Errorf("HTTP Entry: %v", err)
		return
	}
	log.Info("copied %v bytes from %v.\n", nr, req.URL.Host)
}

var http200 = []byte("HTTP/1.1 200 Connection Established\r\n\r\n")
// HTTPSHandler handles any connection which need connect method.
// 处理https连接，主要用于CONNECT方法
func (proxy *entryHTTPHandler) HTTPSHandler(rw http.ResponseWriter, req *http.Request) {
	log.Debugf("HTTP Entry: Tried to connect to %s", req.URL.Host)

	hj, _ := rw.(http.Hijacker)
	client, _, err := hj.Hijack() //获取客户端与代理服务器的tcp连接
	defer client.Close()
	if err != nil {
		log.Errorf("HTTP Entry: Failed to get TCP connection of %s", req.RequestURI)
		log.Error(err)
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	
	// 提前发送200，减少RTT时间
	client.Write(http200)
	err = proxy.Tr.Redirect(client.(*net.TCPConn), req.URL.Host)
	if err != nil {
		log.Errorf("HTTP Entry: fFiled to connect to %s", req.RequestURI)
		log.Error(err)
		// TODO write error msg.
		client.Close()
		return
	}
}
