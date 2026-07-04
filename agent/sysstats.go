package main

import (
	"log"
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

func sendStats(c *websocket.Conn, stopChan <-chan struct{}) {
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

			err = c.WriteJSON(msg)
			if err != nil {
				log.Println("write stats error:", err)
				return // connection closed or error, exit routine
			}
		}
	}
}
