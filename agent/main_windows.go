//go:build windows
// +build windows

package main

import (
	"flag"
	"log"
	"os"

	"golang.org/x/sys/windows/svc"
)

var serverURL = flag.String("server", "localhost:3000", "Server address")

type agentService struct{}

func (m *agentService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}
	
	// Create a stop channel for the agent loop
	stopChan := make(chan struct{})
	
	// Start agent loop in a goroutine
	go runAgent(*serverURL, stopChan)

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	// Service Control Manager loop
	for c := range r {
		switch c.Cmd {
		case svc.Interrogate:
			changes <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			log.Println("Service stop requested")
			changes <- svc.Status{State: svc.StopPending}
			// Signal the agent to stop
			close(stopChan)
			return
		default:
			log.Printf("Unexpected control request #%d", c)
		}
	}
	return
}

func main() {
	flag.Parse()
	log.SetFlags(0)

	isInteractive, err := svc.IsAnInteractiveSession()
	if err != nil {
		log.Fatalf("failed to determine if we are running in an interactive session: %v", err)
	}

	if isInteractive {
		// Run in console mode
		log.Println("Running in interactive console mode.")
		
		stopChan := make(chan struct{})
		// We could catch SIGINT here to close stopChan, but for testing just let it run
		runAgent(*serverURL, stopChan)
	} else {
		// Run as Windows Service
		log.Println("Running as Windows Service.")
		
		// Set up a log file since stdout is hidden in service mode
		f, err := os.OpenFile(`C:\agent_service.log`, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err == nil {
			log.SetOutput(f)
			defer f.Close()
		}

		err = svc.Run("RemotePCManager", &agentService{})
		if err != nil {
			log.Fatalf("service failed: %v", err)
		}
	}
}
