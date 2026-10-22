package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"tunbridge/internal/classifier"
	"tunbridge/internal/db"
	"tunbridge/internal/mail"
	"tunbridge/internal/server"
	"tunbridge/internal/ui"
)

func main() {
	var (
		dbPath   = flag.String("db", "tunbridge.db", "Path to SQLite database file")
		port     = flag.Int("port", 0, "Local server port (0 for automatic free port)")
		headless = flag.Bool("headless", false, "Run backend HTTP server without opening desktop GUI window")
	)
	flag.Parse()

	// Automatically migrate existing popfile_mua.db to tunbridge.db if present
	if *dbPath == "tunbridge.db" {
		if _, err := os.Stat("tunbridge.db"); os.IsNotExist(err) {
			if _, oldErr := os.Stat("popfile_mua.db"); oldErr == nil {
				_ = os.Rename("popfile_mua.db", "tunbridge.db")
				_ = os.Rename("popfile_mua.db-wal", "tunbridge.db-wal")
				_ = os.Rename("popfile_mua.db-shm", "tunbridge.db-shm")
			}
		}
	}

	store, err := db.NewStore(*dbPath)
	if err != nil {
		log.Fatalf("データベース初期化エラー: %v", err)
	}
	defer store.DB.Close()

	engine := classifier.NewEngine(store)
	if err := mail.SeedInitialData(store, engine); err != nil {
		log.Printf("初期サンプルデータのシード警告: %v", err)
	}
	if repaired, err := mail.RepairStoredEmails(store, engine); err != nil {
		log.Printf("文字コード自動修復警告: %v", err)
	} else if repaired > 0 {
		log.Printf("データベース内の %d 通のメールのMIME/多言語文字コードを修復しました", repaired)
	}

	srv := server.NewServer(store, engine, ui.Assets)

	listenAddr := fmt.Sprintf("127.0.0.1:%d", *port)
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("ローカルサーバーの起動に失敗しました: %v", err)
	}
	url := fmt.Sprintf("http://%s", ln.Addr().String())
	log.Printf("Tunbridge Engine running at %s (DB: %s)", url, *dbPath)

	go func() {
		if err := http.Serve(ln, srv.Handler()); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	if *headless {
		select {}
	}

	launchDesktopGUI(url)
}
