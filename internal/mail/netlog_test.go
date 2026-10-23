package mail

import (
	"bufio"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"testing"

	"tunbridge/internal/classifier"
	"tunbridge/internal/db"
)

func TestFetchAccountWithDiag_SocketAndAuthFailures(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.NewStore(filepath.Join(tmpDir, "test_netlog.db"))
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.DB.Close()
	engine := classifier.NewEngine(store)

	// 1. Test Socket Connection Refused (wrong port)
	lnClosed, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen failed: %v", err)
	}
	closedPort := lnClosed.Addr().(*net.TCPAddr).Port
	_ = lnClosed.Close()

	accWrongPort := &db.Account{
		ID:           1,
		Name:         "Wrong Port Account",
		EmailAddress: "user@example.com",
		Protocol:     "pop3",
		IncomingHost: "127.0.0.1",
		IncomingPort: closedPort,
		IncomingTLS:  false,
		Username:     "user@example.com",
		Password:     "secret123",
		Enabled:      true,
	}

	_, diagConnFail, err := FetchAccountWithDiag(accWrongPort, store, engine)
	if err == nil {
		t.Fatalf("Expected error when connecting to closed port")
	}
	if diagConnFail == nil {
		t.Fatalf("Expected non-nil NetworkDiagReport on connection failure")
	}
	if diagConnFail.SocketConnected {
		t.Errorf("Expected SocketConnected=false on closed port, got true")
	}
	if diagConnFail.ErrorStage != "tcp_connect" {
		t.Errorf("Expected ErrorStage='tcp_connect', got %q", diagConnFail.ErrorStage)
	}

	// 2. Test Connected Socket + POP3 Authentication Error (wrong password)
	lnMock, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen mock POP3 failed: %v", err)
	}
	defer lnMock.Close()
	mockPort := lnMock.Addr().(*net.TCPAddr).Port

	go func() {
		conn, err := lnMock.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		_, _ = fmt.Fprintf(conn, "+OK Mock POP3 Server Ready\r\n")
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			cmd := strings.TrimSpace(line)
			if strings.HasPrefix(cmd, "USER ") {
				_, _ = fmt.Fprintf(conn, "+OK User accepted\r\n")
			} else if strings.HasPrefix(cmd, "PASS ") {
				_, _ = fmt.Fprintf(conn, "-ERR [AUTH] Invalid username or password\r\n")
				return
			} else {
				_, _ = fmt.Fprintf(conn, "-ERR Unknown command\r\n")
				return
			}
		}
	}()

	accAuthFail := &db.Account{
		ID:           2,
		Name:         "Auth Fail Account",
		EmailAddress: "user@example.com",
		Protocol:     "pop3",
		IncomingHost: "127.0.0.1",
		IncomingPort: mockPort,
		IncomingTLS:  false,
		Username:     "user@example.com",
		Password:     "wrongpass",
		Enabled:      true,
	}

	_, diagAuthFail, err := FetchAccountWithDiag(accAuthFail, store, engine)
	if err == nil {
		t.Fatalf("Expected error on POP3 authentication failure")
	}
	if diagAuthFail == nil {
		t.Fatalf("Expected non-nil NetworkDiagReport on auth failure")
	}
	if !diagAuthFail.SocketConnected {
		t.Errorf("Expected SocketConnected=true when mock POP3 accepted TCP socket, got false")
	}
	if diagAuthFail.ErrorStage != "auth" {
		t.Errorf("Expected ErrorStage='auth', got %q", diagAuthFail.ErrorStage)
	}

	// Verify password is masked in trace lines
	foundMaskedPass := false
	for _, ln := range diagAuthFail.Lines {
		if strings.Contains(ln.Text, "wrongpass") {
			t.Errorf("Password was leaked in plain text in network log line: %+v", ln)
		}
		if ln.Tag == "C->S" && strings.HasPrefix(ln.Text, "PASS ********") {
			foundMaskedPass = true
		}
	}
	if !foundMaskedPass {
		t.Errorf("Expected masked PASS line in trace log, got lines: %+v", diagAuthFail.Lines)
	}
}
