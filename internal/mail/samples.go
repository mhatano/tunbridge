package mail

import (
	"fmt"
	"math/rand"
	"time"

	"tunbridge/internal/classifier"
	"tunbridge/internal/db"
)

type SampleTemplate struct {
	Category    string
	FromName    string
	FromAddress string
	ToAddress   string
	Subject     string
	BodyText    string
	BodyHTML    string
}

var initialCorpusTemplates = []SampleTemplate{
	// INBOX (個人・日常連絡)
	{
		Category:    "inbox",
		FromName:    "佐藤 健太",
		FromAddress: "kenta.sato@example.jp",
		ToAddress:   "me@example.jp",
		Subject:     "来週末の温泉旅行の宿と新幹線の予約について",
		BodyText:    "お疲れ様です、佐藤です。\n来週末の箱根温泉旅行ですが、旅館の予約が完了しました！\n土曜日の朝9時東京駅集合でロマンスカーまたは新幹線で向かいましょう。夕食は懐石料理コースにしてあります。何かアレルギー等あれば教えてください。",
	},
	{
		Category:    "inbox",
		FromName:    "高橋 美咲",
		FromAddress: "misaki.takahashi@friends.jp",
		ToAddress:   "me@example.jp",
		Subject:     "先日の写真データ共有と次回ランチの相談",
		BodyText:    "こんにちは！日曜日はカフェでゆっくり話せて楽しかったです。\n撮影した写真をアルバムにまとめましたので時間があるときに見てくださいね。来月また表参道のイタリアンでランチしましょう！",
	},
	{
		Category:    "inbox",
		FromName:    "母 (実家)",
		FromAddress: "mom.family@home-mail.jp",
		ToAddress:   "me@example.jp",
		Subject:     "季節の果物と新米を宅配便で送りました",
		BodyText:    "元気で過ごしていますか？親戚の農家から届いた新米コシヒカリと梨を段ボールに詰めてクロネコヤマトで発送しました。明日の夜には届くと思います。寒くなってきたので体調に気をつけてね。",
	},
	{
		Category:    "inbox",
		FromName:    "Alex Rivera",
		FromAddress: "alex.rivera@personal-mail.org",
		ToAddress:   "me@example.jp",
		Subject:     "Dinner plans on Friday evening & book recommendation",
		BodyText:    "Hey! Are we still on for dinner this Friday at the ramen place near Shibuya station? Also, I finished reading the sci-fi novel you lent me last month and loved it. Let me know what time works best for you.",
	},

	// WORK (業務・開発・プロジェクト)
	{
		Category:    "work",
		FromName:    "鈴木 一郎 (開発部PM)",
		FromAddress: "ichiro.suzuki@corp-tech.co.jp",
		ToAddress:   "me@example.jp",
		Subject:     "【議事録】次期リリースv2.4 スプリントレビューおよびAPI仕様書確認",
		BodyText:    "開発チーム各位\nお疲れ様です、鈴木です。\n本日の定例ミーティングの議事録とAPI仕様書の改訂版を共有します。\n・バックエンドの認証エンドポイント改修は水曜までにプルリクエスト提出\n・ステージング環境での負荷テストおよびデプロイ手順の確認を金曜日15時に実施します。\n課題管理チケットのステータス更新をお願いします。",
	},
	{
		Category:    "work",
		FromName:    "GitHub Actions CI",
		FromAddress: "notifications@github.com",
		ToAddress:   "me@example.jp",
		Subject:     "[corp-tech/core-platform] PR #418: Fix SQLite WAL concurrency & unit tests",
		BodyText:    "All checks have passed on pull request #418.\nCommit: 8f92a1c (Fix database connection pool deadlock under high load)\nReviewer @suzuki-pm approved these changes.\nReady to merge into main branch after staging deployment verification.",
	},
	{
		Category:    "work",
		FromName:    "情報システム部 運用担当",
		FromAddress: "it-admin@corp-tech.co.jp",
		ToAddress:   "me@example.jp",
		Subject:     "【社内通知】本番サーバー定期メンテナンスおよびVPN証明書更新のお願い",
		BodyText:    "社員各位\n情報システム部です。来週火曜日22:00〜24:00の間、社内GitサーバーおよびステージングDBクラスターのセキュリティパッチ適用メンテナンスを実施します。\nまた、開発用VPNクライアント証明書の有効期限が近づいている方はポータルから更新手続きを行ってください。",
	},
	{
		Category:    "work",
		FromName:    "田中 裕子 (パートナー営業部)",
		FromAddress: "yuko.tanaka@vip-partner.co.jp",
		ToAddress:   "me@example.jp",
		Subject:     "来期システム導入案件の要件定義書ドラフトと見積もりスケジュール",
		BodyText:    "いつも大変お世話になっております。パートナー営業部の田中です。\n先日のヒアリングに基づき、要件定義書ドラフトおよびフェーズ1開発スケジュールの資料を作成いたしました。次回の定例会議にて技術要件のレビューをお願いいたします。",
	},

	// NEWSLETTER (メルマガ・ニュース・EC通知)
	{
		Category:    "newsletter",
		FromName:    "TechWeekly 編集部",
		FromAddress: "magazine@techweekly-news.jp",
		ToAddress:   "me@example.jp",
		Subject:     "【TechWeekly 第284号】Go言語とWebAssemblyの最新動向・クラウド設計特集",
		BodyText:    "TechWeekly メールマガジンをご購読いただきありがとうございます。\n今週の注目トピック：\n1. Go 1.26 コンパイラ最適化とメモリ効率の改善\n2. ローカルファーストなデスクトップアプリ開発アーキテクチャ特集\n3. 注目のオープンソースプロジェクト10選\n※配信停止（登録解除）をご希望の方は巻末のマイページリンクよりお手続きください。",
	},
	{
		Category:    "newsletter",
		FromName:    "マルシェ・オンラインストア",
		FromAddress: "newsletter@marche-shop.jp",
		ToAddress:   "me@example.jp",
		Subject:     "【週末限定クーポン】秋冬の新作ガジェット＆コーヒー豆ポイント5倍キャンペーン",
		BodyText:    "いつもマルシェ・オンラインストアをご利用いただきありがとうございます。\n今週のおすすめ新着アイテムと会員限定10%OFFクーポンのご案内です！\n特集ページからエントリーするとポイント還元率が5倍になります。\nメールマガジンの配信停止・設定変更はマイアカウント設定画面から行えます。",
	},
	{
		Category:    "newsletter",
		FromName:    "DevDigest Weekly",
		FromAddress: "digest@devdigest.io",
		ToAddress:   "me@example.jp",
		Subject:     "This Week in Software Engineering: Distributed SQLite, Rust & AI Tooling",
		BodyText:    "Welcome to issue #142 of DevDigest Weekly newsletter!\nIn this edition we cover practical Bayesian text classification, modern email protocols, and zero-dependency desktop builds.\nClick here to read the full issue online or manage your newsletter subscription preferences. Unsubscribe anytime.",
	},

	// FINANCE (金融・請求・明細・領収書)
	{
		Category:    "finance",
		FromName:    "メトロポリタン銀行 ネットバンキング",
		FromAddress: "statement@metro-bank.co.jp",
		ToAddress:   "me@example.jp",
		Subject:     "【口座振替のお知らせ】公共料金・クレジットカード引き落とし結果のご確認",
		BodyText:    "メトロポリタン銀行をご利用いただきありがとうございます。\nお客様ご指定の普通預金口座より、本日分の口座振替（引き落とし）が完了いたしましたのでご案内いたします。\n・振替日：10月21日\n・お支払い金額：48,520円\n・振替後残高および入出金明細は公式アプリまたはインターネットバンキングよりご確認ください。",
	},
	{
		Category:    "finance",
		FromName:    "さくらカード 明細デスク",
		FromAddress: "billing@sakura-card.co.jp",
		ToAddress:   "me@example.jp",
		Subject:     "【確定】11月お支払い分のクレジットカードご利用代金明細書のご案内",
		BodyText:    "さくらカード会員様\n11月10日お支払い分のカードご利用代金明細が確定いたしました。\n今回のお支払い総額：72,380円（1回払い）\nご利用明細の内訳および電子領収書のダウンロードは会員専用WEBサービスよりご確認いただけます。",
	},
	{
		Category:    "finance",
		FromName:    "CloudHost Billing",
		FromAddress: "invoices@cloudhost-services.com",
		ToAddress:   "me@example.jp",
		Subject:     "Invoice #INV-2026-1089: Monthly Cloud Server & Storage Statement ($42.00)",
		BodyText:    "Thank you for using CloudHost Services.\nYour monthly invoice #INV-2026-1089 for October 2026 has been generated.\nTotal Amount Due: $42.00 USD\nPayment Method: Visa ending in 4242\nYour receipt and itemized tax invoice are available in the billing portal.",
	},

	// SPAM (迷惑メール・フィッシング・詐欺広告)
	{
		Category:    "spam",
		FromName:    "【至急】当選通知センター",
		FromAddress: "lucky-winner999@promo-free-gift.xyz",
		ToAddress:   "me@example.jp",
		Subject:     "【完全無料】現金1,000万円の特別当選枠にあなたのアドレスが選ばれました！",
		BodyText:    "おめでとうございます！！特別抽選の結果、貴方様が【現金1,000万円】無条件受け取り権利に当選されました！\n本日23時59分までに下記の限定URLをクリックして受取口座を登録してください。手数料は完全無料、審査不要で即日振込いたします！\nhttp://claim-bonus-now.xyz/win",
		BodyHTML:    `<html><body><h1>【完全無料】現金1,000万円当選！</h1><p>今すぐ<a href="http://claim-bonus-now.xyz/win">こちらの限定リンク</a>から口座登録！</p><form action="http://claim-bonus-now.xyz/submit"></form></body></html>`,
	},
	{
		Category:    "spam",
		FromName:    "セキュリティ緊急確認窓口",
		FromAddress: "alert-verify@unusual-login-check.top",
		ToAddress:   "me@example.jp",
		Subject:     "【至急・重要】アカウントが停止されました！24時間以内に本人確認を行ってください",
		BodyText:    "お客様のアカウントに不正なアクセスが検知されたため、安全のため利用を制限・停止いたしました。\nこのまま24時間以内に下記のURLからパスワードと暗証番号を入力して解除手続きを行わない場合、口座およびアカウントが永久に凍結されます。\nhttp://verify-account-freeze.top/login",
		BodyHTML:    `<html><body><p><b>【至急】アカウント凍結の警告</b></p><p><a href="http://verify-account-freeze.top/login">本人確認フォームはこちら</a></p></body></html>`,
	},
	{
		Category:    "spam",
		FromName:    "VIP Crypto Profit Bot",
		FromAddress: "signals@mega-crypto-pump.biz",
		ToAddress:   "me@example.jp",
		Subject:     "URGENT: 5000% Guaranteed Bitcoin Multiplier - Claim Your Free Bonus Token Now!",
		BodyText:    "Act fast! Our exclusive AI trading bot has generated 5000% profit overnight with zero risk!\nDeposit just $100 today and receive $10,000 bonus payout instantly to your wallet. Click http://mega-crypto-pump.biz/claim before this secret offer expires!",
		BodyHTML:    `<html><body><h2>5000% Guaranteed Profit!</h2><a href="http://mega-crypto-pump.biz/claim">Claim Free Bonus</a><img src="http://mega-crypto-pump.biz/track.gif"/></body></html>`,
	},
}

var incomingPool = []SampleTemplate{
	{
		Category:    "spam",
		FromName:    "特別報酬サポート窓口",
		FromAddress: "bonus-transfer@vip-cash-reward.xyz",
		ToAddress:   "me@example.jp",
		Subject:     "【重要・未受領】支援金500万円の振込手続きが未完了です！至急ご確認ください",
		BodyText:    "貴方様宛ての特別支援金500万円の送金準備が完了しております。\n本日中に下記URLへアクセスし、受取用の銀行口座と暗証番号を登録してください。完全無料で即日お受け取りいただけます。\nhttp://vip-cash-reward.xyz/receive",
		BodyHTML:    `<html><body><p>【完全無料】支援金500万円を即日振込！<a href="http://vip-cash-reward.xyz/receive">受取手続きはこちら</a></p></body></html>`,
	},
	{
		Category:    "work",
		FromName:    "山田 太郎 (インフラチーム)",
		FromAddress: "taro.yamada@corp-tech.co.jp",
		ToAddress:   "me@example.jp",
		Subject:     "ステージング環境デプロイ完了とAPI負荷テスト結果の共有",
		BodyText:    "開発チーム各位\n山田です。先ほどプルリクエスト #421 のステージング環境へのデプロイが完了しました。\nAPIエンドポイントのレスポンスタイム計測および負荷テストの結果、エラーレート0%で問題ありませんでした。本番リリースの承認をお願いします。",
	},
	{
		Category:    "finance",
		FromName:    "メトロポリタン銀行 通知サービス",
		FromAddress: "notice@metro-bank.co.jp",
		ToAddress:   "me@example.jp",
		Subject:     "【振込入金のお知らせ】口座へのご入金が確認されました",
		BodyText:    "メトロポリタン銀行をご利用いただきありがとうございます。\nお客様の普通預金口座へお振込による入金がありましたのでご連絡いたします。\n・お取引日：本日\n・お預り金額：125,000円\n詳細な入出金明細および残高はインターネットバンキングでご確認ください。",
	},
	{
		Category:    "newsletter",
		FromName:    "ガジェット通信 ストア",
		FromAddress: "news@marche-shop.jp",
		ToAddress:   "me@example.jp",
		Subject:     "【メルマガ会員限定】メカニカルキーボード特集＆週末15%割引クーポン配布中",
		BodyText:    "いつもメールマガジンをご購読いただきありがとうございます。\n今週の新作ガジェット特集では静電容量無接点方式キーボードや高解像度モニターをご紹介！\n会員限定クーポンコードを利用するとポイント5倍＋15%OFFになります。配信停止・登録解除はマイページから。",
	},
	{
		Category:    "inbox",
		FromName:    "渡辺 健一",
		FromAddress: "kenichi.watanabe@friends.jp",
		ToAddress:   "me@example.jp",
		Subject:     "今週末の温泉旅行の集合時間と新幹線のチケットについて",
		BodyText:    "お疲れ！来週末の箱根旅行だけど、東京駅の改札前に朝8時45分集合でどうかな？\n旅館の夕食コースも予約確認済みです。当日楽しみにしています！",
	},
	{
		Category:    "work",
		FromName:    "田中 裕子 (パートナー営業部)",
		FromAddress: "yuko.tanaka@vip-partner.co.jp",
		ToAddress:   "me@example.jp",
		Subject:     "【VIPパートナー案件】次期フェーズ契約書ドラフトの確認依頼",
		BodyText:    "いつもお世話になっております。パートナー営業部の田中です。\n来期システムの契約書ドラフトおよび要件定義の最終版をお送りいたします。ご確認のほどよろしくお願いいたします。",
	},
}

// SeedInitialData populates initial training corpus, sample magnets, and initial inbox emails if the DB is empty.
func SeedInitialData(store *db.Store, engine *classifier.Engine) error {
	var emailCount int
	if err := store.DB.QueryRow("SELECT COUNT(*) FROM emails").Scan(&emailCount); err != nil {
		return err
	}
	if emailCount > 0 {
		return nil
	}

	buckets, err := store.GetBuckets()
	if err != nil {
		return err
	}
	bucketByName := make(map[string]db.Bucket)
	for _, b := range buckets {
		bucketByName[b.Name] = b
	}

	// 1. Create default POPFile Magnets
	if workB, ok := bucketByName["work"]; ok {
		_, _ = store.CreateMagnet(workB.ID, "domain", "vip-partner.co.jp", "VIPパートナー企業ドメインからのメールを常に「業務・開発」へ振り分け")
	}
	if finB, ok := bucketByName["finance"]; ok {
		_, _ = store.CreateMagnet(finB.ID, "from", "statement@metro-bank.co.jp", "メトロポリタン銀行の明細通知を常に「金融・請求」へ振り分け")
	}

	// 2. Train initial corpus templates & insert them into received folder
	baseTime := time.Now().Add(-48 * time.Hour)
	for _, tpl := range initialCorpusTemplates {
		b, ok := bucketByName[tpl.Category]
		if !ok {
			continue
		}
		if err := engine.TrainEmail(b.ID, tpl.FromName, tpl.FromAddress, tpl.ToAddress, tpl.Subject, tpl.BodyText, tpl.BodyHTML, 1); err != nil {
			return err
		}
	}

	// 3. Now classify & insert the initial emails so their POPFile probability distributions are realistic!
	for i, tpl := range initialCorpusTemplates {
		msgTime := baseTime.Add(time.Duration(i*2) * time.Hour).Format("2006-01-02 15:04:05")
		res, err := engine.ClassifyAndExplain(tpl.FromName, tpl.FromAddress, tpl.ToAddress, tpl.Subject, tpl.BodyText, tpl.BodyHTML, true)
		if err != nil {
			continue
		}
		magID := int64(0)
		action := "classified"
		if res.MagnetMatched != nil {
			magID = res.MagnetMatched.ID
			action = "magnet"
		}
		ccLine := ""
		if tpl.Category == "work" {
			ccLine = "Cc: dev-team@corp-tech.co.jp, pm-office@corp-tech.co.jp\n"
		} else if tpl.Category == "inbox" && i%2 == 0 {
			ccLine = "Cc: travel-group@friends.jp\n"
		}
		rawH := fmt.Sprintf(
			"From: \"%s\" <%s>\nTo: %s\n%sSubject: %s\nDate: %s\nMessage-ID: <seed-%d@tunbridge>\nReply-To: %s\nReturn-Path: <%s>\nMIME-Version: 1.0\nContent-Type: text/plain; charset=UTF-8\nUser-Agent: Mozilla/5.0 (Tunbridge-Sample/1.0)\nX-Text-Classification: %s\nX-POPFile-Confidence: %.4f",
			tpl.FromName, tpl.FromAddress, tpl.ToAddress, ccLine, tpl.Subject, msgTime, i+1, tpl.FromAddress, tpl.FromAddress, res.BucketName, res.Confidence,
		)

		em := &db.Email{
			UIDL:             fmt.Sprintf("seed-%d", i+1),
			MessageID:        fmt.Sprintf("seed-%d@tunbridge", i+1),
			FromName:         tpl.FromName,
			FromAddress:      tpl.FromAddress,
			ToAddress:        tpl.ToAddress,
			Subject:          tpl.Subject,
			BodyText:         tpl.BodyText,
			BodyHTML:         tpl.BodyHTML,
			RawHeaders:       rawH,
			Date:             msgTime,
			BucketID:         res.BucketID,
			OriginalBucketID: res.BucketID,
			MagnetID:         magID,
			Confidence:       res.Confidence,
			Scores:           res.Probabilities,
			IsRead:           i < len(initialCorpusTemplates)-4,
			IsTrained:        true,
			Folder:           "received",
		}
		id, err := store.InsertEmail(em)
		if err == nil {
			store.AddHistoryEvent(id, em.Subject, em.FromAddress, action, "", res.BucketLabel)
		}
	}

	return nil
}

// GenerateIncomingSamples generates N newly arrived emails (untrained) and classifies them using POPFile.
func GenerateIncomingSamples(count int, category string, store *db.Store, engine *classifier.Engine) ([]db.Email, error) {
	if count <= 0 {
		count = 1
	}
	if count > 20 {
		count = 20
	}

	var pool []SampleTemplate
	for _, t := range incomingPool {
		if category == "" || category == "random" || t.Category == category {
			pool = append(pool, t)
		}
	}
	if len(pool) == 0 {
		pool = incomingPool
	}

	var created []db.Email
	for i := 0; i < count; i++ {
		tpl := pool[rand.Intn(len(pool))]
		now := time.Now().Format("2006-01-02 15:04:05")
		uid := fmt.Sprintf("sim-%d-%d", time.Now().UnixNano(), i)

		res, err := engine.ClassifyAndExplain(tpl.FromName, tpl.FromAddress, tpl.ToAddress, tpl.Subject, tpl.BodyText, tpl.BodyHTML, true)
		if err != nil {
			return nil, err
		}

		magID := int64(0)
		action := "classified"
		if res.MagnetMatched != nil {
			magID = res.MagnetMatched.ID
			action = "magnet"
		}

		ccLine := ""
		if tpl.Category == "work" {
			ccLine = "Cc: team-all@corp-tech.co.jp\n"
		}
		rawH := fmt.Sprintf(
			"From: \"%s\" <%s>\nTo: %s\n%sSubject: %s\nDate: %s\nMessage-ID: <%s@tunbridge>\nReply-To: %s\nReturn-Path: <%s>\nMIME-Version: 1.0\nContent-Type: text/plain; charset=UTF-8\nUser-Agent: Tunbridge-Simulator/1.0\nX-Text-Classification: %s\nX-POPFile-Confidence: %.4f",
			tpl.FromName, tpl.FromAddress, tpl.ToAddress, ccLine, tpl.Subject, now, uid, tpl.FromAddress, tpl.FromAddress, res.BucketName, res.Confidence,
		)

		em := &db.Email{
			UIDL:             uid,
			MessageID:        uid + "@tunbridge",
			FromName:         tpl.FromName,
			FromAddress:      tpl.FromAddress,
			ToAddress:        tpl.ToAddress,
			Subject:          tpl.Subject,
			BodyText:         tpl.BodyText,
			BodyHTML:         tpl.BodyHTML,
			RawHeaders:       rawH,
			Date:             now,
			BucketID:         res.BucketID,
			OriginalBucketID: res.BucketID,
			MagnetID:         magID,
			Confidence:       res.Confidence,
			Scores:           res.Probabilities,
			IsRead:           false,
			IsTrained:        false,
			Folder:           "received",
		}
		id, err := store.InsertEmail(em)
		if err != nil {
			return nil, err
		}
		store.AddHistoryEvent(id, em.Subject, em.FromAddress, action, "", res.BucketLabel)
		if full, err := store.GetEmail(id); err == nil {
			created = append(created, *full)
		}
	}
	return created, nil
}
