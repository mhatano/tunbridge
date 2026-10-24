package server_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"tunbridge/internal/classifier"
	"tunbridge/internal/db"
	"tunbridge/internal/mail"
	"tunbridge/internal/server"
	"tunbridge/internal/ui"
)

func TestServerAPIEndToEnd(t *testing.T) {
	tmpDB := filepath.Join(t.TempDir(), "api_test.db")
	store, err := db.NewStore(tmpDB)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer store.DB.Close()

	engine := classifier.NewEngine(store)
	if err := mail.SeedInitialData(store, engine); err != nil {
		t.Fatalf("SeedInitialData: %v", err)
	}

	srv := server.NewServer(store, engine, ui.Assets)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// 1. Check /api/overview
	resp, err := http.Get(ts.URL + "/api/overview")
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("GET /api/overview failed: %v", err)
	}
	var ov struct {
		Buckets []db.Bucket            `json:"buckets"`
		Stats   classifier.EngineStats `json:"stats"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&ov)
	resp.Body.Close()
	if len(ov.Buckets) < 6 {
		t.Errorf("Expected at least 6 default buckets, got %d", len(ov.Buckets))
	}
	if ov.Stats.TotalEmails == 0 {
		t.Errorf("Expected seeded emails > 0")
	}

	// 2. Simulate new incoming emails
	simBody := bytes.NewBufferString(`{"count": 2, "category": "spam"}`)
	simResp, err := http.Post(ts.URL+"/api/emails/simulate", "application/json", simBody)
	if err != nil || simResp.StatusCode != 200 {
		t.Fatalf("POST /api/emails/simulate failed: %v", err)
	}
	var simData struct {
		Count  int        `json:"count"`
		Emails []db.Email `json:"emails"`
	}
	_ = json.NewDecoder(simResp.Body).Decode(&simData)
	simResp.Body.Close()
	if simData.Count != 2 || len(simData.Emails) != 2 {
		t.Fatalf("Expected 2 simulated emails, got %d", simData.Count)
	}
	if simData.Emails[0].BucketName != "spam" {
		t.Errorf("Expected simulated spam email to be classified as 'spam', got '%s'", simData.Emails[0].BucketName)
	}

	// 3. Check /api/emails/detail and word matrix
	detailResp, err := http.Get(ts.URL + "/api/emails/detail?id=1")
	if err != nil || detailResp.StatusCode != 200 {
		t.Fatalf("GET /api/emails/detail failed: %v", err)
	}
	detailResp.Body.Close()

	// 4. Check EML export contains X-Text-Classification header
	expResp, err := http.Get(ts.URL + "/api/emails/export?id=1")
	if err != nil || expResp.StatusCode != 200 {
		t.Fatalf("GET /api/emails/export failed: %v", err)
	}
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(expResp.Body)
	expResp.Body.Close()
	if !strings.Contains(buf.String(), "X-Text-Classification:") {
		t.Errorf("Expected exported EML to contain X-Text-Classification header")
	}

	// 5. Test Delete -> Undo (trash -> restore) and Permanent Delete -> Undo (snapshot restore)
	targetEmail, err := store.GetEmail(1)
	if err != nil {
		t.Fatalf("GetEmail(1): %v", err)
	}

	trashBody := bytes.NewBufferString(`{"email_ids": [1], "action": "trash"}`)
	trashResp, err := http.Post(ts.URL+"/api/emails/mark", "application/json", trashBody)
	if err != nil || trashResp.StatusCode != 200 {
		t.Fatalf("POST /api/emails/mark (trash) failed: %v", err)
	}
	trashResp.Body.Close()

	afterTrash, _ := store.GetEmail(1)
	if afterTrash.Folder != "trash" {
		t.Errorf("Expected folder='trash', got %q", afterTrash.Folder)
	}

	restoreBody := bytes.NewBufferString(`{"email_ids": [1], "action": "restore", "restore_folder": "received"}`)
	restoreResp, err := http.Post(ts.URL+"/api/emails/mark", "application/json", restoreBody)
	if err != nil || restoreResp.StatusCode != 200 {
		t.Fatalf("POST /api/emails/mark (restore) failed: %v", err)
	}
	restoreResp.Body.Close()

	afterRestore, _ := store.GetEmail(1)
	if afterRestore.Folder != "received" {
		t.Errorf("Expected folder='received' after undo, got %q", afterRestore.Folder)
	}

	// Permanent delete and undo via snapshot
	permDelBody := bytes.NewBufferString(`{"email_ids": [1], "action": "delete"}`)
	permDelResp, err := http.Post(ts.URL+"/api/emails/mark", "application/json", permDelBody)
	if err != nil || permDelResp.StatusCode != 200 {
		t.Fatalf("POST /api/emails/mark (delete) failed: %v", err)
	}
	permDelResp.Body.Close()

	if _, err := store.GetEmail(1); err == nil {
		t.Errorf("Expected email 1 to be permanently deleted")
	}

	snapPayload, _ := json.Marshal(map[string]any{
		"email_ids":         []int64{},
		"action":            "restore",
		"restore_snapshots": []db.Email{*targetEmail},
	})
	snapResp, err := http.Post(ts.URL+"/api/emails/mark", "application/json", bytes.NewReader(snapPayload))
	if err != nil || snapResp.StatusCode != 200 {
		t.Fatalf("POST /api/emails/mark (restore_snapshots) failed: %v", err)
	}
	snapResp.Body.Close()

	restoredSnap, err := store.GetEmail(1)
	if err != nil || restoredSnap.Subject != targetEmail.Subject {
		t.Errorf("Expected snapshot undo to restore email 1 with subject %q, err=%v", targetEmail.Subject, err)
	}

	// 6. Test Sending Email with Attachments, Downloading Attachment, Exporting & Re-Importing EML with Attachments
	attData := []byte("POPFile Attachment Binary Content 1234567890\n添付ファイルテスト")
	sendPayload, _ := json.Marshal(map[string]any{
		"account_id": 0,
		"to":         "partner@corp-tech.co.jp",
		"subject":    "添付ファイル送受信テスト",
		"body":       "資料を添付いたします。ご確認ください。",
		"attachments": []map[string]any{
			{
				"filename":     "仕様書_v1.txt",
				"content_type": "text/plain; charset=utf-8",
				"size":         len(attData),
				"data_base64":  base64.StdEncoding.EncodeToString(attData),
			},
		},
	})
	sendResp, err := http.Post(ts.URL+"/api/emails/send", "application/json", bytes.NewReader(sendPayload))
	if err != nil || sendResp.StatusCode != 200 {
		t.Fatalf("POST /api/emails/send with attachment failed: %v", err)
	}
	var sendRes struct {
		Email db.Email `json:"email"`
	}
	_ = json.NewDecoder(sendResp.Body).Decode(&sendRes)
	sendResp.Body.Close()

	if sendRes.Email.AttachmentCount != 1 || len(sendRes.Email.Attachments) != 1 {
		t.Fatalf("Expected 1 attachment on sent email, got count=%d len=%d", sendRes.Email.AttachmentCount, len(sendRes.Email.Attachments))
	}
	attID := sendRes.Email.Attachments[0].ID
	if sendRes.Email.Attachments[0].Filename != "仕様書_v1.txt" {
		t.Errorf("Expected attachment filename '仕様書_v1.txt', got %q", sendRes.Email.Attachments[0].Filename)
	}

	// Download attachment via /api/emails/attachment
	dlResp, err := http.Get(ts.URL + "/api/emails/attachment?id=" + strconv.FormatInt(attID, 10))
	if err != nil || dlResp.StatusCode != 200 {
		t.Fatalf("GET /api/emails/attachment failed: %v", err)
	}
	dlBytes, _ := io.ReadAll(dlResp.Body)
	dlResp.Body.Close()
	if !bytes.Equal(dlBytes, attData) {
		t.Errorf("Downloaded attachment bytes mismatch: got %q, want %q", string(dlBytes), string(attData))
	}

	// Export sent email as .eml and re-import it to test incoming attachment parsing
	expWithAttResp, err := http.Get(ts.URL + "/api/emails/export?id=" + strconv.FormatInt(sendRes.Email.ID, 10))
	if err != nil || expWithAttResp.StatusCode != 200 {
		t.Fatalf("GET /api/emails/export failed: %v", err)
	}
	exportedEML, _ := io.ReadAll(expWithAttResp.Body)
	expWithAttResp.Body.Close()

	impPayload, _ := json.Marshal(map[string]any{
		"raw_eml_base64": base64.StdEncoding.EncodeToString(exportedEML),
	})
	impResp, err := http.Post(ts.URL+"/api/emails/import", "application/json", bytes.NewReader(impPayload))
	if err != nil || impResp.StatusCode != 200 {
		t.Fatalf("POST /api/emails/import with attachment failed: %v", err)
	}
	var impRes struct {
		Email db.Email `json:"email"`
	}
	_ = json.NewDecoder(impResp.Body).Decode(&impRes)
	impResp.Body.Close()

	if impRes.Email.AttachmentCount != 1 || len(impRes.Email.Attachments) != 1 {
		t.Fatalf("Expected imported email to have 1 attachment, got count=%d len=%d", impRes.Email.AttachmentCount, len(impRes.Email.Attachments))
	}
	if impRes.Email.Attachments[0].Filename != "仕様書_v1.txt" {
		t.Errorf("Expected imported attachment filename '仕様書_v1.txt', got %q", impRes.Email.Attachments[0].Filename)
	}

	// 11. Test Account Create, Update (Overwrite), and Duplicate via /api/accounts/save
	accCreatePayload, _ := json.Marshal(db.Account{
		ID:           0,
		Name:         "Main POP3 Account",
		EmailAddress: "user1@example.com",
		Protocol:     "pop3",
		IncomingHost: "pop.example.com",
		IncomingPort: 995,
		IncomingTLS:  true,
		Username:     "user1@example.com",
		Password:     "secret1",
		SMTPHost:     "smtp.example.com",
		SMTPPort:     587,
		SMTPTLS:      true,
		Enabled:      true,
	})
	accCreateResp, err := http.Post(ts.URL+"/api/accounts/save", "application/json", bytes.NewReader(accCreatePayload))
	if err != nil || accCreateResp.StatusCode != 200 {
		t.Fatalf("POST /api/accounts/save (create) failed: %v", err)
	}
	var accCreateRes struct {
		Account db.Account `json:"account"`
	}
	_ = json.NewDecoder(accCreateResp.Body).Decode(&accCreateRes)
	accCreateResp.Body.Close()
	if accCreateRes.Account.ID <= 0 {
		t.Fatalf("Expected created account ID > 0, got %d", accCreateRes.Account.ID)
	}

	// Overwrite existing account by passing its ID
	accUpdatePayload, _ := json.Marshal(db.Account{
		ID:           accCreateRes.Account.ID,
		Name:         "Main POP3 Account (Updated)",
		EmailAddress: "user1-updated@example.com",
		Protocol:     "pop3",
		IncomingHost: "pop.example.com",
		IncomingPort: 995,
		IncomingTLS:  true,
		Username:     "user1-updated@example.com",
		Password:     "secret1-new",
		SMTPHost:     "smtp.example.com",
		SMTPPort:     587,
		SMTPTLS:      true,
		Enabled:      true,
	})
	accUpdateResp, err := http.Post(ts.URL+"/api/accounts/save", "application/json", bytes.NewReader(accUpdatePayload))
	if err != nil || accUpdateResp.StatusCode != 200 {
		t.Fatalf("POST /api/accounts/save (update) failed: %v", err)
	}
	accUpdateResp.Body.Close()

	// Duplicate account as disabled backup (ID = 0)
	accDupPayload, _ := json.Marshal(db.Account{
		ID:           0,
		Name:         "Main POP3 Account (Updated) (コピー)",
		EmailAddress: "user1-updated@example.com",
		Protocol:     "pop3",
		IncomingHost: "pop.example.com",
		IncomingPort: 995,
		IncomingTLS:  true,
		Username:     "user1-updated@example.com",
		Password:     "secret1-new",
		SMTPHost:     "smtp.example.com",
		SMTPPort:     587,
		SMTPTLS:      true,
		Enabled:      false,
	})
	accDupResp, err := http.Post(ts.URL+"/api/accounts/save", "application/json", bytes.NewReader(accDupPayload))
	if err != nil || accDupResp.StatusCode != 200 {
		t.Fatalf("POST /api/accounts/save (duplicate) failed: %v", err)
	}
	accDupResp.Body.Close()

	// Verify 2 accounts exist (1 updated, 1 duplicated & disabled)
	accListResp, err := http.Get(ts.URL + "/api/accounts")
	if err != nil || accListResp.StatusCode != 200 {
		t.Fatalf("GET /api/accounts failed: %v", err)
	}
	var accListRes struct {
		Accounts []db.Account `json:"accounts"`
	}
	_ = json.NewDecoder(accListResp.Body).Decode(&accListRes)
	accListResp.Body.Close()
	if len(accListRes.Accounts) != 2 {
		t.Fatalf("Expected 2 accounts after update + duplicate, got %d", len(accListRes.Accounts))
	}
	if accListRes.Accounts[0].Name != "Main POP3 Account (Updated)" || !accListRes.Accounts[0].Enabled {
		t.Errorf("Unexpected first account state: %+v", accListRes.Accounts[0])
	}
	if accListRes.Accounts[1].Name != "Main POP3 Account (Updated) (コピー)" || accListRes.Accounts[1].Enabled {
		t.Errorf("Unexpected duplicated account state: %+v", accListRes.Accounts[1])
	}

	// 10. Test /api/accounts/export
	exportResp, err := http.Get(ts.URL + "/api/accounts/export")
	if err != nil || exportResp.StatusCode != 200 {
		t.Fatalf("GET /api/accounts/export failed: %v", err)
	}
	cd := exportResp.Header.Get("Content-Disposition")
	if !strings.Contains(cd, "attachment") || !strings.Contains(cd, "tunbridge-accounts-") {
		t.Errorf("Expected attachment Content-Disposition header, got %q", cd)
	}
	var exportData struct {
		App      string `json:"app"`
		Count    int    `json:"count"`
		Accounts []struct {
			Name          string `json:"name"`
			IncomingHost  string `json:"incoming_host"`
			CheckInterval int    `json:"check_interval"`
		} `json:"accounts"`
	}
	if err := json.NewDecoder(exportResp.Body).Decode(&exportData); err != nil {
		t.Fatalf("Failed to decode export JSON: %v", err)
	}
	exportResp.Body.Close()
	if exportData.Count != 2 || len(exportData.Accounts) != 2 {
		t.Fatalf("Expected 2 exported accounts, got %d", exportData.Count)
	}

	// 11. Test /api/accounts/import with new account
	importPayload := map[string]any{
		"accounts": []map[string]any{
			{
				"name":           "Imported Fastmail JMAP",
				"email_address":  "user@fastmail.com",
				"protocol":       "jmap",
				"incoming_host":  "api.fastmail.com",
				"incoming_port":  443,
				"incoming_tls":   true,
				"username":       "user@fastmail.com",
				"password":       "fmpwd",
				"enabled":        true,
				"check_interval": 5,
			},
		},
	}
	importBytes, _ := json.Marshal(importPayload)
	importResp, err := http.Post(ts.URL+"/api/accounts/import", "application/json", bytes.NewReader(importBytes))
	if err != nil || importResp.StatusCode != 200 {
		t.Fatalf("POST /api/accounts/import failed: %v", err)
	}
	var importRes struct {
		OK       bool `json:"ok"`
		Imported int  `json:"imported"`
	}
	_ = json.NewDecoder(importResp.Body).Decode(&importRes)
	importResp.Body.Close()
	if !importRes.OK || importRes.Imported != 1 {
		t.Fatalf("Expected imported=1, got %+v", importRes)
	}

	// Verify 3 accounts now exist and check_interval is 5
	accListResp2, err := http.Get(ts.URL + "/api/accounts")
	if err != nil || accListResp2.StatusCode != 200 {
		t.Fatalf("GET /api/accounts after import failed: %v", err)
	}
	var accListRes2 struct {
		Accounts []db.Account `json:"accounts"`
	}
	_ = json.NewDecoder(accListResp2.Body).Decode(&accListRes2)
	accListResp2.Body.Close()
	if len(accListRes2.Accounts) != 3 {
		t.Fatalf("Expected 3 accounts after import, got %d", len(accListRes2.Accounts))
	}
	var jmapAcc *db.Account
	for _, a := range accListRes2.Accounts {
		if a.Name == "Imported Fastmail JMAP" {
			jmapAcc = &a
			break
		}
	}
	if jmapAcc == nil {
		t.Fatalf("Could not find imported JMAP account in list")
	}
	if jmapAcc.CheckInterval != 5 || jmapAcc.Protocol != "jmap" {
		t.Errorf("Unexpected imported account details: %+v", jmapAcc)
	}

	// Test UpdateAccountCheckStatus
	testTime := "2026-10-24 19:00:00"
	testStatus := "成功 (1通受信)"
	if err := store.UpdateAccountCheckStatus(jmapAcc.ID, testTime, testStatus); err != nil {
		t.Fatalf("UpdateAccountCheckStatus failed: %v", err)
	}
	accountsRefreshed, _ := store.GetAccounts()
	foundUpdated := false
	for _, a := range accountsRefreshed {
		if a.ID == jmapAcc.ID {
			if a.LastCheckedAt != testTime || a.LastCheckStatus != testStatus {
				t.Errorf("Expected updated check status %q/%q, got %q/%q", testTime, testStatus, a.LastCheckedAt, a.LastCheckStatus)
			}
			foundUpdated = true
			break
		}
	}
	if !foundUpdated {
		t.Errorf("Could not verify updated account in store")
	}

	// 12. Test /api/window/state (POST & GET)
	winPayload := `{"x": 120, "y": 80, "width": 1400, "height": 900, "maximized": false}`
	winPostResp, err := http.Post(ts.URL+"/api/window/state", "application/json", strings.NewReader(winPayload))
	if err != nil || winPostResp.StatusCode != 200 {
		t.Fatalf("POST /api/window/state failed: %v", err)
	}
	winPostResp.Body.Close()

	winGetResp, err := http.Get(ts.URL + "/api/window/state")
	if err != nil || winGetResp.StatusCode != 200 {
		t.Fatalf("GET /api/window/state failed: %v", err)
	}
	var winState struct {
		X         int  `json:"x"`
		Y         int  `json:"y"`
		Width     int  `json:"width"`
		Height    int  `json:"height"`
		Maximized bool `json:"maximized"`
	}
	_ = json.NewDecoder(winGetResp.Body).Decode(&winState)
	winGetResp.Body.Close()
	if winState.X != 120 || winState.Y != 80 || winState.Width != 1400 || winState.Height != 900 {
		t.Errorf("Unexpected window state: %+v", winState)
	}
}
