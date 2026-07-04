package main

import (
	"encoding/json"
	"log"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type CommandMsg struct {
	Type     string `json:"type"`
	Action   string `json:"action"`
	Username string `json:"username"`
}

type CommandResultMsg struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func buildAgentWebSocketURL(serverAddr string) string {
	trimmed := strings.TrimSpace(serverAddr)

	// Allow passing full URLs like https://host or ws://host.
	if strings.Contains(trimmed, "://") {
		parsed, err := url.Parse(trimmed)
		if err == nil {
			switch parsed.Scheme {
			case "http":
				parsed.Scheme = "ws"
			case "https":
				parsed.Scheme = "wss"
			case "ws", "wss":
				// keep as-is
			default:
				parsed.Scheme = "wss"
			}

			if parsed.Host == "" && parsed.Path != "" {
				parsed.Host = parsed.Path
				parsed.Path = ""
			}

			parsed.Path = "/agent"
			parsed.RawQuery = ""
			parsed.Fragment = ""
			return parsed.String()
		}
	}

	hostOnly := trimmed
	hostName := trimmed
	if strings.Contains(trimmed, ":") {
		if h, _, err := net.SplitHostPort(trimmed); err == nil {
			hostName = h
		}
	}

	if hostName == "localhost" || hostName == "127.0.0.1" || strings.HasPrefix(hostName, "192.168.") || strings.HasPrefix(hostName, "10.") || strings.HasPrefix(hostName, "172.") {
		u := url.URL{Scheme: "ws", Host: hostOnly, Path: "/agent"}
		return u.String()
	}

	u := url.URL{Scheme: "wss", Host: hostOnly, Path: "/agent"}
	return u.String()
}

func runAgent(serverURL string, stopChan <-chan struct{}) {
	dialURL := buildAgentWebSocketURL(serverURL)
	log.Printf("connecting to %s", dialURL)

	var c *websocket.Conn
	var err error

	// Keep trying to connect
	for {
		select {
		case <-stopChan:
			log.Println("Received stop signal during dial, exiting...")
			return
		default:
		}

		c, _, err = websocket.DefaultDialer.Dial(dialURL, nil)
		if err == nil {
			break
		}
		log.Printf("Dial error: %v, retrying in 5 seconds...", err)
		time.Sleep(5 * time.Second)
	}

	// We are connected.
	defer c.Close()
	log.Println("Connected to server.")

	// Create a channel for WebSocket read errors
	readErrChan := make(chan error, 1)

	// Start sending stats in the background
	go sendStats(c, stopChan)

	// Command listener loop in background
	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				readErrChan <- err
				return
			}

			var cmd CommandMsg
			if err := json.Unmarshal(message, &cmd); err != nil {
				log.Println("Failed to unmarshal command:", err)
				continue
			}

			if cmd.Type == "command" {
				log.Printf("Received command: %s for user: %s", cmd.Action, cmd.Username)

				var success bool
				var msg string

				switch cmd.Action {
				case "LOCK":
					err := LockPC()
					if err != nil {
						success = false
						msg = err.Error()
					} else {
						success = true
					}
				case "HARD_LOCK":
					err := HardLockPC(cmd.Username)
					if err != nil {
						success = false
						msg = err.Error()
					} else {
						success = true
						msg = "Password set to hard-lock admin password and PC locked"
					}
				case "UNLOCK":
					err := UnlockPC(cmd.Username)
					if err != nil {
						success = false
						msg = err.Error()
					} else {
						success = true
						msg = "Account unlocked successfully"
					}
				default:
					success = false
					msg = "Unknown command"
				}

				// Send result back
				res := CommandResultMsg{
					Type:    "command_result",
					Command: cmd.Action,
					Success: success,
					Message: msg,
				}

				if err := c.WriteJSON(res); err != nil {
					log.Println("write result error:", err)
				}
			}
		}
	}()

	// Wait for stop signal or connection error
	select {
	case <-stopChan:
		log.Println("Received stop signal, shutting down agent...")
		c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		time.Sleep(time.Second) // Give it a moment to close cleanly
	case err := <-readErrChan:
		log.Println("WebSocket read error:", err)
	}
}
