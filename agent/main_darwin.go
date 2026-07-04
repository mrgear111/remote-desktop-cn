//go:build darwin || linux
// +build darwin linux

package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var serverURL = flag.String("server", "localhost:3000", "Server address")

func main() {
	flag.Parse()
	log.SetFlags(0)

	log.Println("Running in standard console mode (Mac/Linux).")
	
	stopChan := make(chan struct{})
	
	// Handle graceful shutdown via Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received interrupt signal")
		close(stopChan)
	}()

	runAgent(*serverURL, stopChan)
}
