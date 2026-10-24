// ============================================================================
// Tunbridge Internationalization (i18n) String Resources (JA / EN)
// ============================================================================

const I18N_RESOURCES = {
  ja: {
    app_title: 'Tunbridge - ベイジアン自動分別メールクライアント',
    stat_accuracy: '分類精度:',
    stat_total_emails: '総メール数:',
    stat_vocab: '学習コーパス語彙:',
    stat_vocab_unit: '語',
    stat_magnet_hits: 'マグネット発動:',
    stat_magnet_unit: '回',
    folder_received: 'すべての受信箱',
    folder_unread: '未読メール',
    folder_starred: 'スター付きメール',
    folder_sent: '送信済みメール',
    folder_trash: 'ゴミ箱',
    folder_default: 'メール一覧',
    bucket_prefix: 'バケット: ',
    provider_all: 'すべて',
    provider_demo: 'デモ・サンプル (local)',
    provider_demo_short: 'デモ・サンプル',
    provider_imported_prefix: 'インポート',
    provider_imported_short: 'EMLインポート',
    provider_other: 'その他',
    empty_reader_select: '左のリストからメールを選択すると、本文および POPFile 単語確率・分類根拠が表示されます',
    empty_reader_none: '表示対象のメールがありません',
    empty_list_none: '該当するメールはありません',
    count_suffix: '件',
    unread_prefix: '・未読 ',
    undo_btn_label: '↩ 削除を元に戻す',
    undo_toast_action: '↩ 元に戻す (Undo)',
    undo_restored_toast: '削除を取り消して元の場所へ復元しました',
    undo_failed_toast: '復元に失敗しました: ',
    trained_toe_badge: '↻TOE学習',
    trained_toe_title: 'TOE再分類学習済み',
    trained_corpus_badge: '✓学習済',
    trained_corpus_title: 'コーパス学習済み',
    magnet_applied_prefix: 'マグネット適用: ',
    attachment_count_title: '添付ファイル {count} 件',
    header_config_btn: '⚙️ 表示ヘッダ設定 ({count})',
    header_config_btn_title: '常時表示するヘッダ項目を選択・追加',
    font_config_btn: '🌐 言語・フォント設定',
    font_config_btn_title: '表示言語・フォント種類・サイズを設定',
    other_headers_hide: '🔼 その他のヘッダを隠す',
    other_headers_show: '📋 その他のヘッダを表示 ({count}件)',
    btn_reply_sender: '↩️ 差出人に返信',
    btn_reply_sender_title: '差出人のみへ返信',
    btn_reply_all: '↩️ 全員に返信',
    btn_reply_all_title: '差出人とTo/Ccの全員へ返信',
    btn_forward: '➡️ 転送',
    btn_forward_title: 'メール本文と添付ファイルを別の宛先へ転送',
    btn_register_magnet: '🧲 差出人マグネット登録',
    btn_export_eml: '💾 .eml出力',
    btn_delete_perm: '完全削除',
    btn_delete_trash: '🗑️ 削除',
    header_empty: '(なし)',
    header_unset: '(未設定)',
    popfile_judgment: 'POPFile 判定:',
    magnet_matched_badge: '🧲 マグネット一致: {field}="{pattern}"',
    toe_reclassified_note: '(手動で再分類・TOE学習済み)',
    toe_hint_note: '※誤判定時は下のバケットをクリックして再学習 (TOE)',
    bucket_change_label: 'バケット変更・学習:',
    tab_body: '📄 メール本文 (TextArea)',
    tab_colored: '🎨 POPFile 単語色分けビュー',
    tab_matrix: '🔬 分類根拠・単語確率マトリクス ({count}語)',
    tab_headers: '📋 生ソース (RFC822)',
    raw_rfc822_title: 'POPFile 自動挿入ヘッダ (RFC822)',
    colored_view_desc:
      '💡 <strong>POPFile 単語色分けビュー:</strong> コーパス辞書に存在する単語を、最も強く示唆するバケットの色でハイライト表示しています（マウスホバーで確率を表示）。',
    colored_subject_label: '件名 (Subject):',
    colored_token_title: 'バケット傾向: {label} ({pct}%) / メール内出現: {freq}回',
    matrix_prob_title: '📊 バケット別 事後確率 P(Bucket | Message)',
    matrix_match_words: '一致 {count} 語',
    matrix_table_title: '🔬 メール内トークンのバケット別スコア表 (上位寄与語)',
    matrix_col_token: 'トークン (ヘッダ疑似語含む)',
    matrix_col_freq: '出現数',
    matrix_col_best: '最有力バケット',
    matrix_empty: 'コーパス一致語がありません',
    att_bottom_title: '📎 添付ファイル ({count}件)',
    att_bottom_hint: '※アイコンをシングルクリックすると本文の右側に分割表示します',
    att_close_preview: '右側プレビューを閉じる ✕',
    att_card_title_suffix: ' — シングルクリックで右側に分割表示',
    att_open_newwin: '↗ 別ウィンドウで開く',
    att_open_newwin_title: '添付ファイル全体を別ウィンドウで開いて表示',
    att_save: '💾 保存',
    att_save_full: '💾 ファイルとして保存',
    att_save_title: 'PCに保存',
    att_close_title: 'プレビューを閉じる',
    att_type_size: '種類: {type} ／ サイズ: {size}',
    compose_title_new: '✉️ メール作成・送信',
    compose_title_reply: '↩️ 差出人に返信',
    compose_title_reply_all: '↩️ 全員に返信',
    compose_title_forward: '➡️ メールを転送',
    compose_local_demo: 'ローカル送信 (オフラインデモ)',
    compose_local_addr: 'ローカル送信 (me@local.popfile)',
    magnets_active: '有効',
    magnets_inactive: '無効',
    magnets_empty: '登録されているマグネットはありません',
    corpus_showing_count: '表示中: {count} トークン',
    corpus_empty: '該当する単語がありません',
    lookup_result_title: '「{word}」のバケット別分布:',
    lookup_result_row: '出現 <strong>{count}</strong> 回 / バケット親和度: <strong>{pct}%</strong>',
    sandbox_result_prefix: '判定結果:',
    buckets_emails_col: '{total} 通 (未読 {unread})',
    buckets_docs_col: '{docs} 通',
    buckets_words_col: '{total} 語 ({unique} 種)',
    buckets_clear_btn: 'コーパス消去',
    buckets_delete_btn: '削除',
    history_classified: '<span style="color:#38bdf8;">🤖 ベイズ自動分類</span>',
    history_magnet: '<span style="color:#fcd34d;">🧲 マグネット確定</span>',
    history_reclassified: '<span style="color:#10b981;font-weight:700;">↻ TOE再分類・学習</span>',
    history_empty: '履歴はまだありません',
    accounts_local_save: 'ローカル保存',
    accounts_fetch_btn: '📥 受信',
    accounts_edit_btn: '✏️ 編集',
    accounts_duplicate_btn: '📋 複製',
    accounts_delete_btn: '削除',
    accounts_status_enabled: '🟢 有効',
    accounts_status_disabled: '⚪ 無効',
    accounts_editing_badge: '✏️ 編集中 (ID: #{id} / 保存すると上書き)',
    accounts_btn_save_new: '💾 アカウント保存',
    accounts_btn_save_overwrite: '💾 変更を上書き保存',
    accounts_copy_suffix: ' (コピー)',
    accounts_toast_loaded: '「{name}」を編集フォームに呼び出しました',
    accounts_toast_updated: 'アカウント「{name}」を上書き保存しました',
    accounts_toast_created: '新規アカウント「{name}」を保存しました',
    accounts_toast_duplicated: '「{name}」を複製しました（編集フォームでそのまま調整できます）',
    accounts_empty: '外部POP3/IMAPアカウントは未設定です（デモ生成または.emlインポートですぐに全機能を試せます）',
    accounts_interval_manual: '手動のみ',
    accounts_interval_mins: '{min}分ごと',
    accounts_exported_toast: 'アカウント設定をエクスポートしました',
    accounts_imported_toast: '{count} 件のアカウント設定をインポート・復元しました',
    accounts_import_err_invalid: '無効なアカウント設定ファイルです（JSON形式を確認してください）',
    netlog_btn_view: '🔍 通信ログを見る',
    netlog_socket_connected: '🟢 TCPソケット接続: 成功 ({local} ➔ {remote})',
    netlog_socket_disconnected: '🔴 TCPソケット接続: 失敗（ソケット未接続）',
    netlog_tls_ok: '🔒 SSL/TLS暗号化: 確立済 ({info})',
    netlog_tls_fail: '❌ SSL/TLS暗号化: ハンドシェイク失敗',
    netlog_tls_plain: '🔓 平文通信 (非TLS)',
    netlog_stage_dns: '🌐 原因分類: サーバーアドレス（ホスト名）解決エラー',
    netlog_stage_tcp: '🔌 原因分類: ポート番号 / TCPソケット接続エラー',
    netlog_stage_tls: '🔒 原因分類: SSL/TLS 暗号化・ポート設定不一致エラー',
    netlog_stage_greeting: '📡 原因分類: サーバー初期応答（プロトコル不一致）エラー',
    netlog_stage_auth: '🔑 原因分類: アカウント認証（ユーザー名 / パスワード）エラー',
    netlog_stage_command: '⚠️ 原因分類: プロトコルコマンド応答エラー',
    netlog_transcript_title: '📜 ソケット接続 & プロトコル通信トレースログ（パスワードは自動マスク済）',
    netlog_edit_account_btn: '✏️ このアカウント設定を編集する',
    netlog_copied_toast: 'ネットワーク診断ログをクリップボードにコピーしました',
  },
  en: {
    app_title: 'Tunbridge - Bayesian Mail User Agent',
    stat_accuracy: 'Accuracy:',
    stat_total_emails: 'Total Messages:',
    stat_vocab: 'Corpus Vocab:',
    stat_vocab_unit: 'words',
    stat_magnet_hits: 'Magnet Hits:',
    stat_magnet_unit: 'hits',
    folder_received: 'All Inbox',
    folder_unread: 'Unread Messages',
    folder_starred: 'Starred',
    folder_sent: 'Sent Messages',
    folder_trash: 'Trash',
    folder_default: 'Messages',
    bucket_prefix: 'Bucket: ',
    provider_all: 'All',
    provider_demo: 'Demo / Sample (local)',
    provider_demo_short: 'Demo / Sample',
    provider_imported_prefix: 'Imported',
    provider_imported_short: 'EML Import',
    provider_other: 'Other',
    empty_reader_select: 'Select a message from the list on the left to view its body and POPFile Bayesian word probabilities',
    empty_reader_none: 'No messages to display',
    empty_list_none: 'No matching messages found',
    count_suffix: 'msgs',
    unread_prefix: ' · Unread ',
    undo_btn_label: '↩ Undo Delete',
    undo_toast_action: '↩ Undo',
    undo_restored_toast: 'Deletion undone and message restored',
    undo_failed_toast: 'Failed to restore: ',
    trained_toe_badge: '↻TOE Trained',
    trained_toe_title: 'Reclassified & trained via TOE',
    trained_corpus_badge: '✓Trained',
    trained_corpus_title: 'Trained in corpus',
    magnet_applied_prefix: 'Magnet matched: ',
    attachment_count_title: '{count} attachment(s)',
    header_config_btn: '⚙️ Header Fields ({count})',
    header_config_btn_title: 'Choose or add always-visible mail headers',
    font_config_btn: '🌐 Language & Fonts',
    font_config_btn_title: 'Configure display language, fonts, and text sizes',
    other_headers_hide: '🔼 Hide Other Headers',
    other_headers_show: '📋 Show Other Headers ({count})',
    btn_reply_sender: '↩️ Reply',
    btn_reply_sender_title: 'Reply to sender only',
    btn_reply_all: '↩️ Reply All',
    btn_reply_all_title: 'Reply to sender and all To/Cc recipients',
    btn_forward: '➡️ Forward',
    btn_forward_title: 'Forward message and attachments to another recipient',
    btn_register_magnet: '🧲 Add Sender Magnet',
    btn_export_eml: '💾 Export .eml',
    btn_delete_perm: 'Delete Forever',
    btn_delete_trash: '🗑️ Delete',
    header_empty: '(none)',
    header_unset: '(not set)',
    popfile_judgment: 'POPFile Classification:',
    magnet_matched_badge: '🧲 Magnet Match: {field}="{pattern}"',
    toe_reclassified_note: '(Manually reclassified & TOE-trained)',
    toe_hint_note: '* Click a bucket below to reclassify & train on error (TOE)',
    bucket_change_label: 'Reclassify & Train (TOE):',
    tab_body: '📄 Message Body (TextArea)',
    tab_colored: '🎨 POPFile Colored Words',
    tab_matrix: '🔬 Word Score Matrix ({count} words)',
    tab_headers: '📋 Raw Source (RFC822)',
    raw_rfc822_title: 'POPFile Inserted Headers (RFC822)',
    colored_view_desc:
      '💡 <strong>POPFile Colored Word View:</strong> Tokens found in the learned corpus are highlighted in the color of their most probable bucket (hover to inspect probabilities).',
    colored_subject_label: 'Subject:',
    colored_token_title: 'Best bucket: {label} ({pct}%) / Occurrences in message: {freq}',
    matrix_prob_title: '📊 Posterior Probability by Bucket P(Bucket | Message)',
    matrix_match_words: '{count} matched words',
    matrix_table_title: '🔬 Top Contributing Tokens & Bucket Score Matrix',
    matrix_col_token: 'Token (incl. header pseudo-tokens)',
    matrix_col_freq: 'Freq',
    matrix_col_best: 'Strongest Bucket',
    matrix_empty: 'No matching corpus tokens found',
    att_bottom_title: '📎 Attachments ({count})',
    att_bottom_hint: '* Single-click an icon to split & preview on the right side of the body',
    att_close_preview: 'Close Preview ✕',
    att_card_title_suffix: ' — Single-click to preview on the right pane',
    att_open_newwin: '↗ Open in New Window',
    att_open_newwin_title: 'Open full attachment in a separate window',
    att_save: '💾 Save',
    att_save_full: '💾 Save File',
    att_save_title: 'Save to disk',
    att_close_title: 'Close preview pane',
    att_type_size: 'Type: {type} / Size: {size}',
    compose_title_new: '✉️ Compose Message',
    compose_title_reply: '↩️ Reply to Sender',
    compose_title_reply_all: '↩️ Reply to All',
    compose_title_forward: '➡️ Forward Message',
    compose_local_demo: 'Local Outbox (Offline Demo)',
    compose_local_addr: 'Local Outbox (me@local.popfile)',
    magnets_active: 'Enabled',
    magnets_inactive: 'Disabled',
    magnets_empty: 'No magnets registered',
    corpus_showing_count: 'Showing: {count} tokens',
    corpus_empty: 'No matching tokens found',
    lookup_result_title: 'Bucket distribution for "{word}":',
    lookup_result_row: 'Count: <strong>{count}</strong> / Bucket affinity: <strong>{pct}%</strong>',
    sandbox_result_prefix: 'Classification:',
    buckets_emails_col: '{total} msgs ({unread} unread)',
    buckets_docs_col: '{docs} docs',
    buckets_words_col: '{total} words ({unique} unique)',
    buckets_clear_btn: 'Clear Corpus',
    buckets_delete_btn: 'Delete',
    history_classified: '<span style="color:#38bdf8;">🤖 Auto-Classified (Bayes)</span>',
    history_magnet: '<span style="color:#fcd34d;">🧲 Magnet Matched</span>',
    history_reclassified: '<span style="color:#10b981;font-weight:700;">↻ TOE Reclassified</span>',
    history_empty: 'No history events yet',
    accounts_local_save: 'Local Outbox',
    accounts_fetch_btn: '📥 Fetch',
    accounts_edit_btn: '✏️ Edit',
    accounts_duplicate_btn: '📋 Duplicate',
    accounts_delete_btn: 'Delete',
    accounts_status_enabled: '🟢 Enabled',
    accounts_status_disabled: '⚪ Disabled',
    accounts_editing_badge: '✏️ Editing (ID: #{id} / Will Overwrite)',
    accounts_btn_save_new: '💾 Save Account',
    accounts_btn_save_overwrite: '💾 Overwrite & Save',
    accounts_copy_suffix: ' (Copy)',
    accounts_toast_loaded: 'Loaded "{name}" into editor',
    accounts_toast_updated: 'Updated account "{name}"',
    accounts_toast_created: 'Created new account "{name}"',
    accounts_toast_duplicated: 'Duplicated "{name}" (loaded in editor for adjustment)',
    accounts_empty: 'No external POP3/IMAP accounts configured (you can test all features immediately via Demo Generator or .eml Import)',
    accounts_interval_manual: 'Manual only',
    accounts_interval_mins: 'Every {min}m',
    accounts_exported_toast: 'Account settings exported successfully',
    accounts_imported_toast: 'Imported/restored {count} account configuration(s)',
    accounts_import_err_invalid: 'Invalid accounts backup file format (Please check JSON structure)',
    netlog_btn_view: '🔍 View Network Log',
    netlog_socket_connected: '🟢 TCP Socket: Connected ({local} ➔ {remote})',
    netlog_socket_disconnected: '🔴 TCP Socket: Failed (Not Connected)',
    netlog_tls_ok: '🔒 SSL/TLS: Established ({info})',
    netlog_tls_fail: '❌ SSL/TLS: Handshake Failed',
    netlog_tls_plain: '🔓 Plain TCP (No TLS)',
    netlog_stage_dns: '🌐 Cause: Server Hostname / DNS Resolution Error',
    netlog_stage_tcp: '🔌 Cause: Port Number / TCP Socket Connection Error',
    netlog_stage_tls: '🔒 Cause: SSL/TLS Handshake / Port Mismatch Error',
    netlog_stage_greeting: '📡 Cause: Invalid Server Greeting / Protocol Mismatch',
    netlog_stage_auth: '🔑 Cause: Account Authentication (Username / Password) Error',
    netlog_stage_command: '⚠️ Cause: Protocol Command Error',
    netlog_transcript_title: '📜 Socket & Protocol Trace Log (Passwords Masked)',
    netlog_edit_account_btn: '✏️ Edit This Account Setting',
    netlog_copied_toast: 'Copied network diagnostic log to clipboard',
  },
};

const I18N_BUCKET_LABELS = {
  ja: {
    inbox: '通常受信',
    work: '業務・開発',
    newsletter: 'メルマガ・情報',
    finance: '金融・請求',
    spam: '迷惑メール',
    unclassified: '未分類',
  },
  en: {
    inbox: 'Inbox',
    work: 'Work & Dev',
    newsletter: 'Newsletter',
    finance: 'Finance & Billing',
    spam: 'Spam',
    unclassified: 'Unclassified',
  },
};

// Exact text mapping for static DOM nodes in index.html (Japanese <-> English)
const I18N_DOM_TEXT_JA_TO_EN = {
  '⚡ デモ新着メール生成': '⚡ Generate Demo Mail',
  '📄 .emlインポート': '📄 Import .eml',
  '📥 送受信 (POP3/IMAP)': '📥 Fetch / Sync (POP3/IMAP/JMAP)',
  '📥 送受信 (POP3/IMAP/JMAP)': '📥 Fetch / Sync (POP3/IMAP/JMAP)',
  '🌐 言語・フォント設定': '🌐 Language & Fonts',
  '🔤 フォント設定': '🌐 Language & Fonts',
  '❓ ヘルプ / 使い方': '❓ Help / Manual',
  '✉️ 新規メール作成': '✉️ Compose Mail',
  '🎲 新着3通生成': '🎲 Simulate 3 Mails',
  '🔄 再分類スキャン': '🔄 Rescan All',
  'MUA フォルダ': 'MUA Folders',
  '📥 すべての受信箱': '📥 All Inbox',
  '🔵 未読メール': '🔵 Unread',
  '⭐ スター付き': '⭐ Starred',
  '📤 送信済みメール': '📤 Sent',
  '🗑️ ゴミ箱': '🗑️ Trash',
  'プロバイダ・アカウント別': 'By Provider / Account',
  '＋設定': '+ Setup',
  'POPFile バケット': 'POPFile Buckets',
  '＋管理': '+ Manage',
  'POPFile コントロールセンター': 'POPFile Control Center',
  '🧲 マグネット (ルール振り分け)': '🧲 Magnets (Priority Rules)',
  '📚 コーパス・単語確率辞書': '📚 Corpus & Word Inspector',
  '📊 バケット管理・分類統計': '📊 Buckets & Accuracy Stats',
  '🕒 分類・TOE学習履歴ログ': '🕒 Classification & TOE Log',
  '⚙️ POP3 / IMAP / SMTP 設定': '⚙️ POP3 / IMAP / JMAP / SMTP',
  '⚙️ POP3 / IMAP / JMAP / SMTP': '⚙️ POP3 / IMAP / JMAP / SMTP',
  '❓ 使い方・ヘルプガイド': '❓ Help & User Guide',
  '検索': 'Search',
  'プロバイダ別に分離': 'Group by Provider',
  'すべての受信箱': 'All Inbox',
  '↩ 削除を元に戻す': '↩ Undo Delete',
  '左のリストからメールを選択すると、本文および POPFile 単語確率・分類根拠が表示されます':
    'Select a message from the list on the left to view its body and POPFile Bayesian word probabilities',
  '🧲 POPFile マグネット (無条件振り分けルール)': '🧲 POPFile Magnets (Priority Routing Rules)',
  '特定の差出人・ドメイン・宛先・件名キーワードを含むメールを、ベイズ判定より優先して指定バケットへ確定振り分けします。':
    'Route messages matching specific From, Domain, To, or Subject patterns directly to a target bucket before Bayesian classification.',
  '🔄 既存メールにマグネットを再適用': '🔄 Re-apply Magnets to Existing Mails',
  '新規マグネットの作成': 'Create New Magnet Rule',
  '対象フィールド': 'Target Field',
  'From (差出人アドレス/名前)': 'From (Sender Address / Name)',
  'Domain (送信元ドメイン)': 'Domain (Sender Domain)',
  'Subject (件名キーワード)': 'Subject (Subject Keyword)',
  'To (宛先アドレス)': 'To (Recipient Address)',
  '一致パターン (部分一致)': 'Match Pattern (Substring)',
  '振り分け先バケット': 'Target Bucket',
  'メモ・説明 (任意)': 'Description (Optional)',
  '＋ マグネット追加': '+ Add Magnet',
  '状態': 'Status',
  '対象ヘッダ': 'Header Field',
  '一致パターン': 'Pattern',
  '発動回数': 'Hits',
  '説明': 'Description',
  '操作': 'Actions',
  '📚 POPFile コーパス辞書 & 単語確率インスペクタ': '📚 POPFile Corpus Dictionary & Word Probability Inspector',
  '各バケットが学習した単語（ヘッダ疑似トークン `from:`, `subject:`, `domain:` 含む）の出現頻度とベイズ確率を検証します。':
    'Inspect learned token frequencies and Bayesian probabilities across buckets (including header pseudo-tokens `from:`, `subject:`, `domain:`).',
  '🔍 単語のバケット間比較 (Word Lookup)': '🔍 Cross-Bucket Word Lookup',
  '単語を入力して検索すると、全バケットでの出現回数と親和度を表示します。':
    'Enter a word or token above to compare its occurrence count and affinity across all buckets.',
  '🧪 リアルタイム分類シミュレータ (Sandbox)': '🧪 Live Classification Simulator (Sandbox)',
  '⚡ ベイズ判定を実行': '⚡ Run Bayesian Classification',
  'バケット別コーパス一覧:': 'Corpus Words by Bucket:',
  'トークン (単語 / 疑似ヘッダ)': 'Token (Word / Pseudo-Header)',
  'このバケットでの出現回数': 'Count in Bucket',
  'バケット内確率 P(w|B)': 'Within-Bucket Prob P(w|B)',
  '全バケットでの分布内訳': 'Distribution Across All Buckets',
  '📊 POPFile バケット管理 & 分類精度統計': '📊 POPFile Bucket Management & Accuracy Statistics',
  'マルチバケットの作成・カラー設定・コーパス初期化および TOE (Train On Error) 統計を確認できます。':
    'Create custom buckets, customize colors, reset bucket corpora, and review TOE (Train On Error) accuracy statistics.',
  '＋ 新規バケットの作成': '+ Create New Bucket',
  'バケットID (英数字)': 'Bucket ID (alphanumeric)',
  '表示ラベル': 'Display Label',
  'テーマカラー': 'Theme Color',
  '＋ バケット作成': '+ Create Bucket',
  'バケット名': 'Bucket Label',
  'ID (X-Text-Classification)': 'ID (X-Text-Classification)',
  '現在メール数 (未読)': 'Messages (Unread)',
  '学習文書数': 'Trained Docs',
  '総単語数 (ユニーク語数)': 'Total Words (Unique)',
  '事前確率 P(B)': 'Prior Prob P(B)',
  '誤判定数 (FP / FN)': 'Errors (FP / FN)',
  '🕒 POPFile 分類・再学習 (TOE) 履歴ログ': '🕒 POPFile Classification & TOE Training History',
  '受信時の自動分類結果、マグネット発動、およびユーザーによる再分類（TOE学習）の履歴です。':
    'Chronological log of automatic Bayesian classifications, magnet matches, and manual TOE reclassifications.',
  '🔄 更新': '🔄 Refresh',
  '日時': 'Timestamp',
  'アクション': 'Action',
  '件名': 'Subject',
  '差出人': 'From',
  '変更前バケット': 'Previous Bucket',
  '振り分け・学習先バケット': 'Target Bucket',
  '⚙️ メールアカウント設定 (POP3 / IMAP4 / SMTP)': '⚙️ Mail Accounts & Language Settings (POP3 / IMAP4 / JMAP / SMTP)',
  '⚙️ メールアカウント設定 (POP3 / IMAP4 / JMAP / SMTP)': '⚙️ Mail Accounts & Language Settings (POP3 / IMAP4 / JMAP / SMTP)',
  '実際のPOP3またはIMAPサーバーからメールを受信してPOPFileエンジンで自動分類し、SMTPで送信できます。':
    'Receive messages from POP3, IMAP4, or JMAP servers with automatic POPFile Bayesian classification, and send messages via SMTP.',
  'POP3、IMAP4、または最新のJMAP (RFC 8620/8621) サーバーからメールを受信して自動分類し、SMTPで送信できます。':
    'Receive messages from POP3, IMAP4, or modern JMAP (RFC 8620/8621) servers with automatic Bayesian classification, and send via SMTP.',
  '🌐 表示言語・フォント設定 (Language & Appearance)': '🌐 Language & Appearance Settings',
  '表示言語 (Display Language)': 'Display Language',
  '🔤 フォント・文字サイズ設定を開く': '🔤 Open Font & Text Size Settings',
  'アカウント追加・編集': 'Add / Edit Mail Account',
  'アカウント表示名': 'Account Display Name',
  'メールアドレス': 'Email Address',
  '受信プロトコル': 'Incoming Protocol',
  'JMAP (RFC 8620/8621)': 'JMAP (RFC 8620/8621)',
  '受信サーバー (Host)': 'Incoming Server (Host)',
  '受信ポート': 'Incoming Port',
  '受信TLS': 'Incoming TLS',
  'SSL/TLS 有効': 'SSL/TLS Enabled',
  '平文 (非TLS)': 'Plain (No TLS)',
  'ログインユーザー名': 'Login Username',
  'パスワード / アプリパスワード': 'Password / App Password',
  'SMTP送信サーバー (Host)': 'SMTP Server (Host)',
  'SMTPポート': 'SMTP Port',
  '有効': 'Enabled',
  '無効': 'Disabled',
  '定期チェック間隔': 'Check Interval',
  '定期チェック': 'Auto Check',
  '手動受信のみ': 'Manual Only',
  '1分ごと': 'Every 1 min',
  '5分ごと': 'Every 5 mins',
  '10分ごと': 'Every 10 mins',
  '15分ごと': 'Every 15 mins',
  '30分ごと': 'Every 30 mins',
  '60分ごと': 'Every 60 mins',
  '📤 アカウント設定エクスポート (JSON)': '📤 Export Accounts (JSON)',
  '📥 アカウント設定インポート (JSON)': '📥 Import Accounts (JSON)',
  '💾 アカウント保存': '💾 Save Account',
  '📋 別エントリとして複製保存': '📋 Save as New Copy',
  '＋ 新規作成に切り替え (フォームをクリア)': '+ Switch to New (Clear Form)',
  '💡 登録済みアカウントの行をクリックすると、上の編集フォームに呼び戻して上書き保存や複製ができます。':
    '💡 Click any saved account row below to load it into the editor above for updating or duplicating.',
  '⚠️ 通信エラー診断ログ': '⚠️ Network Error Log',
  '直前のメール送受信で通信・認証エラーが発生しました': 'A network or authentication error occurred during the last mail sync/send',
  'ソケットの接続状態、アドレス・ポート番号の誤り、SSL/TLS設定、またはアカウント認証のどこで失敗したかをネットワークログで確認できます。':
    'Inspect the network log to verify socket connection state, hostname/port settings, SSL/TLS negotiation, or account authentication.',
  '🔍 ネットワーク診断ログを表示': '🔍 View Network Diagnostic Log',
  '✕ 閉じる': '✕ Dismiss',
  '🔍 送受信エラー・ネットワーク診断ログ': '🔍 Mail Sync/Send Network Diagnostic Log',
  '📋 診断ログをクリップボードにコピー': '📋 Copy Diagnostic Log to Clipboard',
  'アカウント名': 'Account Name',
  '受信方式': 'Protocol',
  '受信サーバー': 'Incoming Host',
  'SMTPサーバー': 'SMTP Host',
  '❓ Tunbridge 使い方・操作マニュアル': '❓ Tunbridge User Guide & Manual',
  'マルチバケット・ナイーブベイズ自動分類、TOE学習、マグネット、単語確率インスペクタの仕組みと操作手順を解説します。':
    'Learn how multi-bucket Naive Bayes classification, TOE training, magnets, and the word probability inspector work.',
  '📬 メーラー画面へ戻る': '📬 Back to Mail View',
  '📖 1. 概要・クイックスタート': '📖 1. Overview & Quickstart',
  '📬 2. メーラー操作 & TOE再学習': '📬 2. Mail View & TOE Training',
  '🔬 3. 単語確率・コーパス辞書': '🔬 3. Word Scores & Corpus',
  '🧲 4. マグネット & バケット管理': '🧲 4. Magnets & Buckets',
  '⚙️ 5. POP3/IMAP設定 & FAQ': '⚙️ 5. POP3/IMAP Setup & FAQ',
  '✉️ メール作成・送信': '✉️ Compose & Send Mail',
  '送信アカウント': 'From Account',
  '宛先 (To)': 'To',
  'Cc (任意)': 'Cc (Optional)',
  'Bcc (任意)': 'Bcc (Optional)',
  '件名 (Subject)': 'Subject',
  '本文': 'Body',
  '📎 添付ファイル (複数選択・ドラッグ＆ドロップ対応)': '📎 Attachments (Multi-select & Drag-and-Drop)',
  '＋ ファイルを追加': '+ Add Files',
  'ここにファイルをドロップするか、「＋ ファイルを追加」で添付できます':
    'Drop files here or click "+ Add Files" to attach',
  'キャンセル': 'Cancel',
  '📤 送信する': '📤 Send Message',
  '⚡ デモ新着メール生成 (自動分類テスト)': '⚡ Generate Sample Incoming Messages',
  '未学習の新着サンプルメールを受信させ、POPFileのナイーブベイズ分類器およびマグネットがリアルタイムにどのバケットへ振り分けるか試すことができます。':
    'Generate untrained incoming sample messages to test how the POPFile Naive Bayes classifier and magnets route them in real time.',
  '生成するメールのカテゴリ傾向': 'Sample Message Category',
  '🎲 ランダム (全カテゴリから混合生成)': '🎲 Random (Mix of all categories)',
  '🚨 迷惑メール・フィッシング (Spam)': '🚨 Spam & Phishing (Spam)',
  '💼 業務・プロジェクト・開発 (Work)': '💼 Work, Projects & Dev (Work)',
  '🏦 金融・銀行明細・請求 (Finance)': '🏦 Finance, Banking & Statements (Finance)',
  '📰 メルマガ・EC案内 (Newsletter)': '📰 Newsletters & Promotions (Newsletter)',
  '🏠 個人連絡・日常メール (Inbox)': '🏠 Personal & Daily Mail (Inbox)',
  '生成通数': 'Number of Messages',
  '1 通': '1 message',
  '3 通': '3 messages',
  '5 通': '5 messages',
  '閉じる': 'Close',
  '⚡ 新着メールを受信・自動分類': '⚡ Generate & Auto-Classify',
  '📄 RFC822 (.eml) メールインポート & 自動分類': '📄 Import RFC822 (.eml) & Auto-Classify',
  '.eml ファイルを選択、または下にRFC822メールソースを貼り付けてください':
    'Select a .eml file or paste raw RFC822 message source below',
  '生メールソース (Headers + Body)': 'Raw Message Source (Headers + Body)',
  '📥 インポートしてPOPFile分類': '📥 Import & Classify',
  '⚙️ 常時表示するメールヘッダのカスタマイズ': '⚙️ Customize Always-Visible Mail Headers',
  'プリセットヘッダから選択:': 'Choose from Preset Headers:',
  '任意のカスタムヘッダ名を追加:': 'Add Custom Header Name:',
  '＋ 追加': '+ Add',
  '標準に戻す': 'Reset to Default',
  '✓ 完了': '✓ Done',
  '🌐 言語・フォント・文字サイズの設定': '🌐 Language, Font & Text Size Settings',
  '🌐 言語・テーマ・フォントの設定': '🌐 Language, Theme & Font Settings',
  '🌐 表示言語・テーマ・フォント設定 (Language, Theme & Appearance)': '🌐 Language, Theme & Appearance Settings',
  '🌐 表示言語 & 🌓 テーマ設定 (Language & Theme)': '🌐 Display Language & 🌓 App Theme',
  'アプリテーマ (App Theme)': 'App Theme',
  '言語 (Language)': 'Display Language',
  '🌓 自動 (OS)': '🌓 Auto (OS)',
  '🌙 ダーク': '🌙 Dark',
  '☀️ ライト': '☀️ Light',
  '🌓 自動（OS設定に追随 / Follow OS）': '🌓 Auto (Follow OS Dark/Light Mode)',
  '🌓 自動（OSのダーク/ライト設定に追随 / System Auto）': '🌓 Auto (Follow OS Dark/Light Mode)',
  '🌙 ダークテーマ (Dark)': '🌙 Dark Theme',
  '🌙 ダークテーマ (Dark Theme)': '🌙 Dark Theme',
  '☀️ ライトテーマ (Light)': '☀️ Light Theme',
  '☀️ ライトテーマ (Light Theme)': '☀️ Light Theme',
  '🌐 表示言語設定 (Display Language)': '🌐 Display Language',
  '言語 (Language) — 環境に合わせた自動判定または手動指定': 'Language — Auto-detect from environment or select manually',
  '🌐 自動 (Auto)': '🌐 Auto',
  '🌐 自動（環境設定に合わせる / Auto - Detect Environment）': '🌐 Auto (Detect from system/browser)',
  '🌐 自動（OS・ブラウザ環境設定に合わせる / Auto）': '🌐 Auto (Detect from system/browser)',
  '🇯🇵 日本語': '🇯🇵 日本語 (Japanese)',
  '🇯🇵 日本語 (Japanese)': '🇯🇵 日本語 (Japanese)',
  '🇺🇸 English': '🇺🇸 English',
  '🇺🇸 English (英語)': '🇺🇸 English',
  '📋 メールヘッダ部のフォント設定': '📋 Mail Header Font Settings',
  'ヘッダ用フォント種類（プリセット）': 'Header Font Family (Preset)',
  'フォント名直接指定': 'Custom Font Family',
  '📄 メール本文部 (TextArea / ビュー) のフォント設定': '📄 Message Body (TextArea / Reader) Font Settings',
  '本文用フォント種類（プリセット）': 'Body Font Family (Preset)',
  'リアルタイムプレビュー:': 'Live Preview:',
  '標準サイズ・フォントに戻す': 'Reset Fonts to Default',
};

const I18N_PLACEHOLDER_JA_TO_EN = {
  '件名・差出人・本文を検索...': 'Search subject, sender, or body...',
  '例: vip-partner.co.jp または 【重要】': 'e.g. vip-partner.co.jp or [URGENT]',
  'ルールの説明...': 'Rule description...',
  '例: 無料, 議事録, 口座, subject:当選, domain:corp-tech.co.jp': 'e.g. invoice, meeting, subject:urgent, domain:corp-tech.co.jp',
  'テスト件名を入力 (例: 【至急】口座振替と請求明細のご案内)': 'Enter test subject (e.g. Monthly account statement)',
  'テスト本文を入力...': 'Enter test message body...',
  '単語フィルタ...': 'Filter tokens...',
  '例: shopping': 'e.g. shopping',
  '例: EC・通販 (Shopping)': 'e.g. Shopping & Orders',
  '注文確認・発送通知など': 'Order confirmations, shipping notices, etc.',
  '例: 社内メール / Gmail': 'e.g. Work Mail / Gmail',
  'smtp.example.com (空欄時はローカル送信箱保存)': 'smtp.example.com (leave blank for local outbox)',
  '件名を入力...': 'Enter subject...',
  'メール本文を入力...': 'Enter message body...',
  '例: X-Spam-Status, List-Unsubscribe, Received': 'e.g. X-Spam-Status, List-Unsubscribe, Received',
  'フォントファミリ名': 'Font family name',
};

function loadSavedLangPref() {
  try {
    const raw = localStorage.getItem('tunbridge_lang_pref');
    if (raw === 'ja' || raw === 'en' || raw === 'auto') {
      return raw;
    }
  } catch (e) {}
  return 'auto';
}

function detectEnvironmentLang() {
  try {
    const langs =
      Array.isArray(navigator.languages) && navigator.languages.length > 0
        ? navigator.languages
        : [navigator.language || navigator.userLanguage || 'ja'];
    for (const l of langs) {
      const low = String(l || '').toLowerCase();
      if (low.startsWith('ja')) return 'ja';
      if (low.startsWith('en')) return 'en';
    }
  } catch (e) {}
  return 'ja';
}

let currentLangPref = loadSavedLangPref();

function getCurrentLang() {
  if (currentLangPref === 'ja' || currentLangPref === 'en') {
    return currentLangPref;
  }
  return detectEnvironmentLang();
}

function getLangPreference() {
  return currentLangPref;
}

function t(key, vars = {}) {
  const lang = getCurrentLang();
  const dict = I18N_RESOURCES[lang] || I18N_RESOURCES.ja;
  let str = dict[key] !== undefined ? dict[key] : I18N_RESOURCES.ja[key] || key;
  for (const [k, v] of Object.entries(vars)) {
    str = str.replaceAll(`{${k}}`, String(v));
  }
  return str;
}

function trBucketLabel(bucketOrName, fallbackLabel = '') {
  const lang = getCurrentLang();
  const map = I18N_BUCKET_LABELS[lang] || I18N_BUCKET_LABELS.ja;
  if (bucketOrName && typeof bucketOrName === 'object') {
    const nameKey = String(bucketOrName.name || '').toLowerCase();
    if (map[nameKey]) return map[nameKey];
    return bucketOrName.label || bucketOrName.name || fallbackLabel;
  }
  const key = String(bucketOrName || '').toLowerCase();
  if (map[key]) return map[key];
  // Also check if a Japanese system label was passed directly
  for (const [sysName, jaLabel] of Object.entries(I18N_BUCKET_LABELS.ja)) {
    if (jaLabel === bucketOrName && map[sysName]) {
      return map[sysName];
    }
  }
  return fallbackLabel || String(bucketOrName || '');
}

function applyI18nToDOM() {
  const lang = getCurrentLang();
  document.documentElement.lang = lang;
  document.title = t('app_title');

  // Sync language select dropdowns
  ['topbarLangSelect', 'modalLangSelect', 'accountsViewLangSelect'].forEach((id) => {
    const sel = document.getElementById(id);
    if (sel) sel.value = currentLangPref;
  });

  // Update topbar stat labels while keeping dynamic <strong> values intact
  const statPills = document.querySelectorAll('#topbarStats .stat-pill');
  if (statPills.length >= 4) {
    const accStrong = document.getElementById('statAccuracy');
    const totStrong = document.getElementById('statTotalEmails');
    const vocStrong = document.getElementById('statVocab');
    const magStrong = document.getElementById('statMagnetHits');
    if (accStrong) {
      statPills[0].innerHTML = `${t('stat_accuracy')} <strong id="statAccuracy">${accStrong.textContent}</strong>`;
    }
    if (totStrong) {
      statPills[1].innerHTML = `${t('stat_total_emails')} <strong id="statTotalEmails">${totStrong.textContent}</strong>`;
    }
    if (vocStrong) {
      statPills[2].innerHTML = `${t('stat_vocab')} <strong id="statVocab">${vocStrong.textContent}</strong> ${t('stat_vocab_unit')}`;
    }
    if (magStrong) {
      statPills[3].innerHTML = `${t('stat_magnet_hits')} <strong id="statMagnetHits">${magStrong.textContent}</strong> ${t('stat_magnet_unit')}`;
    }
  }

  // Update keyboard hint bar
  const kbdHint = document.querySelector('.mail-kbd-hint');
  if (kbdHint) {
    if (lang === 'en') {
      kbdHint.innerHTML = `
        <span><kbd>↑</kbd><kbd>↓</kbd> Move</span>
        <span><kbd>PgUp</kbd><kbd>PgDn</kbd> Page</span>
        <span><kbd>Home</kbd><kbd>End</kbd> First/Last</span>
        <span><kbd>Del</kbd> Delete</span>
        <span><kbd>Ctrl+Z</kbd> Undo</span>
      `;
    } else {
      kbdHint.innerHTML = `
        <span><kbd>↑</kbd><kbd>↓</kbd> 移動</span>
        <span><kbd>PgUp</kbd><kbd>PgDn</kbd> ページ</span>
        <span><kbd>Home</kbd><kbd>End</kbd> 先頭/末尾</span>
        <span><kbd>Del</kbd> 削除</span>
        <span><kbd>Ctrl+Z</kbd> 元に戻す</span>
      `;
    }
  }

  // Walk leaf text elements and translate between JA and EN using stored original JA text
  const allElements = document.querySelectorAll(
    'button, label, th, span, div, h3, option, p'
  );
  allElements.forEach((el) => {
    // Only translate elements without element children (or options/buttons with text only)
    if (el.children.length === 0) {
      const curText = el.textContent.trim();
      if (!curText) return;
      if (!el.dataset.i18nOrigJa) {
        if (I18N_DOM_TEXT_JA_TO_EN[curText]) {
          el.dataset.i18nOrigJa = curText;
        }
      }
      const origJa = el.dataset.i18nOrigJa;
      if (origJa && I18N_DOM_TEXT_JA_TO_EN[origJa]) {
        el.textContent = lang === 'en' ? I18N_DOM_TEXT_JA_TO_EN[origJa] : origJa;
      }
    }
  });

  // Translate input/textarea placeholders
  document.querySelectorAll('input[placeholder], textarea[placeholder]').forEach((el) => {
    const curPh = el.getAttribute('placeholder') || '';
    if (!el.dataset.i18nOrigPh && I18N_PLACEHOLDER_JA_TO_EN[curPh]) {
      el.dataset.i18nOrigPh = curPh;
    }
    const origPh = el.dataset.i18nOrigPh;
    if (origPh && I18N_PLACEHOLDER_JA_TO_EN[origPh]) {
      el.setAttribute('placeholder', lang === 'en' ? I18N_PLACEHOLDER_JA_TO_EN[origPh] : origPh);
    }
  });
}
