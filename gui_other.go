//go:build !windows

package main

import (
	"log"
	"os"
	"os/exec"
	"runtime"
)

func launchDesktopGUI(url string) {
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
				cmd := exec.Command(path, "--app="+url, "--window-size=1360,860")
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
