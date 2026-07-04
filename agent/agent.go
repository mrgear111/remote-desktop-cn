package main

import (
	"encoding/json"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

type CommandMsg struct {
	Type          string `json:"type"`
	Action        string `json:"action"`
	Username      string `json:"username"`
	AdminPassword string `json:"adminPassword"`
}

type CommandResultMsg struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func runAgent(serverURL string, stopChan <-chan struct{}) {
	u := url.URL{Scheme: "ws", Host: serverURL, Path: "/agent"}
	log.Printf("connecting to %s", u.String())

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

		c, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
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
					err := HardLockPC(cmd.Username, cmd.AdminPassword)
					if err != nil {
						success = false
						msg = err.Error()
					} else {
						success = true
						msg = "Password updated and PC locked. Use admin unlock password to sign in."
					}
				case "UNLOCK":
					err := UnlockPC(cmd.Username, cmd.AdminPassword)
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
