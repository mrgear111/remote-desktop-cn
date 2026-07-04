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

func LockPC() error {
	r1, _, err := lockWorkStation.Call()
	if r1 == 0 {
		return fmt.Errorf("LockWorkStation failed: %v", err)
	}
	return nil
}

func HardLockPC(username, adminPassword string) error {
	if username == "" {
		return fmt.Errorf("username is required for hard lock")
	}
	if adminPassword == "" {
		return fmt.Errorf("admin unlock password is required for hard lock")
	}

	// Ensure the account is active and then rotate password to the admin unlock password.
	cmd := exec.Command("net", "user", username, "/active:yes")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to enable account: %s", string(output))
	}

	cmd = exec.Command("net", "user", username, adminPassword)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set unlock password: %s", string(output))
	}

	// Lock the session
	return LockPC()
}

func UnlockPC(username, adminPassword string) error {
	if username == "" {
		return fmt.Errorf("username is required for unlock")
	}

	// Enable the account
	cmd := exec.Command("net", "user", username, "/active:yes")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to enable account: %s", string(output))
	}

	// Optionally rotate password again during unlock if provided.
	if adminPassword != "" {
		cmd = exec.Command("net", "user", username, adminPassword)
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to set unlock password: %s", string(output))
		}
	}

	return nil
}
