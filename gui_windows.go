//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/jchv/go-webview2"
)

type WindowState struct {
	X         int  `json:"x"`
	Y         int  `json:"y"`
	Width     int  `json:"width"`
	Height    int  `json:"height"`
	Maximized bool `json:"maximized"`
}

var (
	user32           = syscall.NewLazyDLL("user32.dll")
	setWindowPos     = user32.NewProc("SetWindowPos")
	showWindow       = user32.NewProc("ShowWindow")
	getSystemMetrics = user32.NewProc("GetSystemMetrics")
)

func loadWindowState(path string) (*WindowState, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var state WindowState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, false
	}
	if state.Width < 600 || state.Height < 450 {
		return nil, false
	}
	return &state, true
}

func isPositionVisible(x, y, w, h int) bool {
	vx, _, _ := getSystemMetrics.Call(76) // SM_XVIRTUALSCREEN
	vy, _, _ := getSystemMetrics.Call(77) // SM_YVIRTUALSCREEN
	vw, _, _ := getSystemMetrics.Call(78) // SM_CXVIRTUALSCREEN
	vh, _, _ := getSystemMetrics.Call(79) // SM_CYVIRTUALSCREEN

	if vw == 0 || vh == 0 {
		vw, _, _ = getSystemMetrics.Call(0) // SM_CXSCREEN
		vh, _, _ = getSystemMetrics.Call(1) // SM_CYSCREEN
	}

	minX := int(int32(vx))
	minY := int(int32(vy))
	maxX := minX + int(vw)
	maxY := minY + int(vh)

	return x >= minX-100 && x+100 <= maxX && y >= minY && y+50 <= maxY
}

func launchDesktopGUI(url string) {
	userDataDir, _ := filepath.Abs(".webview2_data")
	_ = os.MkdirAll(userDataDir, 0755)
	stateFile := filepath.Join(userDataDir, "window_state.json")

	wWidth := 1360
	wHeight := 860
	center := true

	savedState, hasSaved := loadWindowState(stateFile)
	if hasSaved {
		wWidth = savedState.Width
		wHeight = savedState.Height
		if isPositionVisible(savedState.X, savedState.Y, savedState.Width, savedState.Height) {
			center = false
		}
	}

	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     true,
		AutoFocus: true,
		DataPath:  userDataDir,
		WindowOptions: webview2.WindowOptions{
			Title:  "Tunbridge - ベイジアン自動分別メールクライアント",
			Width:  uint(wWidth),
			Height: uint(wHeight),
			Center: center,
		},
	})
	if w != nil {
		defer w.Destroy()

		hwnd := uintptr(w.Window())
		if hwnd != 0 {
			if hasSaved && !center {
				// SWP_NOZORDER (0x0004) | SWP_NOACTIVATE (0x0010)
				setWindowPos.Call(hwnd, 0, uintptr(savedState.X), uintptr(savedState.Y), uintptr(wWidth), uintptr(wHeight), 0x0004|0x0010)
			}
			if hasSaved && savedState.Maximized {
				// SW_MAXIMIZE = 3
				showWindow.Call(hwnd, 3)
			}
		}

		w.Navigate(url)
		w.Run()
		return
	}

	// Fallback to Edge --app mode if WebView2 initialization returns nil
	edgePath := `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`
	if _, err := os.Stat(edgePath); err == nil {
		args := []string{"--app=" + url, fmt.Sprintf("--window-size=%d,%d", wWidth, wHeight)}
		if hasSaved && !center {
			args = append(args, fmt.Sprintf("--window-position=%d,%d", savedState.X, savedState.Y))
		}
		if hasSaved && savedState.Maximized {
			args = append(args, "--start-maximized")
		}
		cmd := exec.Command(edgePath, args...)
		_ = cmd.Run()
		return
	}

	log.Printf("ブラウザで %s を開いてください", url)
	select {}
}
