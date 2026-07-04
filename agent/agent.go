package main

import (
	"encoding/json"
	"log"
	"net"
	"net/url"
	"strings"
	"sync"
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

func isInternetDown() bool {
	timeout := 2 * time.Second
	conn, err := net.DialTimeout("tcp", "8.8.8.8:53", timeout)
	if err != nil {
		// Could also check 1.1.1.1 if 8.8.8.8 fails just to be sure
		conn2, err2 := net.DialTimeout("tcp", "1.1.1.1:53", timeout)
		if err2 != nil {
			return true
		}
		conn2.Close()
		return false
	}
	conn.Close()
	return false
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

func StartAgent(serverURL string, stopChan <-chan struct{}) {
	for {
		select {
		case <-stopChan:
			log.Println("StartAgent: Received stop signal, shutting down completely.")
			return
		default:
			log.Println("Starting agent session...")
			runAgentSession(serverURL, stopChan)
			
			// If we get here, the connection dropped. Wait before retrying.
			log.Println("Agent session ended. Reconnecting in 5 seconds...")
			time.Sleep(5 * time.Second)
		}
	}
}

func runAgentSession(serverURL string, stopChan <-chan struct{}) {
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

	// We need a mutex to prevent concurrent writes to the WebSocket connection
	var writeMutex sync.Mutex

	pongWait := 10 * time.Second
	pingPeriod := 5 * time.Second
	writeWait := 5 * time.Second

	c.SetReadDeadline(time.Now().Add(pongWait))
	c.SetPongHandler(func(string) error {
		c.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// Start sending pings
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				writeMutex.Lock()
				c.SetWriteDeadline(time.Now().Add(writeWait))
				err := c.WriteMessage(websocket.PingMessage, nil)
				writeMutex.Unlock()
				if err != nil {
					c.Close()
					return
				}
			}
		}
	}()

	// Track locked users for auto-unlock on disconnect
	var lockedUsersMutex sync.Mutex
	lockedUsers := make(map[string]bool)

	// Start sending stats in the background
	go sendStats(c, stopChan, &writeMutex)

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
						lockedUsersMutex.Lock()
						lockedUsers[cmd.Username] = true
						lockedUsersMutex.Unlock()

						success = true
						msg = "Account disabled and PC locked"
					}
				case "UNLOCK":
					err := UnlockPC(cmd.Username)
					if err != nil {
						success = false
						msg = err.Error()
					} else {
						lockedUsersMutex.Lock()
						delete(lockedUsers, cmd.Username)
						lockedUsersMutex.Unlock()

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

				writeMutex.Lock()
				c.SetWriteDeadline(time.Now().Add(5 * time.Second))
				err := c.WriteJSON(res)
				writeMutex.Unlock()
				
				if err != nil {
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

	// Auto-unlock instantly if agent disconnects AND internet is down
	lockedUsersMutex.Lock()
	if len(lockedUsers) > 0 {
		if isInternetDown() {
			for user := range lockedUsers {
				log.Printf("Internet is down. Auto-unlocking user: %s", user)
				UnlockPC(user)
			}
		} else {
			log.Println("Agent disconnected, but internet is up. Keeping PC locked.")
		}
	}
	lockedUsersMutex.Unlock()
}
