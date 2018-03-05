package proxy

import (
	"net/http"
)

//ReverseHandler handles request for reverse proxy.
//处理反向代理请求
func (proxy *Handler) ReverseHandler(req *http.Request) {
	if cnfg.Reverse == true { //用于反向代理
		proxy.reverseHandler(req)
	}
}

//ReverseHandler handles request for reverse proxy.
//处理反向代理请求
func (proxy *Handler) reverseHandler(req *http.Request) {
	req.Host = cnfg.ProxyPass
	req.URL.Host = req.Host
	req.URL.Scheme = "http"
	log.Debug("%v", req.RequestURI)
}
