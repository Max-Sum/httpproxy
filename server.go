package main

import (
	"os"
	"os/signal"
    "syscall"
	"log"
	"flag"
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

func parseArguments() {
	// Parse arguments
	config := flag.String("c", "config/client.json", "config file")
	listen := flag.String("l", "", "listening address")
	webListen := flag.String("w", "", "web listening address")
	reverse := flag.String("r", "", "reverse proxy to")
	auth := flag.Bool("a", false, "if the proxy is going the check auth")
	failover := flag.String("r", "", "reverse proxy to")
	verbose := flag.Int("v", 1, "level of verbose")
	flag.Parse()
}