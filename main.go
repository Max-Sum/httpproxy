package main

import (
	"log"
	"net/http"

	"httpproxy/proxy"
)

func main() {
	pxy := proxy.NewProxyServer()
	web := proxy.NewWebServer()
	pln := proxy.NewProxyListener()
	wln := proxy.NewWebListener()

	go http.Serve(wln, web)
	log.Println("begin proxy")
	log.Fatal(pxy.Serve(pln))
}
