package mail_test

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"

	"tunbridge/internal/classifier"
	"tunbridge/internal/db"
	"tunbridge/internal/mail"
)

func mustEncode(t *testing.T, enc transform.Transformer, s string) []byte {
	t.Helper()
	b, _, err := transform.Bytes(enc, []byte(s))
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}
	return b
}

func TestDecodeMIMEHeaderMultilingual(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "ISO-2022-JP Q-encoded multi-word FromName",
			in:   "=?ISO-2022-JP?Q?=1B=24B=258=25c=259=25H=25=5E=25=24=1B=28B?= =?ISO-2022-JP?Q?=1B=24B=257=25g=25C=25W=1B=28B_=28?= =?ISO-2022-JP?Q?Just_MyShop=29?=",
			want: "ジャストマイショップ (Just MyShop)",
		},
		{
			name: "ISO-2022-JP B-encoded Subject",
			in:   "=?ISO-2022-JP?B?GyRCMmE3YyRmJCghIkJnQDU0fCRLSC82WCRIJEokQyQ/Qys6aj1hGyhC?= =?ISO-2022-JP?B?GyRCMGxPOiROOm5JSiRyPH1PPyEjIVhDKzpqPWEwbE86GyhCIBskQkgvGyhC?= =?ISO-2022-JP?B?GyRCNlg6bklKPTghWSQsNCk5VDdoRGohKhsoQg==?=",
			want: "過激ゆえ、大正期に発禁となった谷崎潤一郎の作品を収録。『谷崎潤一郎 発禁作品集』が刊行決定！",
		},
		{
			name: "GB2312 B-encoded FromName",
			in:   "=?gb2312?B?pbmpYKXRqWCls6XUqWAgx96+39PDxrc=?=",
			want: "スーパーコピー 寝具用品",
		},
		{
			name: "Shift_JIS B-encoded Subject",
			in:   "=?shift_jis?B?UmU6IJP6lnuNkZPglK2RlyCDdYOJg5ODaINYgVuDcIFbg1I=?= =?shift_jis?B?g3OBWyCOc4/qjcWS4Ym/imk=?=",
			want: "Re: 日本国内発送 ブランドスーパーコピー 市場最低価格",
		},
		{
			name: "Malformed UTF-8 B-encoded with 1-char alignment error",
			in:   "=?UTF-8?B?44CQ6Ieq5YuVZXLmlrDnrIzkuro344CR44K744Kt44Ol44Oq44OG44Oja44OR44OD44Kv44C444O044O844K544Kv44OI77yI44Oa44O844K444Ol44O844K544K344On44Oz77yJ44GU6KuL44Gf44Gu44GK55ml44KJ44Gb77yJ5rOo5paH44OQ44Oz44K044O844O677yAT1JELTIwMjYxMDIyLTQ3MTld?=",
			want: "【自動er新笌人7】セキュリテャパック〸ヴースクト（ページュースション）ご請たのお癥らせ）注文バンゴーヺ＀ORD-20261022-4719]",
		},
		{
			name: "RLO and zero-width spoofed subject",
			in:   "\u202eせら知\u2060おるす関に税\u200d得\u200d所\u202c",
			want: "所得税に関するお知らせ",
		},
		{
			name: "RLO and zero-width spoofed sender",
			in:   "\u202e【 局務\u200c事 x\u2060a\u2060T\ufeff-e\u200d 】\u202c",
			want: "【 e-Tax 事務局 】",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mail.DecodeMIMEHeader(tc.in)
			if got != tc.want {
				t.Errorf("DecodeMIMEHeader() =\n  %q\nwant:\n  %q", got, tc.want)
			}
		})
	}
}

func TestAutoDetectUnencodedCharsets(t *testing.T) {
	jpText := "いつもご利用いただきありがとうございます。口座振替のお知らせとご利用代金明細のご確認をお願いいたします。"
	zhSimpText := "您好，感谢您使用我们的服务。您的订单已经发货，请登录个人中心查看物流信息和电子发票。"
	zhTradText := "親愛的客戶您好，感謝您使用本行網路銀行服務，請確認您的帳單明細與轉帳紀錄。"
	krText := "안녕하세요. 고객님의 결제 내역과 계좌 이체 결과를 안내해 드립니다. 확인 부탁드립니다."
	ruText := "Здравствуйте! Благодарим вас за использование нашего сервиса. Ваш заказ успешно отправлен."

	cases := []struct {
		name string
		raw  []byte
		want string
	}{
		{
			name: "Unencoded Shift_JIS",
			raw:  mustEncode(t, japanese.ShiftJIS.NewEncoder(), jpText),
			want: jpText,
		},
		{
			name: "Unencoded EUC-JP",
			raw:  mustEncode(t, japanese.EUCJP.NewEncoder(), jpText),
			want: jpText,
		},
		{
			name: "Unencoded ISO-2022-JP",
			raw:  mustEncode(t, japanese.ISO2022JP.NewEncoder(), jpText),
			want: jpText,
		},
		{
			name: "Unencoded GB18030 / GBK (Simplified Chinese)",
			raw:  mustEncode(t, simplifiedchinese.GB18030.NewEncoder(), zhSimpText),
			want: zhSimpText,
		},
		{
			name: "Unencoded Big5 (Traditional Chinese)",
			raw:  mustEncode(t, traditionalchinese.Big5.NewEncoder(), zhTradText),
			want: zhTradText,
		},
		{
			name: "Unencoded EUC-KR (Korean)",
			raw:  mustEncode(t, korean.EUCKR.NewEncoder(), krText),
			want: krText,
		},
		{
			name: "Unencoded Windows-1251 (Russian Cyrillic)",
			raw:  mustEncode(t, charmap.Windows1251.NewEncoder(), ruText),
			want: ruText,
		},
		{
			name: "Unencoded KOI8-R (Russian Cyrillic)",
			raw:  mustEncode(t, charmap.KOI8R.NewEncoder(), ruText),
			want: ruText,
		},
		{
			name: "Unencoded UTF-16LE with BOM",
			raw:  mustEncode(t, unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewEncoder(), jpText),
			want: jpText,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mail.DetectAndDecodeBytes(tc.raw, "")
			if got != tc.want {
				t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
			}
			if !utf8.ValidString(got) {
				t.Errorf("%s: output is not valid UTF-8", tc.name)
			}
		})
	}
}

func TestParseRawEmailWithUnencodedCharsetAndMultipart(t *testing.T) {
	sjisBody := mustEncode(t, japanese.ShiftJIS.NewEncoder(), "<html><head><meta http-equiv=\"Content-Type\" content=\"text/html; charset=shift_jis\"><style>body { color: #000; }</style></head><body><p>日本国内発送 ブランドスーパーコピー 市場最低価格</p></body></html>")
	var raw bytes.Buffer
	raw.WriteString("From: =?gb2312?B?pbmpYKXRqWCls6XUqWAgx96+39PDxrc=?= <xlp@trylb.com>\r\n")
	raw.WriteString("To: <user@example.jp>\r\n")
	raw.WriteString("Subject: =?shift_jis?B?UmU6IJP6lnuNkZPglK2RlyCDdYOJg5ODaINYgVuDcIFbg1I=?= =?shift_jis?B?g3OBWyCOc4/qjcWS4Ym/imk=?=\r\n")
	raw.WriteString("MIME-Version: 1.0\r\n")
	raw.WriteString("Content-Type: text/html\r\n\r\n")
	raw.Write(sjisBody)

	parsed, err := mail.ParseRawEmail(raw.Bytes())
	if err != nil {
		t.Fatalf("ParseRawEmail failed: %v", err)
	}
	if parsed.FromName != "スーパーコピー 寝具用品" {
		t.Errorf("FromName = %q, want %q", parsed.FromName, "スーパーコピー 寝具用品")
	}
	if parsed.Subject != "Re: 日本国内発送 ブランドスーパーコピー 市場最低価格" {
		t.Errorf("Subject = %q, want %q", parsed.Subject, "Re: 日本国内発送 ブランドスーパーコピー 市場最低価格")
	}
	if !strings.Contains(parsed.BodyText, "日本国内発送") || strings.Contains(parsed.BodyText, "color: #000") {
		t.Errorf("BodyText = %q, expected clean Japanese text without CSS", parsed.BodyText)
	}
}

func TestRepairStoredEmailsOnTempDB(t *testing.T) {
	store, err := db.NewStore(t.TempDir() + "/test_repair.db")
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.DB.Close()

	engine := classifier.NewEngine(store)
	if err := mail.SeedInitialData(store, engine); err != nil {
		t.Fatalf("SeedInitialData failed: %v", err)
	}

	repaired, err := mail.RepairStoredEmails(store, engine)
	if err != nil {
		t.Fatalf("RepairStoredEmails failed: %v", err)
	}
	_ = repaired

	emails, err := store.ListEmails(db.EmailFilter{Folder: "all", Limit: 50})
	if err != nil {
		t.Fatalf("ListEmails failed: %v", err)
	}
	for _, item := range emails {
		em, err := store.GetEmail(item.ID)
		if err != nil {
			t.Fatalf("GetEmail(%d) failed: %v", item.ID, err)
		}
		if !utf8.ValidString(em.FromName) || !utf8.ValidString(em.Subject) || !utf8.ValidString(em.BodyText) {
			t.Errorf("ID=%d contains invalid UTF-8", em.ID)
		}
	}
}
