package mail

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"tunbridge/internal/classifier"
	"tunbridge/internal/db"
)

// JMAPSession represents the JMAP Session Resource (RFC 8620 Section 2).
type JMAPSession struct {
	Capabilities    map[string]any         `json:"capabilities"`
	Accounts        map[string]JMAPAccount `json:"accounts"`
	PrimaryAccounts map[string]string      `json:"primaryAccounts"`
	Username        string                 `json:"username"`
	APIURL          string                 `json:"apiUrl"`
	DownloadURL     string                 `json:"downloadUrl"`
	UploadURL       string                 `json:"uploadUrl"`
	EventSourceURL  string                 `json:"eventSourceUrl"`
	State           string                 `json:"state"`
}

type JMAPAccount struct {
	Name                string         `json:"name"`
	IsPersonal          bool           `json:"isPersonal"`
	IsReadOnly          bool           `json:"isReadOnly"`
	AccountCapabilities map[string]any `json:"accountCapabilities"`
}

type JMAPRequest struct {
	Using       []string `json:"using"`
	MethodCalls [][]any  `json:"methodCalls"`
}

type JMAPResponse struct {
	MethodResponses [][]any `json:"methodResponses"`
	SessionState    string  `json:"sessionState"`
}

type JMAPEmail struct {
	ID          string                   `json:"id"`
	BlobID      string                   `json:"blobId"`
	ThreadID    string                   `json:"threadId"`
	MessageID   []string                 `json:"messageId"`
	InReplyTo   []string                 `json:"inReplyTo"`
	From        []JMAPAddress            `json:"from"`
	To          []JMAPAddress            `json:"to"`
	Cc          []JMAPAddress            `json:"cc"`
	Bcc         []JMAPAddress            `json:"bcc"`
	Subject     string                   `json:"subject"`
	SentAt      string                   `json:"sentAt"`
	ReceivedAt  string                   `json:"receivedAt"`
	Size        int                      `json:"size"`
	Preview     string                   `json:"preview"`
	BodyValues  map[string]JMAPBodyValue `json:"bodyValues"`
	TextBody    []JMAPBodyPart           `json:"textBody"`
	HTMLBody    []JMAPBodyPart           `json:"htmlBody"`
	Attachments []JMAPAttachment         `json:"attachments"`
}

type JMAPAddress struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type JMAPBodyValue struct {
	Value       string `json:"value"`
	IsTruncated bool   `json:"isTruncated"`
}

type JMAPBodyPart struct {
	PartID string `json:"partId"`
	BlobID string `json:"blobId"`
	Size   int    `json:"size"`
	Type   string `json:"type"`
}

type JMAPAttachment struct {
	BlobID string `json:"blobId"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Size   int    `json:"size"`
	CID    string `json:"cid"`
}

func fetchJMAPWithDiag(acc *db.Account, store *db.Store, engine *classifier.Engine, diag *NetworkDiagReport) (int, error) {
	client, hostName := createJMAPHTTPClient(acc, diag)
	session, sessionURL, err := discoverJMAPSession(client, acc, diag)
	if err != nil {
		return 0, err
	}

	mailAccID := session.PrimaryAccounts["urn:ietf:params:jmap:mail"]
	if mailAccID == "" {
		for id, a := range session.Accounts {
			if _, ok := a.AccountCapabilities["urn:ietf:params:jmap:mail"]; ok {
				mailAccID = id
				break
			}
		}
	}
	if mailAccID == "" {
		for id := range session.Accounts {
			mailAccID = id
			break
		}
	}
	if mailAccID == "" {
		accErr := fmt.Errorf("JMAPセッション内にメール機能 (urn:ietf:params:jmap:mail) を持つアカウントが見つかりません")
		diag.Add("ERROR", accErr.Error())
		diag.SetFailure("command", accErr)
		return 0, accErr
	}

	diag.Add("INFO", fmt.Sprintf("Using JMAP Mail Account ID: %s (API URL: %s)", mailAccID, session.APIURL))

	apiURL, err := sessionURL.Parse(session.APIURL)
	if err != nil {
		diag.Add("ERROR", fmt.Sprintf("Invalid JMAP apiUrl %q: %v", session.APIURL, err))
		diag.SetFailure("command", err)
		return 0, err
	}

	// 1. Email/query + Email/get
	jmapReq := JMAPRequest{
		Using: []string{
			"urn:ietf:params:jmap:core",
			"urn:ietf:params:jmap:mail",
		},
		MethodCalls: [][]any{
			{
				"Email/query",
				map[string]any{
					"accountId": mailAccID,
					"sort": []map[string]any{
						{"property": "receivedAt", "isAscending": false},
					},
					"limit": 25,
				},
				"q0",
			},
			{
				"Email/get",
				map[string]any{
					"accountId": mailAccID,
					"#ids": map[string]any{
						"resultOf": "q0",
						"name":     "Email/query",
						"path":     "/ids",
					},
					"properties": []string{
						"id", "blobId", "threadId", "messageId", "inReplyTo",
						"from", "to", "cc", "bcc", "subject", "sentAt", "receivedAt",
						"size", "preview", "bodyValues", "textBody", "htmlBody", "attachments",
					},
					"fetchTextBodyValues": true,
					"fetchHTMLBodyValues": true,
				},
				"g0",
			},
		},
	}

	reqBytes, _ := json.Marshal(jmapReq)
	httpReq, err := http.NewRequestWithContext(context.Background(), "POST", apiURL.String(), bytes.NewReader(reqBytes))
	if err != nil {
		diag.SetFailure("command", err)
		return 0, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	diag.Add("C->S", fmt.Sprintf("POST %s (Email/query + Email/get)", apiURL.RequestURI()))
	_, respBytes, err := doJMAPRequest(client, httpReq, acc, diag)
	if err != nil {
		return 0, err
	}

	var jmapResp JMAPResponse
	if err := json.Unmarshal(respBytes, &jmapResp); err != nil {
		unmarshalErr := fmt.Errorf("JMAP応答JSONのパースに失敗しました: %w", err)
		diag.Add("ERROR", unmarshalErr.Error())
		diag.SetFailure("command", unmarshalErr)
		return 0, unmarshalErr
	}

	var emailList []JMAPEmail
	for _, call := range jmapResp.MethodResponses {
		if len(call) >= 2 {
			name, _ := call[0].(string)
			if name == "Email/get" {
				if args, ok := call[1].(map[string]any); ok {
					if listRaw, ok := args["list"]; ok {
						listBytes, _ := json.Marshal(listRaw)
						_ = json.Unmarshal(listBytes, &emailList)
					}
				}
			}
		}
	}

	diag.Add("INFO", fmt.Sprintf("JMAP Email/get returned %d message(s)", len(emailList)))

	fetched := 0
	for _, em := range emailList {
		uidl := fmt.Sprintf("jmap:%s:%s", acc.EmailAddress, em.ID)
		msgID := ""
		if len(em.MessageID) > 0 {
			msgID = em.MessageID[0]
		}

		if store.ExistsByUIDLOrMessageID(uidl, msgID) {
			continue
		}

		// Try downloading raw RFC822 blob if downloadUrl is provided
		downloadedRaw := false
		if session.DownloadURL != "" && em.BlobID != "" {
			rawBytes, err := downloadJMAPBlob(client, sessionURL, session.DownloadURL, mailAccID, em.BlobID, acc, diag)
			if err == nil && len(rawBytes) > 10 {
				if err := ingestRawEmail(rawBytes, uidl, store, engine); err == nil {
					fetched++
					downloadedRaw = true
				}
			}
		}

		// Fallback: ingest using JMAP structured fields
		if !downloadedRaw {
			if err := ingestJMAPStructuredEmail(&em, uidl, store, engine); err == nil {
				fetched++
			}
		}
	}

	_ = hostName
	return fetched, nil
}

func downloadJMAPBlob(client *http.Client, sessionURL *url.URL, tmpl, accountID, blobID string, acc *db.Account, diag *NetworkDiagReport) ([]byte, error) {
	dl := tmpl
	dl = strings.ReplaceAll(dl, "{accountId}", url.PathEscape(accountID))
	dl = strings.ReplaceAll(dl, "{blobId}", url.PathEscape(blobID))
	dl = strings.ReplaceAll(dl, "{name}", url.PathEscape("message.eml"))
	dl = strings.ReplaceAll(dl, "{type}", url.QueryEscape("application/octet-stream"))

	targetURL, err := sessionURL.Parse(dl)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(context.Background(), "GET", targetURL.String(), nil)
	if err != nil {
		return nil, err
	}
	diag.Add("C->S", fmt.Sprintf("GET %s (download blob %s)", targetURL.RequestURI(), blobID))

	_, body, err := doJMAPRequest(client, req, acc, diag)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func ingestJMAPStructuredEmail(j *JMAPEmail, uidl string, store *db.Store, engine *classifier.Engine) error {
	fromName := "Unknown"
	fromAddr := "unknown@localhost"
	if len(j.From) > 0 {
		fromName = j.From[0].Name
		fromAddr = j.From[0].Email
		if fromName == "" {
			fromName = fromAddr
		}
	}

	var toAddrs []string
	for _, a := range j.To {
		if a.Name != "" && a.Name != a.Email {
			toAddrs = append(toAddrs, fmt.Sprintf("%s <%s>", a.Name, a.Email))
		} else {
			toAddrs = append(toAddrs, a.Email)
		}
	}
	toStr := strings.Join(toAddrs, ", ")

	var bodyTextParts []string
	for _, part := range j.TextBody {
		if val, ok := j.BodyValues[part.PartID]; ok {
			bodyTextParts = append(bodyTextParts, val.Value)
		}
	}
	bodyText := strings.Join(bodyTextParts, "\n\n")

	var bodyHTMLParts []string
	for _, part := range j.HTMLBody {
		if val, ok := j.BodyValues[part.PartID]; ok {
			bodyHTMLParts = append(bodyHTMLParts, val.Value)
		}
	}
	bodyHTML := strings.Join(bodyHTMLParts, "\n\n")

	if bodyText == "" && bodyHTML != "" {
		bodyText = classifier.StripHTML(bodyHTML)
	}
	if bodyText == "" && j.Preview != "" {
		bodyText = j.Preview
	}

	dateStr := time.Now().Format("2006-01-02 15:04:05")
	rawDate := j.ReceivedAt
	if rawDate == "" {
		rawDate = j.SentAt
	}
	if rawDate != "" {
		if t, err := time.Parse(time.RFC3339, rawDate); err == nil {
			dateStr = t.Local().Format("2006-01-02 15:04:05")
		}
	}

	msgID := ""
	if len(j.MessageID) > 0 {
		msgID = strings.Trim(j.MessageID[0], "<> ")
	}
	if msgID == "" {
		msgID = fmt.Sprintf("jmap-%s", j.ID)
	}

	res, err := engine.ClassifyAndExplain(fromName, fromAddr, toStr, j.Subject, bodyText, bodyHTML, true)
	if err != nil {
		return err
	}

	magID := int64(0)
	action := "classified"
	if res.MagnetMatched != nil {
		magID = res.MagnetMatched.ID
		action = "magnet"
	}

	rawHeaders := fmt.Sprintf("From: \"%s\" <%s>\nTo: %s\nSubject: %s\nDate: %s\nMessage-ID: <%s>\nContent-Type: text/plain; charset=utf-8\nX-Text-Classification: %s",
		fromName, fromAddr, toStr, j.Subject, dateStr, msgID, res.BucketName)

	var attachments []db.Attachment
	for _, att := range j.Attachments {
		attachments = append(attachments, db.Attachment{
			Filename:    att.Name,
			ContentType: att.Type,
			Size:        att.Size,
		})
	}

	em := &db.Email{
		UIDL:             uidl,
		MessageID:        msgID,
		FromName:         fromName,
		FromAddress:      fromAddr,
		ToAddress:        toStr,
		Subject:          j.Subject,
		BodyText:         bodyText,
		BodyHTML:         bodyHTML,
		RawHeaders:       rawHeaders,
		Date:             dateStr,
		BucketID:         res.BucketID,
		OriginalBucketID: res.BucketID,
		MagnetID:         magID,
		Confidence:       res.Confidence,
		Scores:           res.Probabilities,
		IsRead:           false,
		Folder:           "received",
		Attachments:      attachments,
	}

	id, err := store.InsertEmail(em)
	if err != nil {
		return err
	}
	store.AddHistoryEvent(id, em.Subject, em.FromAddress, action, "", res.BucketLabel)
	return nil
}

func discoverJMAPSession(client *http.Client, acc *db.Account, diag *NetworkDiagReport) (*JMAPSession, *url.URL, error) {
	sessionBaseURL, hostName, err := resolveJMAPBaseURL(acc)
	if err != nil {
		diag.Add("ERROR", err.Error())
		diag.SetFailure("dns", err)
		return nil, nil, err
	}

	var candidateURLs []*url.URL
	if sessionBaseURL.Path != "" && sessionBaseURL.Path != "/" {
		// Specific URL given by user (e.g. https://api.fastmail.com/jmap/session)
		candidateURLs = append(candidateURLs, sessionBaseURL)
	} else {
		// Standard RFC 8620 discovery paths
		wellKnown, _ := sessionBaseURL.Parse("/.well-known/jmap")
		jmapSession, _ := sessionBaseURL.Parse("/jmap/session")
		candidateURLs = append(candidateURLs, wellKnown, jmapSession)
	}

	diag.Add("INFO", fmt.Sprintf("Initiating JMAP Session discovery for %s (candidate endpoints: %d)", hostName, len(candidateURLs)))

	var lastErr error
	for _, u := range candidateURLs {
		req, err := http.NewRequestWithContext(context.Background(), "GET", u.String(), nil)
		if err != nil {
			continue
		}
		diag.Add("C->S", fmt.Sprintf("GET %s (JMAP Session Discovery)", u.RequestURI()))
		resp, body, err := doJMAPRequest(client, req, acc, diag)
		if err != nil {
			lastErr = err
			if resp != nil && (resp.StatusCode == 401 || resp.StatusCode == 403) {
				return nil, nil, err
			}
			continue
		}

		var sess JMAPSession
		if err := json.Unmarshal(body, &sess); err != nil {
			lastErr = fmt.Errorf("JMAPセッションJSONのデコードに失敗しました: %w", err)
			diag.Add("ERROR", lastErr.Error())
			continue
		}

		if sess.APIURL == "" {
			lastErr = fmt.Errorf("JMAPセッション内に apiUrl が含まれていません")
			diag.Add("ERROR", lastErr.Error())
			continue
		}

		diag.Add("INFO", fmt.Sprintf("JMAP Session established successfully (Accounts: %d, apiUrl: %s)", len(sess.Accounts), sess.APIURL))
		return &sess, u, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("JMAPセッションディスカバリーに失敗しました (ホスト: %s)", hostName)
	}
	if diag.ErrorStage == "" {
		diag.SetFailure("command", lastErr)
	}
	return nil, nil, lastErr
}

func doJMAPRequest(client *http.Client, req *http.Request, acc *db.Account, diag *NetworkDiagReport) (*http.Response, []byte, error) {
	authHeader := getJMAPAuthHeader(acc)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	req.Header.Set("Accept", "application/json")

	displayAuth := "None"
	if authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			displayAuth = fmt.Sprintf("Bearer ******** (masked, %d chars)", len(acc.Password))
		} else {
			displayAuth = fmt.Sprintf("Basic (user: %s, pass: ********)", acc.Username)
		}
	}

	diag.Add("C->S", fmt.Sprintf("%s %s (Auth: %s)", req.Method, req.URL.RequestURI(), displayAuth))

	resp, err := client.Do(req)
	if err != nil {
		diag.Add("ERROR", fmt.Sprintf("HTTP request error (%s %s): %v", req.Method, req.URL.String(), err))
		if diag.ErrorStage == "" {
			diag.SetFailure("tcp_connect", err)
		}
		return nil, nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		diag.Add("ERROR", fmt.Sprintf("Reading response body error: %v", err))
		return resp, nil, err
	}

	diag.Add("S->C", fmt.Sprintf("HTTP %d %s (%d bytes response)", resp.StatusCode, resp.Status, len(bodyBytes)))

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		authErr := fmt.Errorf("JMAP認証エラー (HTTP %d %s): ユーザー名またはパスワード・APIトークンがサーバーに拒否されました", resp.StatusCode, resp.Status)
		diag.Add("ERROR", authErr.Error())
		diag.SetFailure("auth", authErr)
		return resp, bodyBytes, authErr
	}

	if resp.StatusCode >= 400 {
		httpErr := fmt.Errorf("JMAP HTTPエラー: %s", resp.Status)
		diag.Add("ERROR", fmt.Sprintf("Server returned error response (HTTP %d): %s", resp.StatusCode, string(bodyBytes)))
		return resp, bodyBytes, httpErr
	}

	return resp, bodyBytes, nil
}

func getJMAPAuthHeader(acc *db.Account) string {
	pass := strings.TrimSpace(acc.Password)
	if pass == "" {
		return ""
	}
	if strings.HasPrefix(pass, "Bearer ") {
		return pass
	}
	user := strings.TrimSpace(acc.Username)
	if user == "" || strings.EqualFold(user, "bearer") || strings.EqualFold(user, "token") {
		return "Bearer " + pass
	}
	if strings.HasPrefix(pass, "fmu") || strings.HasPrefix(pass, "jmap-") {
		return "Bearer " + pass
	}
	creds := user + ":" + pass
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(creds))
}

func resolveJMAPBaseURL(acc *db.Account) (*url.URL, string, error) {
	rawHost := strings.TrimSpace(acc.IncomingHost)
	if rawHost == "" {
		return nil, "", fmt.Errorf("JMAP受信サーバー (Host) が空です")
	}

	if strings.HasPrefix(rawHost, "http://") || strings.HasPrefix(rawHost, "https://") {
		u, err := url.Parse(rawHost)
		if err != nil {
			return nil, "", fmt.Errorf("無効なJMAP URL: %w", err)
		}
		return u, u.Hostname(), nil
	}

	scheme := "https"
	if !acc.IncomingTLS {
		scheme = "http"
	}

	port := acc.IncomingPort
	if port <= 0 {
		if acc.IncomingTLS {
			port = 443
		} else {
			port = 80
		}
	}

	var hostPort string
	if (scheme == "https" && port == 443) || (scheme == "http" && port == 80) {
		hostPort = rawHost
	} else {
		hostPort = net.JoinHostPort(rawHost, strconv.Itoa(port))
	}

	u, err := url.Parse(fmt.Sprintf("%s://%s", scheme, hostPort))
	if err != nil {
		return nil, "", err
	}
	return u, rawHost, nil
}

func createJMAPHTTPClient(acc *db.Account, diag *NetworkDiagReport) (*http.Client, string) {
	_, hostName, _ := resolveJMAPBaseURL(acc)

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			h, pStr, err := net.SplitHostPort(addr)
			if err != nil {
				h = addr
				pStr = "80"
			}
			p, _ := strconv.Atoi(pStr)
			return dialMailServerWithDiag(h, p, false, diag)
		},
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			h, pStr, err := net.SplitHostPort(addr)
			if err != nil {
				h = addr
				pStr = "443"
			}
			p, _ := strconv.Atoi(pStr)
			return dialMailServerWithDiag(h, p, true, diag)
		},
		TLSClientConfig: &tls.Config{
			ServerName: hostName,
		},
		ResponseHeaderTimeout: 30 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   45 * time.Second,
	}, hostName
}
