package main

import (
	"os"
	"os/signal"
    "syscall"
	"log"
	"flag"
	"net/http"

	"httpproxy/proxy"
	"httpproxy/config"
)

var (
	cnfg config.Config
)

func main() {
	// Parse arguments
	configPtr := flag.String("c", "config/config.json", "config file")
	flag.Parse()
	// Read config file
	err := cnfg.GetConfig(*configPtr)
	if err != nil {
		log.Fatal(err)
	}
	proxy.Initialize(cnfg)
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
