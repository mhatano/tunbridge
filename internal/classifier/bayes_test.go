package classifier_test

import (
	"path/filepath"
	"strings"
	"testing"

	"tunbridge/internal/classifier"
	"tunbridge/internal/db"
	"tunbridge/internal/mail"
)

func TestPOPFileClassifierAndTOE(t *testing.T) {
	tmpDB := filepath.Join(t.TempDir(), "test_popfile.db")
	store, err := db.NewStore(tmpDB)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.DB.Close()

	engine := classifier.NewEngine(store)
	if err := mail.SeedInitialData(store, engine); err != nil {
		t.Fatalf("SeedInitialData failed: %v", err)
	}

	// 1. Test Bayesian spam classification
	spamRes, err := engine.ClassifyAndExplain(
		"当選センター",
		"promo@free-lottery.xyz",
		"me@example.jp",
		"【完全無料】現金1,000万円が当選しました！至急口座登録してください",
		"特別抽選で貴方様が当選しました。完全無料で即日振込いたします。http://claim-bonus-now.xyz/win",
		"",
		false,
	)
	if err != nil {
		t.Fatalf("ClassifyAndExplain failed: %v", err)
	}
	if spamRes.BucketName != "spam" {
		t.Errorf("Expected spam bucket, got %s (confidence %.4f)", spamRes.BucketName, spamRes.Confidence)
	}
	if len(spamRes.TopWords) == 0 {
		t.Errorf("Expected top contributing words to be populated")
	}

	// 2. Test Bayesian work classification
	workRes, err := engine.ClassifyAndExplain(
		"開発チーム",
		"dev@corp-tech.co.jp",
		"me@example.jp",
		"スプリントレビュー議事録とステージング環境デプロイ手順",
		"プルリクエストのレビューとAPI仕様書の確認をお願いします。負荷テストを実施します。",
		"",
		false,
	)
	if err != nil {
		t.Fatalf("ClassifyAndExplain failed: %v", err)
	}
	if workRes.BucketName != "work" {
		t.Errorf("Expected work bucket, got %s", workRes.BucketName)
	}

	// 3. Test Magnet priority override
	magRes, err := engine.ClassifyAndExplain(
		"VIP Partner",
		"anyone@vip-partner.co.jp",
		"me@example.jp",
		"完全無料のお知らせ", // Even with spam-like subject, domain magnet routes to work!
		"無料キャンペーンのお知らせです",
		"",
		false,
	)
	if err != nil {
		t.Fatalf("ClassifyAndExplain failed: %v", err)
	}
	if magRes.MagnetMatched == nil || magRes.BucketName != "work" {
		t.Errorf("Expected magnet match to 'work', got bucket=%s magnet=%v", magRes.BucketName, magRes.MagnetMatched)
	}

	// 4. Test TOE Reclassification
	emails, err := store.ListEmails(db.EmailFilter{Folder: "received"})
	if err != nil || len(emails) == 0 {
		t.Fatalf("Expected seeded emails, err=%v", err)
	}
	targetEmail := emails[0]
	financeBucket, err := store.GetBucketByName("finance")
	if err != nil {
		t.Fatalf("GetBucketByName failed: %v", err)
	}

	reclassified, err := engine.ReclassifyEmail(targetEmail.ID, financeBucket.ID)
	if err != nil {
		t.Fatalf("ReclassifyEmail failed: %v", err)
	}
	if reclassified.BucketID != financeBucket.ID {
		t.Errorf("Expected bucket_id %d after TOE, got %d", financeBucket.ID, reclassified.BucketID)
	}
	if !strings.Contains(reclassified.RawHeaders, "X-Text-Classification: finance") {
		t.Errorf("Expected X-Text-Classification: finance in headers, got:\n%s", reclassified.RawHeaders)
	}
}
