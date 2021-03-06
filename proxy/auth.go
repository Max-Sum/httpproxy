package proxy

import (
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"strings"
)

var HTTP_407 = []byte("HTTP/1.1 407 Proxy Authorization Required\r\nProxy-Authenticate: Basic realm=\"Secure Proxys\"\r\n\r\n")

//Auth provides basic authorizaton for proxy server.
func (proxy *Handler) Auth(rw http.ResponseWriter, req *http.Request) bool {
	var err error
	if cnfg.Reverse == false && cnfg.Auth == true { //代理服务器登入认证
		if proxy.User, err = proxy.auth(rw, req); err != nil {
			log.Debug(err)
			return true
		}
	} else {
		proxy.User = "Anonymous"
	}

	return false
}

//Auth provides basic authorizaton for proxy server.
func (proxy *Handler) auth(rw http.ResponseWriter, req *http.Request) (string, error) {
	auth := req.Header.Get("Proxy-Authorization")
	auth = strings.Replace(auth, "Basic ", "", 1)

	if auth == "" {
		AuthFailover(rw, req)
		return "", errors.New("Need Proxy Authorization!")
	}
	data, err := base64.StdEncoding.DecodeString(auth)
	if err != nil {
		log.Debug("when decoding %v, got an error of %v", auth, err)
		return "", errors.New("Fail to decoding Proxy-Authorization")
	}

	userPasswdPair := strings.Split(string(data), ":")
	if len(userPasswdPair) != 2 {
		AuthFailover(rw, req)
		return "", errors.New("Fail to log in")
	}
	if Check(userPasswdPair[0], userPasswdPair[1]) == false {
		AuthFailover(rw, req)
		return "", errors.New("Fail to log in")
	}
	return userPasswdPair[0], nil
}

func NeedAuth(rw http.ResponseWriter, challenge []byte) error {
	hj, _ := rw.(http.Hijacker)
	Client, _, err := hj.Hijack()
	if err != nil {
		return errors.New("Fail to get Tcp connection of Client")
	}
	defer Client.Close()

	Client.Write(challenge)
	return nil
}


//反向代理到外部服务器，模仿其行为
func AuthFailover(rw http.ResponseWriter, req *http.Request) {
	// Send 407 if no failover is set
	if cnfg.Failover == "" {
		NeedAuth(rw, HTTP_407)
		return
	}

	remote, err := net.Dial("tcp", cnfg.Failover) //建立failover和代理服务器的tcp连接
	if err != nil {
		log.Errorf("%s failed set up a connection to failover server.", "Unauthorized")
		http.Error(rw, "Failed", http.StatusBadRequest)
		return
	}
	hj, _ := rw.(http.Hijacker)
	client, _, err := hj.Hijack() //获取客户端与代理服务器的tcp连接
	if err != nil {
		log.Errorf("%s failed to get Tcp connection of %s", "Unauthorized", req.RequestURI)
		http.Error(rw, "Failed", http.StatusBadRequest)
		return
	}
	// 将请求发送到 failover 服务器
	req.Write(remote)
	go copyRemoteToClient("Unauthorized", remote, client)
	go copyRemoteToClient("Unauthorized", client, remote)
}

// Check checks username and password
func Check(user, passwd string) bool {
	if user != "" && passwd != "" && cnfg.User[user] == passwd {
		return true
	}
	return false
}
