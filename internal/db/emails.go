package db

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

type EmailFilter struct {
	Folder     string
	BucketID   int64
	UnreadOnly bool
	Starred    bool
	Search     string
	Limit      int
}

func (s *Store) InsertEmail(e *Email) (int64, error) {
	if e.Scores != nil {
		b, _ := json.Marshal(e.Scores)
		e.ScoresJSON = string(b)
	} else if e.ScoresJSON == "" {
		e.ScoresJSON = "{}"
	}
	if e.Folder == "" {
		e.Folder = "received"
	}
	if e.OriginalBucketID == 0 {
		e.OriginalBucketID = e.BucketID
	}

	isRead := 0
	if e.IsRead {
		isRead = 1
	}
	isStarred := 0
	if e.IsStarred {
		isStarred = 1
	}
	isTrained := 0
	if e.IsTrained {
		isTrained = 1
	}
	isReclass := 0
	if e.IsReclassified {
		isReclass = 1
	}

	if e.ID > 0 {
		_, err := s.DB.Exec(`
			INSERT OR REPLACE INTO emails (
				id, uidl, message_id, from_name, from_address, to_address,
				subject, body_text, body_html, raw_headers, date,
				bucket_id, original_bucket_id, magnet_id, confidence, scores_json,
				is_read, is_starred, is_trained, is_reclassified, folder
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			e.ID, e.UIDL, e.MessageID, e.FromName, e.FromAddress, e.ToAddress,
			e.Subject, e.BodyText, e.BodyHTML, e.RawHeaders, e.Date,
			e.BucketID, e.OriginalBucketID, e.MagnetID, e.Confidence, e.ScoresJSON,
			isRead, isStarred, isTrained, isReclass, e.Folder,
		)
		if err == nil && len(e.Attachments) > 0 {
			_, _ = s.DB.Exec("DELETE FROM attachments WHERE email_id = ?", e.ID)
			for _, att := range e.Attachments {
				raw := att.Data
				if len(raw) == 0 && att.DataBase64 != "" {
					raw, _ = base64.StdEncoding.DecodeString(att.DataBase64)
				}
				_, _ = s.InsertAttachment(e.ID, att.Filename, att.ContentType, raw)
			}
		}
		return e.ID, err
	}

	res, err := s.DB.Exec(`
		INSERT INTO emails (
			uidl, message_id, from_name, from_address, to_address,
			subject, body_text, body_html, raw_headers, date,
			bucket_id, original_bucket_id, magnet_id, confidence, scores_json,
			is_read, is_starred, is_trained, is_reclassified, folder
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		e.UIDL, e.MessageID, e.FromName, e.FromAddress, e.ToAddress,
		e.Subject, e.BodyText, e.BodyHTML, e.RawHeaders, e.Date,
		e.BucketID, e.OriginalBucketID, e.MagnetID, e.Confidence, e.ScoresJSON,
		isRead, isStarred, isTrained, isReclass, e.Folder,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	e.ID = id
	if len(e.Attachments) > 0 {
		for _, att := range e.Attachments {
			raw := att.Data
			if len(raw) == 0 && att.DataBase64 != "" {
				raw, _ = base64.StdEncoding.DecodeString(att.DataBase64)
			}
			_, _ = s.InsertAttachment(id, att.Filename, att.ContentType, raw)
		}
	}
	return id, err
}

func (s *Store) ExistsByUIDLOrMessageID(uidl, messageID string) bool {
	if uidl != "" {
		var c int
		if err := s.DB.QueryRow("SELECT COUNT(*) FROM emails WHERE uidl = ?", uidl).Scan(&c); err == nil && c > 0 {
			return true
		}
	}
	if messageID != "" {
		var c int
		if err := s.DB.QueryRow("SELECT COUNT(*) FROM emails WHERE message_id = ?", messageID).Scan(&c); err == nil && c > 0 {
			return true
		}
	}
	return false
}

func (s *Store) ListEmails(f EmailFilter) ([]Email, error) {
	var where []string
	var args []any

	if f.Folder != "" && f.Folder != "all" {
		where = append(where, "e.folder = ?")
		args = append(args, f.Folder)
	} else if f.Folder == "" {
		where = append(where, "e.folder = 'received'")
	}
	if f.BucketID > 0 {
		where = append(where, "e.bucket_id = ?")
		args = append(args, f.BucketID)
	}
	if f.UnreadOnly {
		where = append(where, "e.is_read = 0")
	}
	if f.Starred {
		where = append(where, "e.is_starred = 1")
	}
	if strings.TrimSpace(f.Search) != "" {
		q := "%" + strings.TrimSpace(f.Search) + "%"
		where = append(where, "(e.subject LIKE ? OR e.from_address LIKE ? OR e.from_name LIKE ? OR e.body_text LIKE ?)")
		args = append(args, q, q, q, q)
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}
	limit := f.Limit
	if limit <= 0 {
		limit = 300
	}

	query := fmt.Sprintf(`
		SELECT 
			e.id, e.uidl, e.message_id, e.from_name, e.from_address, e.to_address,
			e.subject, e.body_text, e.body_html, e.raw_headers, e.date,
			e.bucket_id, b.name, b.label, b.color,
			e.original_bucket_id, COALESCE(ob.name, ''),
			e.magnet_id, COALESCE(m.pattern, ''),
			e.confidence, e.scores_json,
			e.is_read, e.is_starred, e.is_trained, e.is_reclassified, e.folder,
			(SELECT COUNT(*) FROM attachments a WHERE a.email_id = e.id) as attachment_count
		FROM emails e
		JOIN buckets b ON e.bucket_id = b.id
		LEFT JOIN buckets ob ON e.original_bucket_id = ob.id
		LEFT JOIN magnets m ON e.magnet_id = m.id
		%s
		ORDER BY e.date DESC, e.id DESC
		LIMIT %d
	`, whereClause, limit)

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Email
	for rows.Next() {
		e, err := scanEmailRow(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *e)
	}
	return list, nil
}

func (s *Store) GetEmail(id int64) (*Email, error) {
	query := `
		SELECT 
			e.id, e.uidl, e.message_id, e.from_name, e.from_address, e.to_address,
			e.subject, e.body_text, e.body_html, e.raw_headers, e.date,
			e.bucket_id, b.name, b.label, b.color,
			e.original_bucket_id, COALESCE(ob.name, ''),
			e.magnet_id, COALESCE(m.pattern, ''),
			e.confidence, e.scores_json,
			e.is_read, e.is_starred, e.is_trained, e.is_reclassified, e.folder,
			(SELECT COUNT(*) FROM attachments a WHERE a.email_id = e.id) as attachment_count
		FROM emails e
		JOIN buckets b ON e.bucket_id = b.id
		LEFT JOIN buckets ob ON e.original_bucket_id = ob.id
		LEFT JOIN magnets m ON e.magnet_id = m.id
		WHERE e.id = ?
	`
	row := s.DB.QueryRow(query, id)
	em, err := scanEmailSingle(row)
	if err != nil {
		return nil, err
	}
	em.Attachments, _ = s.GetAttachmentsByEmail(id)
	return em, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanEmailCommon(r rowScanner) (*Email, error) {
	var e Email
	var isRead, isStarred, isTrained, isReclass int
	if err := r.Scan(
		&e.ID, &e.UIDL, &e.MessageID, &e.FromName, &e.FromAddress, &e.ToAddress,
		&e.Subject, &e.BodyText, &e.BodyHTML, &e.RawHeaders, &e.Date,
		&e.BucketID, &e.BucketName, &e.BucketLabel, &e.BucketColor,
		&e.OriginalBucketID, &e.OriginalBucketName,
		&e.MagnetID, &e.MagnetDesc,
		&e.Confidence, &e.ScoresJSON,
		&isRead, &isStarred, &isTrained, &isReclass, &e.Folder,
		&e.AttachmentCount,
	); err != nil {
		return nil, err
	}
	e.IsRead = isRead == 1
	e.IsStarred = isStarred == 1
	e.IsTrained = isTrained == 1
	e.IsReclassified = isReclass == 1
	e.Scores = make(map[string]float64)
	if e.ScoresJSON != "" {
		_ = json.Unmarshal([]byte(e.ScoresJSON), &e.Scores)
	}
	return &e, nil
}

func (s *Store) InsertAttachment(emailID int64, filename, contentType string, data []byte) (int64, error) {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		filename = "attachment.bin"
	}
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	res, err := s.DB.Exec(
		`INSERT INTO attachments (email_id, filename, content_type, size, data) VALUES (?, ?, ?, ?, ?)`,
		emailID, filename, contentType, len(data), data,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) GetAttachmentsByEmail(emailID int64) ([]Attachment, error) {
	rows, err := s.DB.Query(
		`SELECT id, email_id, filename, content_type, size FROM attachments WHERE email_id = ? ORDER BY id ASC`,
		emailID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Attachment
	for rows.Next() {
		var a Attachment
		if err := rows.Scan(&a.ID, &a.EmailID, &a.Filename, &a.ContentType, &a.Size); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, nil
}

func (s *Store) GetAttachmentsWithDataByEmail(emailID int64) ([]Attachment, error) {
	rows, err := s.DB.Query(
		`SELECT id, email_id, filename, content_type, size, data FROM attachments WHERE email_id = ? ORDER BY id ASC`,
		emailID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Attachment
	for rows.Next() {
		var a Attachment
		if err := rows.Scan(&a.ID, &a.EmailID, &a.Filename, &a.ContentType, &a.Size, &a.Data); err != nil {
			return nil, err
		}
		a.DataBase64 = base64.StdEncoding.EncodeToString(a.Data)
		list = append(list, a)
	}
	return list, nil
}

func (s *Store) GetAttachment(id int64) (*Attachment, error) {
	var a Attachment
	err := s.DB.QueryRow(
		`SELECT id, email_id, filename, content_type, size, data FROM attachments WHERE id = ?`,
		id,
	).Scan(&a.ID, &a.EmailID, &a.Filename, &a.ContentType, &a.Size, &a.Data)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func scanEmailRow(rows *sql.Rows) (*Email, error) {
	return scanEmailCommon(rows)
}

func scanEmailSingle(row *sql.Row) (*Email, error) {
	return scanEmailCommon(row)
}

func (s *Store) MarkRead(id int64, read bool) error {
	val := 0
	if read {
		val = 1
	}
	_, err := s.DB.Exec("UPDATE emails SET is_read = ? WHERE id = ?", val, id)
	return err
}

func (s *Store) MarkStarred(id int64, starred bool) error {
	val := 0
	if starred {
		val = 1
	}
	_, err := s.DB.Exec("UPDATE emails SET is_starred = ? WHERE id = ?", val, id)
	return err
}

func (s *Store) MoveFolder(id int64, folder string) error {
	_, err := s.DB.Exec("UPDATE emails SET folder = ? WHERE id = ?", folder, id)
	return err
}

func (s *Store) DeleteEmailPermanently(id int64) error {
	_, err := s.DB.Exec("DELETE FROM emails WHERE id = ?", id)
	return err
}

func (s *Store) GetAccounts() ([]Account, error) {
	rows, err := s.DB.Query(`
		SELECT id, name, email_address, protocol, incoming_host, incoming_port, incoming_tls,
		       username, password, smtp_host, smtp_port, smtp_tls, enabled,
		       check_interval, COALESCE(last_checked_at, ''), COALESCE(last_check_status, '')
		FROM accounts ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Account
	for rows.Next() {
		var a Account
		var inTLS, smtpTLS, en int
		if err := rows.Scan(
			&a.ID, &a.Name, &a.EmailAddress, &a.Protocol, &a.IncomingHost, &a.IncomingPort, &inTLS,
			&a.Username, &a.Password, &a.SMTPHost, &a.SMTPPort, &smtpTLS, &en,
			&a.CheckInterval, &a.LastCheckedAt, &a.LastCheckStatus,
		); err != nil {
			return nil, err
		}
		a.IncomingTLS = inTLS == 1
		a.SMTPTLS = smtpTLS == 1
		a.Enabled = en == 1
		list = append(list, a)
	}
	return list, nil
}

func (s *Store) SaveAccount(a *Account) (*Account, error) {
	inTLS, smtpTLS, en := 0, 0, 0
	if a.IncomingTLS {
		inTLS = 1
	}
	if a.SMTPTLS {
		smtpTLS = 1
	}
	if a.Enabled {
		en = 1
	}

	if a.ID > 0 {
		_, err := s.DB.Exec(`
			UPDATE accounts SET
				name=?, email_address=?, protocol=?, incoming_host=?, incoming_port=?, incoming_tls=?,
				username=?, password=?, smtp_host=?, smtp_port=?, smtp_tls=?, enabled=?,
				check_interval=?
			WHERE id=?
		`, a.Name, a.EmailAddress, a.Protocol, a.IncomingHost, a.IncomingPort, inTLS,
			a.Username, a.Password, a.SMTPHost, a.SMTPPort, smtpTLS, en, a.CheckInterval, a.ID)
		return a, err
	}

	res, err := s.DB.Exec(`
		INSERT INTO accounts (
			name, email_address, protocol, incoming_host, incoming_port, incoming_tls,
			username, password, smtp_host, smtp_port, smtp_tls, enabled,
			check_interval, last_checked_at, last_check_status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, a.Name, a.EmailAddress, a.Protocol, a.IncomingHost, a.IncomingPort, inTLS,
		a.Username, a.Password, a.SMTPHost, a.SMTPPort, smtpTLS, en,
		a.CheckInterval, a.LastCheckedAt, a.LastCheckStatus)
	if err != nil {
		return nil, err
	}
	a.ID, _ = res.LastInsertId()
	return a, nil
}

func (s *Store) UpdateAccountCheckStatus(id int64, lastCheckedAt, lastCheckStatus string) error {
	_, err := s.DB.Exec(`
		UPDATE accounts SET last_checked_at = ?, last_check_status = ? WHERE id = ?
	`, lastCheckedAt, lastCheckStatus, id)
	return err
}

func (s *Store) DeleteAccount(id int64) error {
	_, err := s.DB.Exec("DELETE FROM accounts WHERE id = ?", id)
	return err
}
