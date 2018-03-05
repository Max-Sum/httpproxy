package client

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

type EntryHTTPServer struct {
	Tr   *HTTPProxyClient
	
}

// NewProxyServer returns a new proxyserver.
func NewProxyServer(client *HTTPProxyClient) *http.Server {
	return &http.Server{
		Handler:        &EntryHTTPServer{Tr: client},
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
}

//ServeHTTP will be automatically called by system.
//ProxyServer implements the Handler interface which need ServeHTTP.
func (proxy *EntryHTTPServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			log.Debug("Panic: %v\n", err)
			fmt.Fprintf(rw, fmt.Sprintln(err))
		}
	}()

	// log.Debug("Host := %v", req.URL.Host)
	//if proxy.Ban(rw, req) {
	//	return
	//}

	if req.Method == "CONNECT" {
		proxy.HttpsHandler(rw, req)
	} else {
		proxy.HttpHandler(rw, req)
	}
}

//HttpHandler handles http connections.
//处理普通的http请求
func (proxy *EntryHTTPServer) HttpHandler(rw http.ResponseWriter, req *http.Request) {
	log.Info("sending request %v %v \n", req.Method, req.Host)
	SanitizeRequest(req)
	RmProxyHeaders(req)

	resp, err := proxy.Tr.RoundTrip(req)
	if err != nil {
		log.Error("%v", err)
		http.Error(rw, err.Error(), 500)
		return
	}
	defer resp.Body.Close()

	ClearHeaders(rw.Header())
	CopyHeaders(rw.Header(), resp.Header)

	rw.WriteHeader(resp.StatusCode) //写入响应状态

	nr, err := io.Copy(rw, resp.Body)
	if err != nil && err != io.EOF {
		log.Error("%v got an error when copy remote response to client.%v\n", proxy.User, err)
		return
	}
	log.Info("copied %v bytes from %v.\n", nr, req.URL.Host)
}

var HTTP_200 = []byte("HTTP/1.1 200 Connection Established\r\n\r\n")

// HttpsHandler handles any connection which need connect method.
// 处理https连接，主要用于CONNECT方法
func (proxy *EntryHTTPServer) HttpsHandler(rw http.ResponseWriter, req *http.Request) {
	log.Info("tried to connect to %v", req.URL.Host)

	hj, _ := rw.(http.Hijacker)
	client, _, err := hj.Hijack() //获取客户端与代理服务器的tcp连接
	if err != nil {
		log.Error("failed to get Tcp connection of \n", req.RequestURI)
		http.Error(rw, "Failed", http.StatusBadRequest)
		return
	}
	err = proxy.Tr.Redirect(client, req.URL.Host)
	if err != nil {
		log.Error("failed to connect %v\n", req.RequestURI)
		// TODO write error msg.
		client.Close()
		return
	}
}