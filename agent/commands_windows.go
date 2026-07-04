//go:build windows
// +build windows

package main

import (
	"fmt"
	"os/exec"
	"syscall"
)

var (
	user32          = syscall.MustLoadDLL("user32.dll")
	lockWorkStation = user32.MustFindProc("LockWorkStation")
)

const hardLockPassword = "notforyou"

func LockPC() error {
	r1, _, err := lockWorkStation.Call()
	if r1 == 0 {
		return fmt.Errorf("LockWorkStation failed: %v", err)
	}
	return nil
}

func HardLockPC(username string) error {
	if username == "" {
		return fmt.Errorf("username is required for hard lock")
	}

	// Keep account active and rotate to the configured hard-lock password.
	cmd := exec.Command("net", "user", username, "/active:yes")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to enable account: %s", string(output))
	}

	cmd = exec.Command("net", "user", username, hardLockPassword)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set hard-lock password: %s", string(output))
	}

	// Lock the session
	return LockPC()
}

func UnlockPC(username string) error {
	if username == "" {
		return fmt.Errorf("username is required for unlock")
	}

	// Enable the account
	cmd := exec.Command("net", "user", username, "/active:yes")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to enable account: %s", string(output))
	}

	return nil
}
