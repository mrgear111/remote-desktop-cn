package main

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

type StatsMsg struct {
	Type   string  `json:"type"`
	CPU    float64 `json:"cpu"`
	Memory float64 `json:"memory"`
}

func sendStats(c *websocket.Conn, stopChan <-chan struct{}, writeMutex *sync.Mutex) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			cpuPercents, err := cpu.Percent(0, false)
			var cpuUsage float64
			if err == nil && len(cpuPercents) > 0 {
				cpuUsage = cpuPercents[0]
			}

			v, err := mem.VirtualMemory()
			var memUsage float64
			if err == nil {
				memUsage = v.UsedPercent
			}

			msg := StatsMsg{
				Type:   "stats",
				CPU:    cpuUsage,
				Memory: memUsage,
			}

			writeMutex.Lock()
			c.SetWriteDeadline(time.Now().Add(5 * time.Second))
			err = c.WriteJSON(msg)
			writeMutex.Unlock()

			if err != nil {
				log.Println("write stats error:", err)
				return // connection closed or error, exit routine
			}
		}
	}
}
