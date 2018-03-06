package main

import (
	"os"
	"os/signal"
    "syscall"
	"log"
	"net/http"

	"httpproxy/proxy"
)

func main() {
	pxy := proxy.NewProxyServer()
	web := proxy.NewWebServer()
	pln := proxy.NewProxyListener()
	wln := proxy.NewWebListener()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<- sigs
		pxy.Close()
		pln.Close()
		wln.Close()
		log.Println("Close socket")
		os.Exit(0)
	}()

	go http.Serve(wln, web)
	log.Println("begin proxy")
	log.Fatal(pxy.Serve(pln))
}
