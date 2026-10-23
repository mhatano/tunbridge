package mail

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"tunbridge/internal/classifier"
	"tunbridge/internal/db"
)

func TestFetchAccountWithDiag_JMAPSuccessAndAuthFail(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.NewStore(filepath.Join(tmpDir, "test_jmap.db"))
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.DB.Close()
	engine := classifier.NewEngine(store)

	const validToken = "valid-secret-token"
	const rawSampleEML = "From: Alice <alice@example.com>\r\nTo: Bob <bob@example.com>\r\nSubject: JMAP Hello World\r\nDate: Fri, 23 Oct 2026 12:00:00 +0000\r\nMessage-ID: <jmap-test-1@example.com>\r\n\r\nHello from JMAP!"

	// Set up mock JMAP server
	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer "+validToken {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"type":"urn:ietf:params:jmap:error:unauthorized","status":401}`))
			return
		}

		switch r.URL.Path {
		case "/.well-known/jmap", "/jmap/session":
			sess := JMAPSession{
				Capabilities: map[string]any{
					"urn:ietf:params:jmap:core": map[string]any{},
					"urn:ietf:params:jmap:mail": map[string]any{},
				},
				Accounts: map[string]JMAPAccount{
					"acc-001": {
						Name:       "user@example.com",
						IsPersonal: true,
						AccountCapabilities: map[string]any{
							"urn:ietf:params:jmap:mail": map[string]any{},
						},
					},
				},
				PrimaryAccounts: map[string]string{
					"urn:ietf:params:jmap:mail": "acc-001",
				},
				APIURL:      ts.URL + "/jmap/api",
				DownloadURL: ts.URL + "/jmap/download/{accountId}/{blobId}/{name}?type={type}",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(sess)

		case "/jmap/api":
			var req JMAPRequest
			_ = json.NewDecoder(r.Body).Decode(&req)

			resp := JMAPResponse{
				MethodResponses: [][]any{
					{
						"Email/query",
						map[string]any{
							"accountId": "acc-001",
							"ids":       []string{"em-101"},
						},
						"q0",
					},
					{
						"Email/get",
						map[string]any{
							"accountId": "acc-001",
							"list": []map[string]any{
								{
									"id":        "em-101",
									"blobId":    "blob-101",
									"messageId": []string{"<jmap-test-1@example.com>"},
									"subject":   "JMAP Hello World",
									"from": []map[string]string{
										{"name": "Alice", "email": "alice@example.com"},
									},
									"to": []map[string]string{
										{"name": "Bob", "email": "bob@example.com"},
									},
								},
							},
						},
						"g0",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)

		case "/jmap/download/acc-001/blob-101/message.eml":
			w.Header().Set("Content-Type", "message/rfc822")
			_, _ = w.Write([]byte(rawSampleEML))

		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	port, _ := strconv.Atoi(u.Port())

	// 1. Test JMAP Success with Bearer token
	accOK := &db.Account{
		ID:           10,
		Name:         "JMAP Account",
		EmailAddress: "bob@example.com",
		Protocol:     "jmap",
		IncomingHost: u.Hostname(),
		IncomingPort: port,
		IncomingTLS:  false, // httptest.Server is http
		Username:     "",
		Password:     validToken,
		Enabled:      true,
	}

	fetched, diagOK, err := FetchAccountWithDiag(accOK, store, engine)
	if err != nil {
		t.Fatalf("FetchAccountWithDiag (JMAP) failed: %v", err)
	}
	if fetched != 1 {
		t.Errorf("Expected 1 email fetched via JMAP, got %d", fetched)
	}
	if diagOK == nil || !diagOK.SocketConnected {
		t.Errorf("Expected diagOK.SocketConnected=true, got %+v", diagOK)
	}

	// Verify email in store
	emails, err := store.ListEmails(db.EmailFilter{Folder: "received"})
	if err != nil || len(emails) != 1 {
		t.Fatalf("Expected 1 stored email, got %d (err: %v)", len(emails), err)
	}
	if emails[0].Subject != "JMAP Hello World" {
		t.Errorf("Subject mismatch: got %q, want 'JMAP Hello World'", emails[0].Subject)
	}
	if emails[0].FromAddress != "alice@example.com" {
		t.Errorf("FromAddress mismatch: got %q, want 'alice@example.com'", emails[0].FromAddress)
	}

	// 2. Test JMAP Auth Failure with wrong token
	accAuthFail := &db.Account{
		ID:           11,
		Name:         "JMAP Bad Auth",
		EmailAddress: "bob@example.com",
		Protocol:     "jmap",
		IncomingHost: u.Hostname(),
		IncomingPort: port,
		IncomingTLS:  false,
		Username:     "",
		Password:     "wrong-invalid-token",
		Enabled:      true,
	}

	_, diagFail, err := FetchAccountWithDiag(accAuthFail, store, engine)
	if err == nil {
		t.Fatalf("Expected error on invalid JMAP token")
	}
	if diagFail == nil {
		t.Fatalf("Expected non-nil diagFail report")
	}
	if !diagFail.SocketConnected {
		t.Errorf("Expected SocketConnected=true for HTTP 401, got false")
	}
	if diagFail.ErrorStage != "auth" {
		t.Errorf("Expected ErrorStage='auth', got %q", diagFail.ErrorStage)
	}
	if !strings.Contains(diagFail.DiagnosisJA, "JMAP") && !strings.Contains(diagFail.DiagnosisJA, "認証") {
		t.Errorf("Expected auth explanation in DiagnosisJA, got: %s", diagFail.DiagnosisJA)
	}
}
