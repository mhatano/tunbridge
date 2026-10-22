# Tunbridge (ベイジアン自動分別機能付きメールクライアント)

**Tunbridge** は、マルチバケット・ナイーブベイズ分類エンジン、TOE（Train On Error）再学習、マグネット（無条件振り分けルール）、単語確率マトリクス＆色分け表示、コーパス管理機能を備えたクロスプラットフォーム対応メールクライアント（MUA）です。

Pure Go 製 SQLite（`modernc.org/sqlite`）と埋め込みアセット（`embed.FS`）を採用しており、**Cコンパイラ（CGO）や外部ランタイム不要の単一バイナリ**として Windows・Linux・macOS でビルド・実行できます。また、文字列リソース（`internal/ui/i18n.js`）を分離しており、OS・ブラウザ環境に応じた**日本語／英語（JA / EN）の自動判定および設定画面からの言語切り替え**に対応しています。

## 主な機能

1. **マルチバケット・ナイーブベイズ自動分類 (`internal/classifier`)**
   - 標準バケット（`inbox`, `work`, `newsletter`, `finance`, `spam`, `unclassified`）に加え、任意のカスタムバケットを追加・編集可能。
   - POPFile互換のヘッダ疑似トークン（`from:`, `domain:`, `to:`, `subject:`, `url:`, `html:`）と日本語（漢字複合語・N-gram・カタカナ語）・英語トークナイザを搭載。
   - メールヘッダへ `X-Text-Classification: <bucket>` および `X-POPFile-Confidence` を自動付与。
2. **TOE (Train On Error) ワンクリック再学習**
   - メール閲覧画面上部のバケットボタンをクリックするだけで即座に再分類し、旧バケットからの単語減算・新バケットへの単語加算と精度統計（FP/FN）の更新を実行。
3. **単語スコア・確率の可視化 & コーパス管理**
   - **単語色分けビュー**: 件名と本文中の認識トークンを、最も強く示唆するバケットの色でハイライト表示（ホバーで確率と出現数を表示）。
   - **分類根拠・単語確率マトリクス**: バケット別事後確率 $P(\text{Bucket} \mid \text{Message})$ のグラフと、各トークンのバケット別寄与度一覧表。
   - **コーパス辞書 & Word Lookup**: バケット別の学習語彙一覧、特定の単語の全バケット間比較、リアルタイム分類テスト用サンドボックス。
4. **マグネット機能 (Magnets)**
   - 特定の差出人（`From`）、送信元ドメイン（`Domain`）、宛先（`To`）、件名（`Subject`）に一致するメールを、ベイズ判定に優先して特定バケットへ無条件振り分け。
5. **MUA（メール送受信・添付ファイル・プロバイダ別管理）機能**
   - **POP3 / IMAP4 受信 & SMTP 送信（添付ファイル送受信対応）**: 実際のメールサーバー（SSL/TLS対応）からの受信・自動分類とSMTP送信に対応。多言語エンコーディングの自動判定・デコード、下部中アイコン一覧＆右側分割プレビュー（別ウィンドウ表示対応）の添付ファイルビューアを搭載。
   - **プロバイダ別分離表示 & キーボード高速ナビゲーション**: プロバイダごとのグループ表示・フィルタ、カーソルキー / `Home` / `End` / `PageUp` / `PageDown` 移動、`Delete` 削除＆ `Ctrl+Z` 安全 Undo 機能を搭載。

---

## ビルド方法

Go 1.22 以上（推奨: Go 1.26+）がインストールされていれば、Cコンパイラ（GCC等）なしでどのOSからでもビルドできます。

### 1. Windows でのビルド

```powershell
git clone https://github.com/mhatano/tunbridge.git
cd tunbridge
go build -o tunbridge.exe .
```

### 2. Linux / macOS でのビルド

```bash
git clone https://github.com/mhatano/tunbridge.git
cd tunbridge
CGO_ENABLED=0 go build -o tunbridge .
```

### 3. クロスコンパイル（例: Windows から Linux 向けバイナリを生成）

```powershell
# Linux (x86_64 / amd64) 向けビルド
$env:CGO_ENABLED="0"; $env:GOOS="linux"; $env:GOARCH="amd64"; go build -o tunbridge-linux-amd64 .

# Linux (ARM64 / Raspberry Pi 等) 向けビルド
$env:CGO_ENABLED="0"; $env:GOOS="linux"; $env:GOARCH="arm64"; go build -o tunbridge-linux-arm64 .
```

### 4. 単体テストの実行

```bash
go test ./...
```

---

## 起動方法

### Windows（ネイティブ WebView2 デスクトップウィンドウ）

```powershell
.\tunbridge.exe
```

### Linux / macOS

デスクトップ環境（X11 / Wayland / macOS）で起動すると、Chromium / Chrome のアプリモードウィンドウ（`--app`）または既定のブラウザが自動的に開きます。

```bash
./tunbridge
```

### サーバー / ヘッドレスモード（固定ポート指定）

ブラウザから任意のポートでアクセスしたい場合や、Linux サーバー上で常駐させる場合は `-headless` と `-port` オプションを使用します：

```bash
./tunbridge -headless -port 8080 -db tunbridge.db
```
その後、ブラウザで `http://127.0.0.1:8080` を開いてください。

---

## ライセンス (License)

本プロジェクト（**Tunbridge**）は **[MIT License](LICENSE)** の下で公開されています。

### 使用ライブラリとライセンス一覧
本ソフトウェアが参照しているすべての外部モジュールは、MIT ライセンスと互換性のある非コピーレフト（Permissive / Public Domain）ライセンスで提供されています：

| パッケージ | ライセンス | 用途 |
| :--- | :--- | :--- |
| `modernc.org/sqlite` (および `libc`, `mathutil`, `memory`) | BSD-3-Clause (SQLite 本体は Public Domain) | Pure Go SQLite データベースドライバ |
| `github.com/jchv/go-webview2` | MIT License | Windows WebView2 デスクトップウィンドウ表示 |
| `github.com/jchv/go-winloader` | ISC License | Windows メモリモジュールローダ |
| `golang.org/x/text`, `golang.org/x/sys` | BSD-3-Clause | 文字エンコーディング変換・OSシステムコール |
| `github.com/dustin/go-humanize` | MIT License | 数値・サイズフォーマット補助 |
| `github.com/google/uuid` | BSD-3-Clause | UUID 生成 |
| `github.com/mattn/go-isatty` | MIT License | 端末判定補助 |
| `github.com/ncruces/go-strftime` | MIT License | 日時フォーマット補助 |
| `github.com/remyoudompheng/bigfft` | BSD-3-Clause | 多倍長演算補助 |

