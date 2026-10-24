//go:build !windows

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type WindowState struct {
	X         int  `json:"x"`
	Y         int  `json:"y"`
	Width     int  `json:"width"`
	Height    int  `json:"height"`
	Maximized bool `json:"maximized"`
}

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

func launchDesktopGUI(url string) {
	stateFile := filepath.Join(".webview2_data", "window_state.json")
	wWidth := 1360
	wHeight := 860
	savedState, hasSaved := loadWindowState(stateFile)
	if hasSaved {
		wWidth = savedState.Width
		wHeight = savedState.Height
	}

	// If running on a graphical Linux / macOS session, try opening an app-mode window or default browser
	hasDisplay := os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != "" || runtime.GOOS == "darwin"
	if hasDisplay {
		appBrowsers := []string{
			"chromium",
			"chromium-browser",
			"google-chrome",
			"google-chrome-stable",
			"brave-browser",
			"microsoft-edge",
		}
		for _, bin := range appBrowsers {
			if path, err := exec.LookPath(bin); err == nil {
				log.Printf("Launching GUI window with %s (%s)", bin, url)
				args := []string{"--app=" + url, fmt.Sprintf("--window-size=%d,%d", wWidth, wHeight)}
				if hasSaved && savedState.X > 0 && savedState.Y > 0 {
					args = append(args, fmt.Sprintf("--window-position=%d,%d", savedState.X, savedState.Y))
				}
				if hasSaved && savedState.Maximized {
					args = append(args, "--start-maximized")
				}
				cmd := exec.Command(path, args...)
				if err := cmd.Start(); err == nil {
					_ = cmd.Wait()
					return
				}
			}
		}

		openCmd := "xdg-open"
		if runtime.GOOS == "darwin" {
			openCmd = "open"
		}
		if path, err := exec.LookPath(openCmd); err == nil {
			_ = exec.Command(path, url).Start()
		}
	}

	log.Printf("ブラウザで %s を開いてください (Ctrl+C で終了)", url)
	select {}
}
