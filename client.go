package main

import (
	"os"
	"os/exec"
	"os/signal"
	"strings"
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
		if err := client.Shutdown(); err != nil {
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
		if err := client.Close(); err != nil {
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
	// Run after start hook
	if cnfg.AfterStart != "" {
		args := strings.SplitN(cnfg.AfterStart, " ", 2)
		if len(args) == 1 {
			args = append(args, "")
		}
		cmd := exec.Command(args[0], args[1])
		go func() {
			err := cmd.Run()
			if err != nil {
				log.Printf("AfterStart hook error: %v", err)
			}
		}()
	}

	<-allClosed
}
