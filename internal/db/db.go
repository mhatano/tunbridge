package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Bucket struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Label       string  `json:"label"`
	Color       string  `json:"color"`
	Description string  `json:"description"`
	IsSystem    bool    `json:"is_system"`
	EmailCount  int     `json:"email_count"`
	UnreadCount int     `json:"unread_count"`
	TrainedDocs int     `json:"trained_docs"`
	TotalWords  int     `json:"total_words"`
	UniqueWords int     `json:"unique_words"`
	FalsePos    int     `json:"false_pos"`
	FalseNeg    int     `json:"false_neg"`
	PriorProb   float64 `json:"prior_prob"`
}

type Magnet struct {
	ID          int64  `json:"id"`
	BucketID    int64  `json:"bucket_id"`
	BucketName  string `json:"bucket_name"`
	BucketLabel string `json:"bucket_label"`
	BucketColor string `json:"bucket_color"`
	Field       string `json:"field"` // "from", "to", "subject", "domain"
	Pattern     string `json:"pattern"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	HitCount    int    `json:"hit_count"`
	CreatedAt   string `json:"created_at"`
}

type Attachment struct {
	ID          int64  `json:"id"`
	EmailID     int64  `json:"email_id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int    `json:"size"`
	Data        []byte `json:"-"`
	DataBase64  string `json:"data_base64,omitempty"`
}

type Email struct {
	ID                 int64              `json:"id"`
	UIDL               string             `json:"uidl"`
	MessageID          string             `json:"message_id"`
	FromName           string             `json:"from_name"`
	FromAddress        string             `json:"from_address"`
	ToAddress          string             `json:"to_address"`
	Subject            string             `json:"subject"`
	BodyText           string             `json:"body_text"`
	BodyHTML           string             `json:"body_html"`
	RawHeaders         string             `json:"raw_headers"`
	Date               string             `json:"date"`
	BucketID           int64              `json:"bucket_id"`
	BucketName         string             `json:"bucket_name"`
	BucketLabel        string             `json:"bucket_label"`
	BucketColor        string             `json:"bucket_color"`
	OriginalBucketID   int64              `json:"original_bucket_id"`
	OriginalBucketName string             `json:"original_bucket_name"`
	MagnetID           int64              `json:"magnet_id"`
	MagnetDesc         string             `json:"magnet_desc"`
	Confidence         float64            `json:"confidence"`
	ScoresJSON         string             `json:"scores_json"`
	IsRead             bool               `json:"is_read"`
	IsStarred          bool               `json:"is_starred"`
	IsTrained          bool               `json:"is_trained"`
	IsReclassified     bool               `json:"is_reclassified"`
	Folder             string             `json:"folder"` // "received", "sent", "trash"
	AttachmentCount    int                `json:"attachment_count"`
	Attachments        []Attachment       `json:"attachments,omitempty"`
	Scores             map[string]float64 `json:"scores,omitempty"`
}

type Account struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	EmailAddress string `json:"email_address"`
	Protocol     string `json:"protocol"` // "pop3" or "imap"
	IncomingHost string `json:"incoming_host"`
	IncomingPort int    `json:"incoming_port"`
	IncomingTLS  bool   `json:"incoming_tls"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPTLS      bool   `json:"smtp_tls"`
	Enabled      bool   `json:"enabled"`
}

type HistoryEvent struct {
	ID         int64  `json:"id"`
	EmailID    int64  `json:"email_id"`
	Subject    string `json:"subject"`
	FromAddr   string `json:"from_addr"`
	Action     string `json:"action"` // "classified", "magnet", "reclassified", "trained"
	FromBucket string `json:"from_bucket"`
	ToBucket   string `json:"to_bucket"`
	CreatedAt  string `json:"created_at"`
}

type Store struct {
	DB *sql.DB
}

func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db: %w", err)
	}
	s := &Store{DB: db}
	if err := s.initSchema(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS buckets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		label TEXT NOT NULL,
		color TEXT NOT NULL,
		description TEXT DEFAULT '',
		is_system INTEGER DEFAULT 0,
		trained_docs INTEGER DEFAULT 0,
		total_words INTEGER DEFAULT 0,
		false_pos INTEGER DEFAULT 0,
		false_neg INTEGER DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS word_counts (
		bucket_id INTEGER NOT NULL,
		word TEXT NOT NULL,
		count INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (bucket_id, word),
		FOREIGN KEY (bucket_id) REFERENCES buckets(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_word_counts_word ON word_counts(word);

	CREATE TABLE IF NOT EXISTS magnets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		bucket_id INTEGER NOT NULL,
		field TEXT NOT NULL,
		pattern TEXT NOT NULL,
		description TEXT DEFAULT '',
		enabled INTEGER DEFAULT 1,
		hit_count INTEGER DEFAULT 0,
		created_at TEXT NOT NULL,
		FOREIGN KEY (bucket_id) REFERENCES buckets(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS emails (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		uidl TEXT DEFAULT '',
		message_id TEXT DEFAULT '',
		from_name TEXT DEFAULT '',
		from_address TEXT DEFAULT '',
		to_address TEXT DEFAULT '',
		subject TEXT DEFAULT '',
		body_text TEXT DEFAULT '',
		body_html TEXT DEFAULT '',
		raw_headers TEXT DEFAULT '',
		date TEXT NOT NULL,
		bucket_id INTEGER NOT NULL,
		original_bucket_id INTEGER NOT NULL,
		magnet_id INTEGER DEFAULT 0,
		confidence REAL DEFAULT 0.0,
		scores_json TEXT DEFAULT '{}',
		is_read INTEGER DEFAULT 0,
		is_starred INTEGER DEFAULT 0,
		is_trained INTEGER DEFAULT 0,
		is_reclassified INTEGER DEFAULT 0,
		folder TEXT DEFAULT 'received'
	);
	CREATE INDEX IF NOT EXISTS idx_emails_bucket ON emails(bucket_id);
	CREATE INDEX IF NOT EXISTS idx_emails_folder ON emails(folder);

	CREATE TABLE IF NOT EXISTS accounts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email_address TEXT NOT NULL,
		protocol TEXT NOT NULL DEFAULT 'pop3',
		incoming_host TEXT NOT NULL,
		incoming_port INTEGER NOT NULL DEFAULT 995,
		incoming_tls INTEGER NOT NULL DEFAULT 1,
		username TEXT NOT NULL,
		password TEXT NOT NULL,
		smtp_host TEXT NOT NULL DEFAULT '',
		smtp_port INTEGER NOT NULL DEFAULT 587,
		smtp_tls INTEGER NOT NULL DEFAULT 1,
		enabled INTEGER NOT NULL DEFAULT 1
	);

	CREATE TABLE IF NOT EXISTS history_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email_id INTEGER DEFAULT 0,
		subject TEXT DEFAULT '',
		from_addr TEXT DEFAULT '',
		action TEXT NOT NULL,
		from_bucket TEXT DEFAULT '',
		to_bucket TEXT DEFAULT '',
		created_at TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS attachments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email_id INTEGER NOT NULL,
		filename TEXT NOT NULL,
		content_type TEXT NOT NULL DEFAULT 'application/octet-stream',
		size INTEGER NOT NULL DEFAULT 0,
		data BLOB NOT NULL,
		FOREIGN KEY (email_id) REFERENCES emails(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_attachments_email ON attachments(email_id);
	`
	if _, err := s.DB.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return s.seedDefaultBuckets()
}

func (s *Store) seedDefaultBuckets() error {
	var count int
	if err := s.DB.QueryRow("SELECT COUNT(*) FROM buckets").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	defaults := []struct {
		name  string
		label string
		color string
		desc  string
		sys   int
	}{
		{"inbox", "通常受信 (Inbox)", "#3b82f6", "個人連絡・一般的な通常メール", 1},
		{"work", "業務・開発 (Work)", "#10b981", "社内連絡・プロジェクト・GitHub・業務システム通知", 0},
		{"newsletter", "メルマガ・情報 (News)", "#8b5cf6", "ニュースレター・ECサイト案内・定期配信", 0},
		{"finance", "金融・請求 (Finance)", "#f59e0b", "銀行明細・クレジットカード・請求書・領収書", 0},
		{"spam", "迷惑メール (Spam)", "#ef4444", "フィッシング・広告スパム・不審な勧誘メール", 1},
		{"unclassified", "未分類 (Unclassified)", "#64748b", "確信度がしきい値未満のメール", 1},
	}

	for _, b := range defaults {
		_, err := s.DB.Exec(
			"INSERT INTO buckets (name, label, color, description, is_system) VALUES (?, ?, ?, ?, ?)",
			b.name, b.label, b.color, b.desc, b.sys,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) GetBuckets() ([]Bucket, error) {
	rows, err := s.DB.Query(`
		SELECT 
			b.id, b.name, b.label, b.color, b.description, b.is_system,
			b.trained_docs, b.total_words, b.false_pos, b.false_neg,
			(SELECT COUNT(*) FROM emails e WHERE e.bucket_id = b.id AND e.folder = 'received') as email_count,
			(SELECT COUNT(*) FROM emails e WHERE e.bucket_id = b.id AND e.folder = 'received' AND e.is_read = 0) as unread_count,
			(SELECT COUNT(*) FROM word_counts wc WHERE wc.bucket_id = b.id AND wc.count > 0) as unique_words
		FROM buckets b
		ORDER BY b.id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var buckets []Bucket
	totalDocs := 0
	for rows.Next() {
		var b Bucket
		var isSys int
		if err := rows.Scan(
			&b.ID, &b.Name, &b.Label, &b.Color, &b.Description, &isSys,
			&b.TrainedDocs, &b.TotalWords, &b.FalsePos, &b.FalseNeg,
			&b.EmailCount, &b.UnreadCount, &b.UniqueWords,
		); err != nil {
			return nil, err
		}
		b.IsSystem = isSys == 1
		totalDocs += b.TrainedDocs
		buckets = append(buckets, b)
	}
	for i := range buckets {
		if totalDocs > 0 {
			buckets[i].PriorProb = float64(buckets[i].TrainedDocs) / float64(totalDocs)
		}
	}
	return buckets, nil
}

func (s *Store) GetBucketByName(name string) (*Bucket, error) {
	buckets, err := s.GetBuckets()
	if err != nil {
		return nil, err
	}
	for _, b := range buckets {
		if b.Name == name {
			return &b, nil
		}
	}
	return nil, fmt.Errorf("bucket not found: %s", name)
}

func (s *Store) GetBucketByID(id int64) (*Bucket, error) {
	buckets, err := s.GetBuckets()
	if err != nil {
		return nil, err
	}
	for _, b := range buckets {
		if b.ID == id {
			return &b, nil
		}
	}
	return nil, fmt.Errorf("bucket not found: %d", id)
}

func (s *Store) CreateBucket(name, label, color, description string) (*Bucket, error) {
	res, err := s.DB.Exec(
		"INSERT INTO buckets (name, label, color, description, is_system) VALUES (?, ?, ?, ?, 0)",
		name, label, color, description,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.GetBucketByID(id)
}

func (s *Store) UpdateBucket(id int64, label, color, description string) error {
	_, err := s.DB.Exec(
		"UPDATE buckets SET label = ?, color = ?, description = ? WHERE id = ?",
		label, color, description, id,
	)
	return err
}

func (s *Store) ClearBucketCorpus(id int64) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM word_counts WHERE bucket_id = ?", id); err != nil {
		return err
	}
	if _, err := tx.Exec("UPDATE buckets SET trained_docs = 0, total_words = 0, false_pos = 0, false_neg = 0 WHERE id = ?", id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) DeleteBucket(id int64) error {
	var isSys int
	if err := s.DB.QueryRow("SELECT is_system FROM buckets WHERE id = ?", id).Scan(&isSys); err != nil {
		return err
	}
	if isSys == 1 {
		return fmt.Errorf("システム標準バケットは削除できません")
	}
	unclassified, err := s.GetBucketByName("unclassified")
	if err != nil {
		return err
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("UPDATE emails SET bucket_id = ? WHERE bucket_id = ?", unclassified.ID, id); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM word_counts WHERE bucket_id = ?", id); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM magnets WHERE bucket_id = ?", id); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM buckets WHERE id = ?", id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) GetMagnets() ([]Magnet, error) {
	rows, err := s.DB.Query(`
		SELECT m.id, m.bucket_id, b.name, b.label, b.color, m.field, m.pattern, m.description, m.enabled, m.hit_count, m.created_at
		FROM magnets m
		JOIN buckets b ON m.bucket_id = b.id
		ORDER BY m.id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Magnet
	for rows.Next() {
		var m Magnet
		var en int
		if err := rows.Scan(&m.ID, &m.BucketID, &m.BucketName, &m.BucketLabel, &m.BucketColor, &m.Field, &m.Pattern, &m.Description, &en, &m.HitCount, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.Enabled = en == 1
		list = append(list, m)
	}
	return list, nil
}

func (s *Store) CreateMagnet(bucketID int64, field, pattern, description string) (*Magnet, error) {
	now := time.Now().Format("2006-01-02 15:04:05")
	res, err := s.DB.Exec(
		"INSERT INTO magnets (bucket_id, field, pattern, description, enabled, hit_count, created_at) VALUES (?, ?, ?, ?, 1, 0, ?)",
		bucketID, field, pattern, description, now,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	magnets, err := s.GetMagnets()
	if err != nil {
		return nil, err
	}
	for _, m := range magnets {
		if m.ID == id {
			return &m, nil
		}
	}
	return nil, nil
}

func (s *Store) ToggleMagnet(id int64, enabled bool) error {
	en := 0
	if enabled {
		en = 1
	}
	_, err := s.DB.Exec("UPDATE magnets SET enabled = ? WHERE id = ?", en, id)
	return err
}

func (s *Store) DeleteMagnet(id int64) error {
	_, err := s.DB.Exec("DELETE FROM magnets WHERE id = ?", id)
	return err
}

func (s *Store) AddHistoryEvent(emailID int64, subject, fromAddr, action, fromBucket, toBucket string) {
	now := time.Now().Format("2006-01-02 15:04:05")
	_, _ = s.DB.Exec(
		"INSERT INTO history_events (email_id, subject, from_addr, action, from_bucket, to_bucket, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		emailID, subject, fromAddr, action, fromBucket, toBucket, now,
	)
}

func (s *Store) GetHistoryEvents(limit int) ([]HistoryEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.DB.Query(`
		SELECT id, email_id, subject, from_addr, action, from_bucket, to_bucket, created_at
		FROM history_events
		ORDER BY id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []HistoryEvent
	for rows.Next() {
		var ev HistoryEvent
		if err := rows.Scan(&ev.ID, &ev.EmailID, &ev.Subject, &ev.FromAddr, &ev.Action, &ev.FromBucket, &ev.ToBucket, &ev.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, ev)
	}
	return events, nil
}
