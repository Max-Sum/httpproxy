// Package proxy implements a http proxy.
//
// Support GET, POST, CONNECT method and so on.
// Support proxy auth and web management.
// Support web cache.
package proxy

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"time"
	"httpproxy/cache"
)

// Handler is the main structure
type Handler struct {
	// User records user's name
	Tr   *http.Transport
	User string
}

// NewProxyServer returns a new proxyserver.
func NewProxyServer() *http.Server {
	if cnfg.Cache {
		RegisterCacheBox(cache.NewCacheBox(":6379", ""))
	}

	return &http.Server{
		Handler:        &Handler{Tr: &http.Transport{Proxy: http.ProxyFromEnvironment}},
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
}

//ServeHTTP will be automatically called by system.
//Server implements the Handler interface which need ServeHTTP.
func (proxy *Handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			log.Debug("Panic: %v\n", err)
			fmt.Fprintf(rw, fmt.Sprintln(err))
		}
	}()

	// log.Debug("Host := %v", req.URL.Host)

	if proxy.Auth(rw, req) {
		return
	}

	proxy.ReverseHandler(req)

	if proxy.Ban(rw, req) {
		return
	}

	if req.Method == "CONNECT" {
		boost := req.Header.Get("X-Proxy-Boost") != "boosted"
		proxy.HttpsHandler(rw, req, boost)
	} else if cnfg.Cache == true && req.Method == "GET" {
		proxy.CacheHandler(rw, req)
	} else {
		proxy.HttpHandler(rw, req)
	}
}

//HttpHandler handles http connections.
//处理普通的http请求
func (proxy *Handler) HttpHandler(rw http.ResponseWriter, req *http.Request) {
	log.Info("%v is sending request %v %v \n", proxy.User, req.Method, req.Host)
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
	log.Info("%v copied %v bytes from %v.\n", proxy.User, nr, req.URL.Host)
}

var HTTP_200 = []byte("HTTP/1.1 200 Connection Established\r\n\r\n")

// HttpsHandler handles any connection which need connect method.
// 处理https连接，主要用于CONNECT方法
func (proxy *Handler) HttpsHandler(rw http.ResponseWriter, req *http.Request, boost200 bool) {
	log.Info("%v tried to connect to %v", proxy.User, req.URL.Host)

	hj, _ := rw.(http.Hijacker)
	client, _, err := hj.Hijack() //获取客户端与代理服务器的tcp连接
	if err != nil {
		log.Error("%v failed to get Tcp connection of \n", proxy.User, req.RequestURI)
		http.Error(rw, "Failed", http.StatusBadRequest)
		return
	}
	if boost200 {
		// 提前发送200，减少RTT时间
		client.Write(HTTP_200)
	}
	remote, err := net.Dial("tcp", req.URL.Host) //建立服务端和代理服务器的tcp连接
	if err != nil {
		log.Error("%v failed to connect %v\n", proxy.User, req.RequestURI)
		// If 200 is not sent, we can report the error to client.
		if !boost200 {
			resp := &http.Response{
				StatusCode:      502,
				ProtoMajor:      req.ProtoMajor,
				ProtoMinor:      req.ProtoMinor,
				Header:          make(http.Header),
				Body:            ioutil.NopCloser(bytes.NewBufferString(err.Error())),
				ContentLength:   int64(len(err.Error())),
				Close:           true,
				Request:         req,
			}
			if err.(*net.OpError).Timeout() {
				resp.StatusCode = 504
			}
			resp.Write(client)
		}
		client.Close()
		return
	}
	// Write 200 after getting connection.
	if !boost200 {
		client.Write(HTTP_200)
	}

	go copyRemoteToClient(proxy.User, remote, client)
	go copyRemoteToClient(proxy.User, client, remote)
}

func copyRemoteToClient(User string, remote, client net.Conn) {
	defer func() {
		remote.Close()
		client.Close()
	}()

	nr, err := io.Copy(remote, client)
	if err != nil && err != io.EOF {
		//log.Error("%v got an error when handles CONNECT %v\n", User, err)
		return
	}
	log.Info("%v transported %v bytes betwwen %v and %v.\n", User, nr, remote.RemoteAddr(), client.RemoteAddr())
}
