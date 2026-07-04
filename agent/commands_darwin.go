//go:build darwin || linux
// +build darwin linux

package main

import (
	"log"
)

func LockPC() error {
	log.Println("[MOCK] Locking PC (Not implemented natively on this OS)")
	return nil
}

func HardLockPC(username, adminPassword string) error {
	log.Printf("[MOCK] Hard Locking PC for user %s with admin password flow (Not implemented natively on this OS)", username)
	return nil
}

func UnlockPC(username, adminPassword string) error {
	log.Printf("[MOCK] Unlocking PC for user %s with admin password flow (Not implemented natively on this OS)", username)
	return nil
}
