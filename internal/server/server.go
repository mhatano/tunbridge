package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"tunbridge/internal/classifier"
	"tunbridge/internal/db"
	"tunbridge/internal/mail"
)

type Server struct {
	Store  *db.Store
	Engine *classifier.Engine
	UIFS   fs.FS
}

func NewServer(store *db.Store, engine *classifier.Engine, uiFS fs.FS) *Server {
	return &Server{
		Store:  store,
		Engine: engine,
		UIFS:   uiFS,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Overview & Stats
	mux.HandleFunc("/api/overview", s.handleOverview)
	mux.HandleFunc("/api/history", s.handleHistory)

	// Emails & MUA actions
	mux.HandleFunc("/api/emails", s.handleListEmails)
	mux.HandleFunc("/api/emails/detail", s.handleEmailDetail)
	mux.HandleFunc("/api/emails/reclassify", s.handleReclassify)
	mux.HandleFunc("/api/emails/mark", s.handleMarkEmail)
	mux.HandleFunc("/api/emails/send", s.handleSendEmail)
	mux.HandleFunc("/api/emails/import", s.handleImportEML)
	mux.HandleFunc("/api/emails/export", s.handleExportEML)
	mux.HandleFunc("/api/emails/attachment", s.handleDownloadAttachment)
	mux.HandleFunc("/api/emails/simulate", s.handleSimulateIncoming)
	mux.HandleFunc("/api/emails/rescan", s.handleRescan)

	// Buckets
	mux.HandleFunc("/api/buckets", s.handleGetBuckets)
	mux.HandleFunc("/api/buckets/create", s.handleCreateBucket)
	mux.HandleFunc("/api/buckets/update", s.handleUpdateBucket)
	mux.HandleFunc("/api/buckets/clear", s.handleClearBucket)
	mux.HandleFunc("/api/buckets/delete", s.handleDeleteBucket)

	// Magnets
	mux.HandleFunc("/api/magnets", s.handleGetMagnets)
	mux.HandleFunc("/api/magnets/create", s.handleCreateMagnet)
	mux.HandleFunc("/api/magnets/toggle", s.handleToggleMagnet)
	mux.HandleFunc("/api/magnets/delete", s.handleDeleteMagnet)

	// Corpus & Word Inspector
	mux.HandleFunc("/api/corpus", s.handleCorpus)
	mux.HandleFunc("/api/corpus/lookup", s.handleWordLookup)
	mux.HandleFunc("/api/corpus/test", s.handleCorpusTest)

	// POP3 / IMAP / SMTP Accounts
	mux.HandleFunc("/api/accounts", s.handleGetAccounts)
	mux.HandleFunc("/api/accounts/save", s.handleSaveAccount)
	mux.HandleFunc("/api/accounts/delete", s.handleDeleteAccount)
	mux.HandleFunc("/api/accounts/fetch", s.handleFetchAccount)

	// Static UI
	mux.Handle("/", http.FileServer(http.FS(s.UIFS)))

	return mux
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	buckets, err := s.Store.GetBuckets()
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	stats, err := s.Engine.GetStats()
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	var unreadTotal, starredTotal, sentTotal, trashTotal int
	_ = s.Store.DB.QueryRow("SELECT COUNT(*) FROM emails WHERE folder = 'received' AND is_read = 0").Scan(&unreadTotal)
	_ = s.Store.DB.QueryRow("SELECT COUNT(*) FROM emails WHERE is_starred = 1 AND folder != 'trash'").Scan(&starredTotal)
	_ = s.Store.DB.QueryRow("SELECT COUNT(*) FROM emails WHERE folder = 'sent'").Scan(&sentTotal)
	_ = s.Store.DB.QueryRow("SELECT COUNT(*) FROM emails WHERE folder = 'trash'").Scan(&trashTotal)

	writeJSON(w, 200, map[string]any{
		"buckets": buckets,
		"stats":   stats,
		"counts": map[string]int{
			"received": stats.TotalEmails,
			"unread":   unreadTotal,
			"starred":  starredTotal,
			"sent":     sentTotal,
			"trash":    trashTotal,
		},
	})
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	events, err := s.Store.GetHistoryEvents(80)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"events": events})
}

func (s *Server) handleListEmails(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	folder := q.Get("folder")
	bucketID, _ := strconv.ParseInt(q.Get("bucket_id"), 10, 64)
	unreadOnly := q.Get("unread") == "1"
	starred := q.Get("starred") == "1"
	search := q.Get("search")

	emails, err := s.Store.ListEmails(db.EmailFilter{
		Folder:     folder,
		BucketID:   bucketID,
		UnreadOnly: unreadOnly,
		Starred:    starred,
		Search:     search,
	})
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	if emails == nil {
		emails = []db.Email{}
	}
	writeJSON(w, 200, map[string]any{"emails": emails})
}

func (s *Server) handleEmailDetail(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if id <= 0 {
		writeError(w, 400, "invalid id")
		return
	}
	em, err := s.Store.GetEmail(id)
	if err != nil {
		writeError(w, 404, err.Error())
		return
	}
	// Mark as read automatically when opened
	if !em.IsRead {
		_ = s.Store.MarkRead(id, true)
		em.IsRead = true
	}

	explanation, err := s.Engine.ClassifyAndExplain(em.FromName, em.FromAddress, em.ToAddress, em.Subject, em.BodyText, em.BodyHTML, false)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}

	writeJSON(w, 200, map[string]any{
		"email":       em,
		"explanation": explanation,
	})
}

func (s *Server) handleReclassify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EmailIDs []int64 `json:"email_ids"`
		EmailID  int64   `json:"email_id"`
		BucketID int64   `json:"bucket_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	ids := req.EmailIDs
	if len(ids) == 0 && req.EmailID > 0 {
		ids = []int64{req.EmailID}
	}
	if len(ids) == 0 || req.BucketID <= 0 {
		writeError(w, 400, "email_id and bucket_id required")
		return
	}

	var lastEmail *db.Email
	for _, id := range ids {
		em, err := s.Engine.ReclassifyEmail(id, req.BucketID)
		if err != nil {
			writeError(w, 500, err.Error())
			return
		}
		lastEmail = em
	}

	writeJSON(w, 200, map[string]any{
		"email":   lastEmail,
		"updated": len(ids),
	})
}

func (s *Server) handleMarkEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EmailIDs         []int64    `json:"email_ids"`
		Action           string     `json:"action"` // "read", "unread", "star", "unstar", "trash", "restore", "delete"
		RestoreFolder    string     `json:"restore_folder"`
		RestoreSnapshots []db.Email `json:"restore_snapshots"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	var deletedSnapshots []db.Email
	for _, id := range req.EmailIDs {
		switch req.Action {
		case "read":
			_ = s.Store.MarkRead(id, true)
		case "unread":
			_ = s.Store.MarkRead(id, false)
		case "star":
			_ = s.Store.MarkStarred(id, true)
		case "unstar":
			_ = s.Store.MarkStarred(id, false)
		case "trash":
			_ = s.Store.MoveFolder(id, "trash")
		case "restore":
			targetFolder := req.RestoreFolder
			if targetFolder == "" {
				targetFolder = "received"
			}
			_ = s.Store.MoveFolder(id, targetFolder)
		case "delete":
			if em, err := s.Store.GetEmail(id); err == nil && em != nil {
				em.Attachments, _ = s.Store.GetAttachmentsWithDataByEmail(id)
				deletedSnapshots = append(deletedSnapshots, *em)
			}
			_ = s.Store.DeleteEmailPermanently(id)
		}
	}
	for _, snap := range req.RestoreSnapshots {
		cp := snap
		_, _ = s.Store.InsertEmail(&cp)
	}
	writeJSON(w, 200, map[string]any{"ok": true, "deleted_snapshots": deletedSnapshots})
}

func (s *Server) handleSendEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AccountID   int64           `json:"account_id"`
		To          string          `json:"to"`
		Cc          string          `json:"cc"`
		Bcc         string          `json:"bcc"`
		Subject     string          `json:"subject"`
		Body        string          `json:"body"`
		Attachments []db.Attachment `json:"attachments"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	if strings.TrimSpace(req.To) == "" {
		writeError(w, 400, "宛先メールアドレスを入力してください")
		return
	}

	var selectedAcc *db.Account
	if req.AccountID > 0 {
		accs, _ := s.Store.GetAccounts()
		for _, a := range accs {
			if a.ID == req.AccountID {
				selectedAcc = &a
				break
			}
		}
	}

	em, statusMsg, diag, err := mail.SendEmailWithDiag(selectedAcc, req.To, req.Cc, req.Bcc, req.Subject, req.Body, req.Attachments, s.Store)
	if err != nil {
		var netLogs []*mail.NetworkDiagReport
		if diag != nil {
			netLogs = append(netLogs, diag)
		}
		writeJSON(w, 500, map[string]any{
			"error":    err.Error(),
			"net_logs": netLogs,
		})
		return
	}
	writeJSON(w, 200, map[string]any{
		"email":   em,
		"message": statusMsg,
	})
}

func (s *Server) handleImportEML(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RawEML       string `json:"raw_eml"`
		RawEMLBase64 string `json:"raw_eml_base64"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	var rawBytes []byte
	if strings.TrimSpace(req.RawEMLBase64) != "" {
		b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(req.RawEMLBase64))
		if err != nil {
			writeError(w, 400, "EMLバイナリのBase64デコードに失敗しました: "+err.Error())
			return
		}
		rawBytes = b
	} else if strings.TrimSpace(req.RawEML) != "" {
		rawBytes = []byte(req.RawEML)
	} else {
		writeError(w, 400, "EMLテキストが空です")
		return
	}

	parsed, err := mail.ParseRawEmail(rawBytes)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}

	res, err := s.Engine.ClassifyAndExplain(parsed.FromName, parsed.FromAddress, parsed.ToAddress, parsed.Subject, parsed.BodyText, parsed.BodyHTML, true)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	magID := int64(0)
	action := "classified"
	if res.MagnetMatched != nil {
		magID = res.MagnetMatched.ID
		action = "magnet"
	}

	headers := classifier.InjectPOPFileHeader(parsed.RawHeaders, res.BucketName)
	em := &db.Email{
		UIDL:             fmt.Sprintf("import-%d", time.Now().UnixNano()),
		MessageID:        parsed.MessageID,
		FromName:         parsed.FromName,
		FromAddress:      parsed.FromAddress,
		ToAddress:        parsed.ToAddress,
		Subject:          parsed.Subject,
		BodyText:         parsed.BodyText,
		BodyHTML:         parsed.BodyHTML,
		RawHeaders:       headers,
		Date:             parsed.Date,
		BucketID:         res.BucketID,
		OriginalBucketID: res.BucketID,
		MagnetID:         magID,
		Confidence:       res.Confidence,
		Scores:           res.Probabilities,
		IsRead:           false,
		Folder:           "received",
		Attachments:      parsed.Attachments,
	}
	id, err := s.Store.InsertEmail(em)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	s.Store.AddHistoryEvent(id, em.Subject, em.FromAddress, action, "", res.BucketLabel)
	full, _ := s.Store.GetEmail(id)
	writeJSON(w, 200, map[string]any{"email": full, "classification": res})
}

func (s *Server) handleExportEML(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	em, err := s.Store.GetEmail(id)
	if err != nil {
		writeError(w, 404, err.Error())
		return
	}
	em.Attachments, _ = s.Store.GetAttachmentsWithDataByEmail(id)
	emlContent := mail.FormatEML(em)
	w.Header().Set("Content-Type", "message/rfc822; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="message-%d-%s.eml"`, em.ID, em.BucketName))
	_, _ = w.Write([]byte(emlContent))
}

func (s *Server) handleDownloadAttachment(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if id <= 0 {
		writeError(w, 400, "invalid attachment id")
		return
	}
	att, err := s.Store.GetAttachment(id)
	if err != nil {
		writeError(w, 404, "添付ファイルが見つかりません")
		return
	}
	ct := att.ContentType
	if ct == "" {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)
	dispType := "attachment"
	if r.URL.Query().Get("inline") == "1" {
		dispType = "inline"
	}
	safeEscaped := url.PathEscape(att.Filename)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`%s; filename="%s"; filename*=UTF-8''%s`, dispType, strings.ReplaceAll(att.Filename, `"`, `'`), safeEscaped))
	w.Header().Set("Content-Length", strconv.Itoa(len(att.Data)))
	_, _ = w.Write(att.Data)
}

func (s *Server) handleSimulateIncoming(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Count    int    `json:"count"`
		Category string `json:"category"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	created, err := mail.GenerateIncomingSamples(req.Count, req.Category, s.Store, s.Engine)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"emails": created, "count": len(created)})
}

func (s *Server) handleRescan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OnlyUnclassified bool `json:"only_unclassified"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	repaired, _ := mail.RepairStoredEmails(s.Store, s.Engine)
	updated, err := s.Engine.RescanEmails(req.OnlyUnclassified)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"updated": updated, "repaired": repaired})
}

func (s *Server) handleGetBuckets(w http.ResponseWriter, r *http.Request) {
	buckets, err := s.Store.GetBuckets()
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"buckets": buckets})
}

func (s *Server) handleCreateBucket(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Label       string `json:"label"`
		Color       string `json:"color"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	name := strings.ToLower(strings.TrimSpace(req.Name))
	if name == "" || req.Label == "" {
		writeError(w, 400, "バケット識別子と表示名は必須です")
		return
	}
	if req.Color == "" {
		req.Color = "#06b6d4"
	}
	b, err := s.Store.CreateBucket(name, req.Label, req.Color, req.Description)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"bucket": b})
}

func (s *Server) handleUpdateBucket(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID          int64  `json:"id"`
		Label       string `json:"label"`
		Color       string `json:"color"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	if err := s.Store.UpdateBucket(req.ID, req.Label, req.Color, req.Description); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) handleClearBucket(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	if err := s.Store.ClearBucketCorpus(req.ID); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) handleDeleteBucket(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	if err := s.Store.DeleteBucket(req.ID); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) handleGetMagnets(w http.ResponseWriter, r *http.Request) {
	magnets, err := s.Store.GetMagnets()
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	if magnets == nil {
		magnets = []db.Magnet{}
	}
	writeJSON(w, 200, map[string]any{"magnets": magnets})
}

func (s *Server) handleCreateMagnet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BucketID    int64  `json:"bucket_id"`
		Field       string `json:"field"`
		Pattern     string `json:"pattern"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	if req.BucketID <= 0 || strings.TrimSpace(req.Pattern) == "" {
		writeError(w, 400, "バケットとパターン文字列を指定してください")
		return
	}
	m, err := s.Store.CreateMagnet(req.BucketID, req.Field, strings.TrimSpace(req.Pattern), req.Description)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"magnet": m})
}

func (s *Server) handleToggleMagnet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID      int64 `json:"id"`
		Enabled bool  `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	if err := s.Store.ToggleMagnet(req.ID, req.Enabled); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) handleDeleteMagnet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	if err := s.Store.DeleteMagnet(req.ID); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) handleCorpus(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	bucketID, _ := strconv.ParseInt(q.Get("bucket_id"), 10, 64)
	search := q.Get("search")
	limit, _ := strconv.Atoi(q.Get("limit"))
	if bucketID <= 0 {
		writeError(w, 400, "bucket_id is required")
		return
	}
	words, err := s.Engine.GetCorpusWords(bucketID, search, limit)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	if words == nil {
		words = []classifier.CorpusWordItem{}
	}
	writeJSON(w, 200, map[string]any{"words": words})
}

func (s *Server) handleWordLookup(w http.ResponseWriter, r *http.Request) {
	word := r.URL.Query().Get("word")
	if strings.TrimSpace(word) == "" {
		writeError(w, 400, "word is required")
		return
	}
	res, err := s.Engine.LookupWord(word)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, res)
}

func (s *Server) handleCorpusTest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		From    string `json:"from"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	res, err := s.Engine.ClassifyAndExplain("", req.From, "me@example.jp", req.Subject, req.Body, "", false)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, res)
}

func (s *Server) handleGetAccounts(w http.ResponseWriter, r *http.Request) {
	accs, err := s.Store.GetAccounts()
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	if accs == nil {
		accs = []db.Account{}
	}
	writeJSON(w, 200, map[string]any{"accounts": accs})
}

func (s *Server) handleSaveAccount(w http.ResponseWriter, r *http.Request) {
	var acc db.Account
	if err := json.NewDecoder(r.Body).Decode(&acc); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	saved, err := s.Store.SaveAccount(&acc)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"account": saved})
}

func (s *Server) handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	if err := s.Store.DeleteAccount(req.ID); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) handleFetchAccount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID int64 `json:"id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	accs, err := s.Store.GetAccounts()
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}

	totalFetched := 0
	var errors []string
	var netLogs []*mail.NetworkDiagReport
	for _, a := range accs {
		if req.ID > 0 && a.ID != req.ID {
			continue
		}
		if !a.Enabled && req.ID == 0 {
			continue
		}
		n, diag, err := mail.FetchAccountWithDiag(&a, s.Store, s.Engine)
		totalFetched += n
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", a.Name, err))
			if diag != nil {
				netLogs = append(netLogs, diag)
			}
		}
	}

	writeJSON(w, 200, map[string]any{
		"fetched":  totalFetched,
		"errors":   errors,
		"net_logs": netLogs,
	})
}
