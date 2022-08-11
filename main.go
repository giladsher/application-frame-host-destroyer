package main

import (
	"log"
	"os"
	"syscall"
	"time"
	"unsafe"

	ps "github.com/mitchellh/go-ps"
)

var logger = log.Default()

const (
	ModAlt = 1 << iota
	ModCtrl
	ModShift
	ModWin
)

type Hotkey struct {
	Id        int // Unique id
	Modifiers int // Mask of modifiers
	KeyCode   int // Key code, e.g. 'A'
}

type MSG struct {
	HWND   uintptr
	UINT   uintptr
	WPARAM int16
	LPARAM int64
	DWORD  int32
	POINT  struct{ X, Y int64 }
}

func main() {
	logger.Println("This script was created by ThEldeRS - A.K.A Hans.")
	logger.Println("To stop the execution press CTRL+SHIFT+Q")
	user32 := syscall.MustLoadDLL("user32")
	keys := registerKeyboardShortcut(user32)
	peekmsg := user32.MustFindProc("PeekMessageW")
	ticker := time.NewTicker(5 * time.Second)
	done := make(chan bool)

	defer func() {
		user32.Release()
		logger.Println("Released DLL")
		ticker.Stop()
		logger.Println("Deregistered timer")
		logger.Println("Thank you for using this script! Follow me on https://twitch.tv/ThEldeRS")
		time.Sleep(5 * time.Second)
	}()
	go handleWork(ticker, done)
	for {
		var msg = &MSG{}
		peekmsg.Call(uintptr(unsafe.Pointer(msg)), 0, 0, 0, 1)

		// Registered id is in the WPARAM field:
		if id := msg.WPARAM; id != 0 {
			logger.Println("Hotkey pressed:", keys[id])
			if id == 1 { // CTRL+SHIFT+Q = Exit
				done <- true
				return
			}
		}
	}
}

func registerKeyboardShortcut(user32 *syscall.DLL) map[int16]*Hotkey {
	reghotkey := user32.MustFindProc("RegisterHotKey")
	keys := map[int16]*Hotkey{
		1: {1, ModShift + ModCtrl, 'Q'}, // SHIFT+CTRL+Q
	}
	for _, v := range keys {
		r1, _, err := reghotkey.Call(
			0, uintptr(v.Id), uintptr(v.Modifiers), uintptr(v.KeyCode))
		if r1 == 1 {
			logger.Println("Registered", v)
		} else {
			logger.Println("Failed to register", v, ", error:", err)
		}
	}
	return keys
}

func handleWork(t *time.Ticker, c chan bool) {
	for {
		select {
		case <-c:
			return
		case <-t.C:
			pID, err := getApplicationFrameHostPID()
			if err != nil {
				logger.Println(err)
				logger.Println("Trying again in 1 second...")
			}
			if pID != 0 {
				err = closeApplicationFrameHost(pID)
				if err != nil {
					logger.Println(err)
				}
			}
		}
	}
}

func closeApplicationFrameHost(pId int) error {
	process, err := os.FindProcess(pId)
	if err != nil {
		return err
	}
	err = process.Kill()
	if err != nil {
		return err
	}
	logger.Println("Application Frame Host successfully exited!")
	return nil
}

func getApplicationFrameHostPID() (int, error) {
	processes, err := ps.Processes()
	var applicationFrameHostPID int

	if err != nil {
		return applicationFrameHostPID, err
	}
	for _, v := range processes {
		if v.Executable() == "ApplicationFrameHost.exe" {
			applicationFrameHostPID = v.Pid()
			logger.Println("Found ApplicationFrameHost.exe PID: ", applicationFrameHostPID)
		}
	}
	if applicationFrameHostPID == 0 {
		logger.Println("ApplicationFrameHost.exe is not running")
	}
	return applicationFrameHostPID, nil
}
