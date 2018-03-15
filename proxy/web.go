package proxy

import (
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"httpproxy/config"
)

type WebServer struct{}

func NewWebServer() *WebServer {
	return &WebServer{}
}

// ServeHTTP handles web admin pages
func (ws *WebServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if err := ws.WebAuth(rw, req); err != nil {
		log.Debug("%v", err)
		return
	}

	if req.URL.Path == "/" {
		ws.HomeHandler(rw, req)
		return
	} else {
		p := strings.Trim(req.URL.Path, "/")
		s := strings.Split(p, "/")
		switch s[0] {
		case "static":
			hadler := http.FileServer(http.Dir("."))
			hadler.ServeHTTP(rw, req)
		case "user":
			ws.UserHandler(rw, req)
		case "setting":
			ws.SettingHandler(rw, req)
		}
	}
}

type data struct {
	config.Config
	Nav string
}

// HomeHandler handles web home page
func (ws *WebServer) HomeHandler(rw http.ResponseWriter, req *http.Request) {
	t := template.New("layout.tpl")
	t, err := t.ParseFiles("views/layout.tpl", "views/home.tpl")
	if err != nil {
		log.Error(err)
		http.Error(rw, "tpl error", 500)
		return
	}
	Data := data{cnfg, "home"}
	err = t.Execute(rw, Data)
	if err != nil {
		log.Error(err)
		http.Error(rw, "tpl error", 500)
		return
	}
}

// UserHandler handles user-list page
func (ws *WebServer) UserHandler(rw http.ResponseWriter, req *http.Request) {
	p := strings.Trim(req.URL.Path, "/")
	s := strings.Split(p, "/")
	if len(s) < 3 {
		http.Error(rw, "request error", 500)
		return
	}

	user := s[2]
	switch s[1] {
	case "list": //list all users
		t := template.New("layout.tpl")
		t, err := t.ParseFiles("views/layout.tpl", "views/user.tpl")
		if err != nil {
			log.Error(err)
			http.Error(rw, "tpl error", 500)
			return
		}
		Data := data{cnfg, "user"}
		err = t.Execute(rw, Data)
		if err != nil {
			log.Error(err)
			http.Error(rw, "tpl error", 500)
			return
		}
	case "modify": //modify specific user
		passwd := req.FormValue("passwd")
		if passwd != "" {
			cnfg.User[user] = passwd
		}
	case "delete": //delete specific user
		delete(cnfg.User, user)
	case "add": //add new user
		user := req.FormValue("user")
		passwd := req.FormValue("passwd")
		if cnfg.User[user] != "" {
			http.Error(rw, "post error", 500)
			return
		}
		cnfg.User[user] = passwd
	}
	err := cnfg.WriteToFile()
	if err != nil {
		log.Error(err)
	}
}

// SettingHandler allows admin modifies proxy's setting.
func (ws *WebServer) SettingHandler(rw http.ResponseWriter, req *http.Request) {
	p := strings.Trim(req.URL.Path, "/")
	s := strings.Split(p, "/")
	if len(s) < 2 {
		http.Error(rw, "request error", 500)
		return
	}
	switch s[1] {
	case "list":
		t := template.New("layout.tpl")
		t, err := t.ParseFiles("views/layout.tpl", "views/setting.tpl")
		if err != nil {
			log.Error(err)
			http.Error(rw, "tpl error", 500)
			return
		}
		Data := data{cnfg, "setting"}
		err = t.Execute(rw, Data)
		if err != nil {
			log.Error(err)
			http.Error(rw, "tpl error", 500)
			return
		}
	case "set":
		auth := req.FormValue("auth")
		cache := req.FormValue("cache")
		cachetimeout := req.FormValue("cachetimeout")
		failover := req.FormValue("failover")
		gfwlist := req.FormValue("gfwlist")
		logging, _ := strconv.Atoi(req.FormValue("log"))
		//TODO check those value
		if auth == "true" {
			cnfg.Auth = true
		}
		if cache == "true" {
			cnfg.Cache = true
		}
		ctint, _ := strconv.Atoi(cachetimeout)
		cnfg.CacheTimeout = int64(ctint)
		gfwlist = strings.Trim(gfwlist, ";")
		cnfg.GFWList = strings.Split(gfwlist, ";")
		cnfg.Failover = failover
		cnfg.Log = logging
		err := cnfg.WriteToFile()
		if err != nil {
			log.Error(err)
		}
		rw.WriteHeader(http.StatusOK)
	}
}

// WebAuth checks the authorization
func (ws *WebServer) WebAuth(rw http.ResponseWriter, req *http.Request) error {
	_, passwd, ok := req.BasicAuth()
	if !ok {
		err := NeedAuth(rw, HTTP_401)
		log.Debug(err)
		return errors.New("Need Authorization")
	}

	if passwd != cnfg.AdminPass {
		NeedAuth(rw, HTTP_401)
		return errors.New(req.RemoteAddr + "Fail to log in")
	}
	return nil
}

var HTTP_401 = []byte("HTTP/1.1 401 Authorization Required\r\nWWW-Authenticate: Basic realm=\"Secure Web\"\r\n\r\n")
