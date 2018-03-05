package main

import (
	"os"
	"os/signal"
	"syscall"
	"flag"
	"log"

	"httpproxy/config"
	"httpproxy/client"
)

var (
	cnfg config.Client
)

func main() {
	// Parse arguments
	configPtr := flag.String("c", "config/client.json", "config file")
	flag.Parse()
	// Read config file
	err := cnfg.GetConfig(*configPtr)
	if err != nil {
		log.Fatal(err)
	}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<- sigs
		client.Close()
		log.Println("Close socket")
		os.Exit(0)
	}()
	allClosed := make(chan struct{})
	// Gracefully shutdown
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		// We received an interrupt signal, shut down.
		if err := client.Shutdown(context.Background()); err != nil {
			// Error from closing listeners:
			log.Printf("Client Shutdown: %v", err)
		}
		close(allClosed)
		os.Exit(0)
	}()

	// Force termination
	go func() {
		sigterm := make(chan os.Signal, 1)
		signal.Notify(sigterm, syscall.SIGTERM)
		<-sigterm

		// Forcefully Shutdown
		if err := srv.Close(); err != nil {
			// Error from closing listeners:
			log.Printf("HTTP server Shutdown: %v", err)
		}
		close(allClosed)
		os.Exit(0)
	}()

	//go http.Serve(wln, web)
	log.Println("begin proxy client")
	// Initialize client
	client.Initialize(cnfg)

	<-allClosed
}
