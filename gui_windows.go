//go:build windows

package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/jchv/go-webview2"
)

func launchDesktopGUI(url string) {
	userDataDir, _ := filepath.Abs(".webview2_data")
	_ = os.MkdirAll(userDataDir, 0755)

	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     true,
		AutoFocus: true,
		DataPath:  userDataDir,
		WindowOptions: webview2.WindowOptions{
			Title:  "Tunbridge - ベイジアン自動分別メールクライアント",
			Width:  1360,
			Height: 860,
			Center: true,
		},
	})
	if w != nil {
		defer w.Destroy()
		w.Navigate(url)
		w.Run()
		return
	}

	// Fallback to Edge --app mode if WebView2 initialization returns nil
	edgePath := `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`
	if _, err := os.Stat(edgePath); err == nil {
		cmd := exec.Command(edgePath, "--app="+url, "--window-size=1360,860")
		_ = cmd.Run()
		return
	}

	log.Printf("ブラウザで %s を開いてください", url)
	select {}
}
