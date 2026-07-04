//go:build windows
// +build windows

package main

import (
	"fmt"
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32          = syscall.MustLoadDLL("user32.dll")
	lockWorkStation = user32.MustFindProc("LockWorkStation")

	wtsapi32             = syscall.MustLoadDLL("wtsapi32.dll")
	wtsDisconnectSession = wtsapi32.MustFindProc("WTSDisconnectSession")
)

func LockPC() error {
	// First, try LockWorkStation. This works perfectly if running interactively.
	r1, _, err := lockWorkStation.Call()
	if r1 != 0 {
		return nil // Success
	}

	// If LockWorkStation fails (e.g. Access is Denied because we are a Windows Service in Session 0),
	// we spawn a process in the interactive user's session to gracefully lock it without a black screen.
	sessionID := windows.WTSGetActiveConsoleSessionId()
	if sessionID == 0xFFFFFFFF {
		return fmt.Errorf("LockWorkStation failed: %v, and no active console session found", err)
	}

	var token windows.Token
	tokenErr := windows.WTSQueryUserToken(sessionID, &token)
	if tokenErr == nil {
		defer token.Close()
		var si windows.StartupInfo
		si.Cb = uint32(unsafe.Sizeof(si))
		si.Desktop = windows.StringToUTF16Ptr(`winsta0\default`)

		var pi windows.ProcessInformation
		cmdLine := windows.StringToUTF16Ptr(`rundll32.exe user32.dll,LockWorkStation`)

		spawnErr := windows.CreateProcessAsUser(
			token,
			nil,
			cmdLine,
			nil,
			nil,
			false,
			0,
			nil,
			nil,
			&si,
			&pi,
		)
		if spawnErr == nil {
			windows.CloseHandle(pi.Process)
			windows.CloseHandle(pi.Thread)
			return nil // Graceful interactive lock success
		}
	}

	// Fallback to WTSDisconnectSession if we couldn't spawn the process.
	// This forces the active session to disconnect, causing a 2-second black screen but guarantees a lock.
	rDis, _, errDis := wtsDisconnectSession.Call(0, uintptr(sessionID), 0)
	if rDis == 0 {
		return fmt.Errorf("WTSDisconnectSession failed: %v", errDis)
	}

	return nil
}

func HardLockPC(username string) error {
	if username == "" {
		return fmt.Errorf("username is required for hard lock")
	}

	// Disable account so normal password cannot be used while hard-locked.
	cmd := exec.Command("net", "user", username, "/active:no")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to disable account: %s", string(output))
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
