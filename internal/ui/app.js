const DEFAULT_PINNED_HEADERS = [
  'Date',
  'X-Text-Classification',
  'X-POPFile-Confidence',
  'Message-ID',
];

const PRESET_HEADER_OPTIONS = [
  { name: 'Date', label: 'Date (送信日時)', labelEn: 'Date (Sent Date)' },
  { name: 'X-Text-Classification', label: 'X-Text-Classification (自動分類)', labelEn: 'X-Text-Classification (Classification)' },
  { name: 'X-POPFile-Confidence', label: 'X-POPFile-Confidence (POPFile互換・確信度)', labelEn: 'X-POPFile-Confidence (POPFile Confidence)' },
  { name: 'X-Bayes-Confidence', label: 'X-Bayes-Confidence (確信度)', labelEn: 'X-Bayes-Confidence (Confidence)' },
  { name: 'Message-ID', label: 'Message-ID (メッセージ識別子)', labelEn: 'Message-ID' },
  { name: 'Reply-To', label: 'Reply-To (返信先アドレス)', labelEn: 'Reply-To' },
  { name: 'Return-Path', label: 'Return-Path (バウンス返送先)', labelEn: 'Return-Path (Bounce Return)' },
  { name: 'User-Agent', label: 'User-Agent (送信メーラー)', labelEn: 'User-Agent (Mail Client)' },
  { name: 'X-Mailer', label: 'X-Mailer (メーラー情報)', labelEn: 'X-Mailer (Mailer Info)' },
  { name: 'Content-Type', label: 'Content-Type (コンテンツ形式)', labelEn: 'Content-Type' },
  { name: 'MIME-Version', label: 'MIME-Version', labelEn: 'MIME-Version' },
];

function loadSavedPinnedHeaders() {
  try {
    const raw = localStorage.getItem('tunbridge_pinned_headers') || localStorage.getItem('popfile_pinned_headers');
    if (raw) {
      const arr = JSON.parse(raw);
      if (Array.isArray(arr)) {
        return arr.map((h) => (h === 'X-Bayes-Confidence' ? 'X-POPFile-Confidence' : h));
      }
    }
  } catch (e) {}
  return [...DEFAULT_PINNED_HEADERS];
}

const DEFAULT_FONT_SETTINGS = {
  headerFontFamily: '"BIZ UDGothic", "Yu Gothic UI", "Meiryo", sans-serif',
  headerFontSize: 14.5,
  bodyFontFamily: '"BIZ UDGothic", "Consolas", "Meiryo", monospace',
  bodyFontSize: 16,
  bodyLineHeight: 1.75,
};

function loadSavedFontSettings() {
  try {
    const raw = localStorage.getItem('tunbridge_font_settings') || localStorage.getItem('popfile_font_settings');
    if (raw) {
      const obj = JSON.parse(raw);
      if (obj && typeof obj === 'object') {
        return { ...DEFAULT_FONT_SETTINGS, ...obj };
      }
    }
  } catch (e) {}
  return { ...DEFAULT_FONT_SETTINGS };
}

function loadSavedGroupByProvider() {
  try {
    const raw = localStorage.getItem('tunbridge_group_by_provider') ?? localStorage.getItem('popfile_group_by_provider');
    if (raw !== null) return raw === '1';
  } catch (e) {}
  return true;
}

function loadSavedThemePref() {
  try {
    const raw = localStorage.getItem('tunbridge_theme_pref');
    if (raw === 'dark' || raw === 'light' || raw === 'auto') {
      return raw;
    }
  } catch (e) {}
  return 'auto';
}

function detectEnvironmentTheme() {
  try {
    if (window.matchMedia && window.matchMedia('(prefers-color-scheme: light)').matches) {
      return 'light';
    }
  } catch (e) {}
  return 'dark';
}

let currentThemePref = loadSavedThemePref();

function getThemePreference() {
  return currentThemePref;
}

function getCurrentTheme() {
  if (currentThemePref === 'dark' || currentThemePref === 'light') {
    return currentThemePref;
  }
  return detectEnvironmentTheme();
}

function applyThemeToDOM() {
  const theme = getCurrentTheme();
  document.documentElement.setAttribute('data-theme', theme);
  document.documentElement.style.colorScheme = theme;
  ['topbarThemeSelect', 'modalThemeSelect', 'accountsViewThemeSelect'].forEach((id) => {
    const sel = document.getElementById(id);
    if (sel) sel.value = currentThemePref;
  });
}

function setThemePreference(pref) {
  currentThemePref = pref === 'dark' || pref === 'light' ? pref : 'auto';
  try {
    localStorage.setItem('tunbridge_theme_pref', currentThemePref);
  } catch (e) {}
  applyThemeToDOM();
}

// Listen for OS theme changes so 'auto' follows OS Dark/Light mode in real time
try {
  if (window.matchMedia) {
    const mq = window.matchMedia('(prefers-color-scheme: light)');
    const onSchemeChange = () => {
      if (currentThemePref === 'auto') {
        applyThemeToDOM();
      }
    };
    if (typeof mq.addEventListener === 'function') {
      mq.addEventListener('change', onSchemeChange);
    } else if (typeof mq.addListener === 'function') {
      mq.addListener(onSchemeChange);
    }
  }
} catch (e) {}

// Apply theme immediately to avoid flash
applyThemeToDOM();

// Global App State
const state = {
  buckets: [],
  stats: {},
  counts: {},
  currentView: 'mua',
  folder: 'received',
  bucketId: 0,
  unreadOnly: false,
  starredOnly: false,
  providerFilter: 'all', // 'all' or provider key (e.g. 'gmail.com', 'outlook.com', 'demo')
  groupByProvider: loadSavedGroupByProvider(),
  collapsedProviders: {}, // key -> boolean
  emails: [],
  visibleEmails: [], // ordered list of emails currently displayed in the list
  selectedEmailId: 0,
  selectedDetail: null, // { email, explanation }
  selectedAttachmentId: 0, // currently previewed attachment ID in the right split pane
  readerTab: 'body', // 'body', 'colored', 'matrix', 'headers'
  accounts: [],
  pinnedHeaders: loadSavedPinnedHeaders(),
  showAllOtherHeaders: false,
  fontSettings: loadSavedFontSettings(),
  undoStack: [], // stack of delete operations for safe Undo
  lastFailedNetLogs: [], // NetworkDiagReport[] populated ONLY when a send/receive fails
};

function applyFontSettings() {
  const fs = state.fontSettings || DEFAULT_FONT_SETTINGS;
  const root = document.documentElement;
  root.style.setProperty('--header-font-family', fs.headerFontFamily);
  root.style.setProperty('--header-font-size', `${fs.headerFontSize}px`);
  root.style.setProperty('--body-font-family', fs.bodyFontFamily);
  root.style.setProperty('--body-font-size', `${fs.bodyFontSize}px`);
  root.style.setProperty('--body-line-height', String(fs.bodyLineHeight || 1.75));
}

function saveFontSettings() {
  try {
    localStorage.setItem('tunbridge_font_settings', JSON.stringify(state.fontSettings));
  } catch (e) {}
}

function openFontConfigModal() {
  const fs = state.fontSettings;
  const syncPreset = (selectId, customId, val) => {
    const sel = document.getElementById(selectId);
    const cust = document.getElementById(customId);
    if (!sel || !cust) return;
    const match = Array.from(sel.options).find((o) => o.value === val);
    if (match) {
      sel.value = val;
      cust.value = '';
    } else {
      sel.value = 'custom';
      cust.value = val;
    }
  };

  syncPreset('headerFontPreset', 'headerFontCustom', fs.headerFontFamily);
  syncPreset('bodyFontPreset', 'bodyFontCustom', fs.bodyFontFamily);

  document.getElementById('headerFontSizeRange').value = fs.headerFontSize;
  document.getElementById('headerFontSizeLabel').textContent = `${fs.headerFontSize}px`;
  document.getElementById('bodyFontSizeRange').value = fs.bodyFontSize;
  document.getElementById('bodyFontSizeLabel').textContent = `${fs.bodyFontSize}px`;
  document.getElementById('bodyLineHeightRange').value = fs.bodyLineHeight || 1.75;
  document.getElementById('bodyLineHeightLabel').textContent = String(fs.bodyLineHeight || 1.75);

  const modalLangSel = document.getElementById('modalLangSelect');
  if (modalLangSel && typeof getLangPreference === 'function') {
    modalLangSel.value = getLangPreference();
  }
  const modalThemeSel = document.getElementById('modalThemeSelect');
  if (modalThemeSel && typeof getThemePreference === 'function') {
    modalThemeSel.value = getThemePreference();
  }

  document.getElementById('fontConfigModal').classList.add('open');
}

function onFontPresetChange(target) {
  if (target === 'header') {
    const v = document.getElementById('headerFontPreset').value;
    if (v !== 'custom') {
      document.getElementById('headerFontCustom').value = '';
    }
  } else {
    const v = document.getElementById('bodyFontPreset').value;
    if (v !== 'custom') {
      document.getElementById('bodyFontCustom').value = '';
    }
  }
  updateFontSettingsFromUI();
}

function updateFontSettingsFromUI() {
  const hPreset = document.getElementById('headerFontPreset').value;
  const hCustom = document.getElementById('headerFontCustom').value.trim();
  const bPreset = document.getElementById('bodyFontPreset').value;
  const bCustom = document.getElementById('bodyFontCustom').value.trim();

  const hSize = parseFloat(document.getElementById('headerFontSizeRange').value) || 14.5;
  const bSize = parseFloat(document.getElementById('bodyFontSizeRange').value) || 16;
  const bLine = parseFloat(document.getElementById('bodyLineHeightRange').value) || 1.75;

  state.fontSettings = {
    headerFontFamily: hCustom ? hCustom : hPreset !== 'custom' ? hPreset : state.fontSettings.headerFontFamily,
    headerFontSize: hSize,
    bodyFontFamily: bCustom ? bCustom : bPreset !== 'custom' ? bPreset : state.fontSettings.bodyFontFamily,
    bodyFontSize: bSize,
    bodyLineHeight: bLine,
  };

  document.getElementById('headerFontSizeLabel').textContent = `${hSize}px`;
  document.getElementById('bodyFontSizeLabel').textContent = `${bSize}px`;
  document.getElementById('bodyLineHeightLabel').textContent = String(bLine);

  applyFontSettings();
  saveFontSettings();
}

function stepFontSize(target, delta) {
  if (target === 'header') {
    const el = document.getElementById('headerFontSizeRange');
    const next = Math.min(24, Math.max(11, (parseFloat(el.value) || 14.5) + delta));
    el.value = next;
  } else {
    const el = document.getElementById('bodyFontSizeRange');
    const next = Math.min(28, Math.max(12, (parseFloat(el.value) || 16) + delta));
    el.value = next;
  }
  updateFontSettingsFromUI();
}

function resetFontSettingsDefault() {
  state.fontSettings = { ...DEFAULT_FONT_SETTINGS };
  applyFontSettings();
  saveFontSettings();
  openFontConfigModal();
  showToast(getCurrentLang() === 'en' ? 'Fonts and text sizes reset to default' : 'フォント種類と文字サイズを標準設定に戻しました');
}

function setLanguagePreference(pref) {
  currentLangPref = pref === 'ja' || pref === 'en' ? pref : 'auto';
  try {
    localStorage.setItem('tunbridge_lang_pref', currentLangPref);
  } catch (e) {}
  applyI18nToDOM();
  updateCurrentListTitle();
  renderBucketSidebar();
  populateBucketSelects();
  renderProviderFilterUI();
  renderEmailList();
  updateUndoButtonUI();
  if (state.selectedDetail && state.selectedDetail.email) {
    renderMailReader();
  } else {
    renderEmptyReader();
  }
  if (state.currentView === 'magnets') loadMagnetsView();
  if (state.currentView === 'corpus') loadCorpusWords();
  if (state.currentView === 'buckets') renderBucketsTable();
  if (state.currentView === 'history') loadHistoryView();
  if (state.currentView === 'accounts') loadAccountsList();
}

function updateCurrentListTitle() {
  const el = document.getElementById('currentListTitle');
  if (!el) return;
  if (state.bucketId > 0) {
    const b = state.buckets.find((x) => x.id === state.bucketId);
    const label = b ? trBucketLabel(b) : '';
    el.textContent = `${t('bucket_prefix')}${label}`;
    return;
  }
  const titles = {
    received: state.unreadOnly ? t('folder_unread') : t('folder_received'),
    all: t('folder_starred'),
    sent: t('folder_sent'),
    trash: t('folder_trash'),
  };
  el.textContent = titles[state.folder] || t('folder_default');
}

document.addEventListener('DOMContentLoaded', async () => {
  applyThemeToDOM();
  applyFontSettings();
  applyI18nToDOM();
  const grpChk = document.getElementById('groupByProviderCheckbox');
  if (grpChk) grpChk.checked = state.groupByProvider;
  setupKeyboardNavigation();
  initMailPaneResizer();
  initWindowStateListener();
  await loadAccountsList();
  await refreshOverview();
  updateCurrentListTitle();
  await loadEmails();
  startBackgroundPoll();
});

function showToast(msg, color = '#3b82f6', actionLabel = '', actionCallback = null, duration = 4200) {
  const c = document.getElementById('toastContainer');
  const el = document.createElement('div');
  el.className = 'toast';
  el.style.borderLeftColor = color;
  el.style.display = 'flex';
  el.style.alignItems = 'center';
  el.style.justifyContent = 'space-between';
  el.style.gap = '10px';

  const span = document.createElement('span');
  span.textContent = msg;
  el.appendChild(span);

  if (actionLabel && typeof actionCallback === 'function') {
    const btn = document.createElement('button');
    btn.className = 'toast-undo-btn';
    btn.textContent = actionLabel;
    btn.onclick = () => {
      el.remove();
      actionCallback();
    };
    el.appendChild(btn);
  }

  c.appendChild(el);
  setTimeout(() => {
    if (el.parentNode) el.remove();
  }, duration);
}

function escapeHTML(str) {
  if (!str) return '';
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

async function apiGet(url) {
  const res = await fetch(url);
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || 'API Error');
  return data;
}

async function apiPost(url, body) {
  const res = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body || {}),
  });
  const data = await res.json();
  if (!res.ok) {
    const err = new Error(data.error || 'API Error');
    if (Array.isArray(data.net_logs)) {
      err.netLogs = data.net_logs;
    }
    throw err;
  }
  return data;
}

async function refreshOverview() {
  try {
    const data = await apiGet('/api/overview');
    state.buckets = data.buckets || [];
    state.stats = data.stats || {};
    state.counts = data.counts || {};

    // Topbar stats
    document.getElementById('statAccuracy').textContent =
      (state.stats.accuracy_rate !== undefined ? state.stats.accuracy_rate.toFixed(1) : '100.0') + '%';
    document.getElementById('statTotalEmails').textContent = state.stats.total_emails || 0;
    document.getElementById('statVocab').textContent = state.stats.unique_vocab_size || 0;
    document.getElementById('statMagnetHits').textContent = state.stats.magnet_hits || 0;

    // Folder badges
    document.getElementById('badgeReceived').textContent = state.counts.received || 0;
    document.getElementById('badgeUnread').textContent = state.counts.unread || 0;
    document.getElementById('badgeStarred').textContent = state.counts.starred || 0;
    document.getElementById('badgeSent').textContent = state.counts.sent || 0;
    document.getElementById('badgeTrash').textContent = state.counts.trash || 0;

    renderBucketSidebar();
    populateBucketSelects();
    lastKnownTotalReceived = Number(state.counts.received) || 0;
  } catch (e) {
    console.error(e);
  }
}

let lastKnownTotalReceived = -1;
let backgroundPollTimer = null;

function startBackgroundPoll() {
  if (backgroundPollTimer) clearInterval(backgroundPollTimer);
  backgroundPollTimer = setInterval(async () => {
    try {
      const data = await apiGet('/api/overview');
      if (!data || !data.counts) return;
      const currentReceived = Number(data.counts.received) || 0;
      if (lastKnownTotalReceived >= 0 && currentReceived > lastKnownTotalReceived) {
        const diff = currentReceived - lastKnownTotalReceived;
        const isEn = typeof getCurrentLang === 'function' && getCurrentLang() === 'en';
        showToast(
          isEn ? `📥 ${diff} new message(s) received` : `📥 新着メールが ${diff} 通届きました`,
          '#10b981'
        );
        lastKnownTotalReceived = currentReceived;
        await refreshOverview();
        if (state.currentFolder === 'received' || !state.currentFolder) {
          await loadEmails();
        }
        if (state.currentMainView === 'accounts') {
          await loadAccountsList();
        }
      } else {
        lastKnownTotalReceived = currentReceived;
      }
    } catch (e) {
      // background poll silently ignores transient errors
    }
  }, 15000);
}

function renderBucketSidebar() {
  const container = document.getElementById('bucketNavList');
  container.innerHTML = state.buckets
    .map((b) => {
      const isActive = state.currentView === 'mua' && state.bucketId === b.id;
      const badgeText = b.unread_count > 0 ? `${b.unread_count} / ${b.email_count}` : `${b.email_count}`;
      const badgeClass = b.unread_count > 0 ? 'nav-badge unread' : 'nav-badge';
      const dispLabel = trBucketLabel(b);
      return `
        <div class="nav-item ${isActive ? 'active' : ''}" onclick="selectBucketNav(${b.id}, '${escapeHTML(dispLabel)}')">
          <div class="nav-item-left">
            <span class="bucket-dot" style="background:${b.color}"></span>
            <span>${escapeHTML(dispLabel)}</span>
          </div>
          <span class="${badgeClass}">${badgeText}</span>
        </div>
      `;
    })
    .join('');
}

function populateBucketSelects() {
  const activeBuckets = state.buckets.filter((b) => b.name !== 'unclassified');
  const magSelect = document.getElementById('magBucket');
  if (magSelect) {
    magSelect.innerHTML = activeBuckets
      .map((b) => `<option value="${b.id}">${escapeHTML(trBucketLabel(b))}</option>`)
      .join('');
  }
  const corpusSelect = document.getElementById('corpusBucketSelect');
  if (corpusSelect) {
    const prev = corpusSelect.value;
    corpusSelect.innerHTML = activeBuckets
      .map((b) => `<option value="${b.id}">${escapeHTML(trBucketLabel(b))} (${b.unique_words} ${t('stat_vocab_unit')})</option>`)
      .join('');
    if (prev && activeBuckets.some((b) => String(b.id) === String(prev))) {
      corpusSelect.value = prev;
    }
  }
}

function clearNavActive() {
  document.querySelectorAll('.sidebar .nav-item').forEach((el) => el.classList.remove('active'));
}

function switchMainView(viewName) {
  state.currentView = viewName;
  const views = {
    mua: 'viewMUA',
    magnets: 'viewMagnets',
    corpus: 'viewCorpus',
    buckets: 'viewBuckets',
    history: 'viewHistory',
    accounts: 'viewAccounts',
    help: 'viewHelp',
  };
  Object.entries(views).forEach(([k, id]) => {
    const el = document.getElementById(id);
    if (el) el.style.display = k === viewName ? 'flex' : 'none';
  });

  if (viewName !== 'mua') {
    clearNavActive();
    const mapNav = {
      magnets: 'navMagnets',
      corpus: 'navCorpus',
      buckets: 'navBuckets',
      history: 'navHistory',
      accounts: 'navAccounts',
      help: 'navHelp',
    };
    const navEl = document.getElementById(mapNav[viewName]);
    if (navEl) navEl.classList.add('active');
  }

  if (viewName === 'magnets') loadMagnetsView();
  if (viewName === 'corpus') loadCorpusWords();
  if (viewName === 'buckets') renderBucketsTable();
  if (viewName === 'history') loadHistoryView();
  if (viewName === 'accounts') loadAccountsList();
}

function switchHelpTab(tabKey, btnEl) {
  document.querySelectorAll('#viewHelp .help-tab-btn').forEach((b) => b.classList.remove('active'));
  if (btnEl) btnEl.classList.add('active');
  document.querySelectorAll('#viewHelp .help-section').forEach((sec) => sec.classList.remove('active'));
  const target = document.getElementById('helpSec_' + tabKey);
  if (target) target.classList.add('active');
}

function selectFolder(folder, bucketId, unreadOnly, starredOnly, el) {
  clearNavActive();
  if (el) el.classList.add('active');
  state.folder = folder;
  state.bucketId = bucketId;
  state.unreadOnly = unreadOnly;
  state.starredOnly = starredOnly;
  switchMainView('mua');
  updateCurrentListTitle();
  loadEmails();
}

function selectBucketNav(bucketId, label) {
  clearNavActive();
  state.folder = 'received';
  state.bucketId = bucketId;
  state.unreadOnly = false;
  state.starredOnly = false;
  switchMainView('mua');
  renderBucketSidebar();
  updateCurrentListTitle();
  loadEmails();
}

// ============================================================================
// Provider Classification & Separation
// ============================================================================
const PROVIDER_PALETTE = [
  '#38bdf8',
  '#10b981',
  '#a855f7',
  '#f59e0b',
  '#ec4899',
  '#14b8a6',
  '#6366f1',
];

function getEmailProviderInfo(em) {
  const uidl = String(em.uidl || '');
  const toAddr = String(em.to_address || '').trim();
  const fromAddr = String(em.from_address || '').trim();

  // 1. Check POP3 / IMAP UIDL prefix: e.g. "pop3:user@example.com:..." or "imap:user@example.jp:..."
  if (uidl.startsWith('pop3:') || uidl.startsWith('imap:')) {
    const parts = uidl.split(':');
    const proto = parts[0].toUpperCase();
    const accEmail = (parts[1] || '').toLowerCase();
    const matchedAcc = (state.accounts || []).find(
      (a) => String(a.email_address || '').toLowerCase() === accEmail
    );
    if (matchedAcc) {
      return {
        key: `acc:${matchedAcc.email_address.toLowerCase()}`,
        name: matchedAcc.name || matchedAcc.email_address,
        shortLabel: matchedAcc.name || matchedAcc.email_address.split('@')[1] || matchedAcc.email_address,
        subLabel: `${matchedAcc.email_address} (${proto})`,
        color: getProviderColor(matchedAcc.email_address),
        order: 10 + (matchedAcc.id || 0),
      };
    }
    const domain = accEmail.includes('@') ? accEmail.split('@')[1] : accEmail;
    const niceName = formatDomainProviderName(domain);
    return {
      key: `acc:${accEmail}`,
      name: niceName,
      shortLabel: niceName,
      subLabel: `${accEmail} (${proto})`,
      color: getProviderColor(accEmail),
      order: 20,
    };
  }

  // 2. Check if ToAddress or FromAddress matches a configured account
  for (const acc of state.accounts || []) {
    const accEmail = String(acc.email_address || '').toLowerCase();
    if (accEmail && (toAddr.toLowerCase().includes(accEmail) || fromAddr.toLowerCase().includes(accEmail))) {
      const proto = String(acc.protocol || 'mail').toUpperCase();
      return {
        key: `acc:${accEmail}`,
        name: acc.name || acc.email_address,
        shortLabel: acc.name || accEmail.split('@')[1] || accEmail,
        subLabel: `${acc.email_address} (${proto})`,
        color: getProviderColor(accEmail),
        order: 10 + (acc.id || 0),
      };
    }
  }

  // 3. Check imported .eml
  if (uidl.startsWith('import-')) {
    const domain = extractDomain(toAddr) || extractDomain(fromAddr) || 'imported';
    return {
      key: `import:${domain}`,
      name: `${t('provider_imported_prefix')} (${domain})`,
      shortLabel: t('provider_imported_short'),
      subLabel: toAddr || fromAddr,
      color: '#14b8a6',
      order: 80,
    };
  }

  // 4. Demo / Seed / Local Sample emails
  if (uidl.startsWith('seed-') || uidl.startsWith('sim-') || toAddr.includes('example.jp') || fromAddr.includes('local.tunbridge') || fromAddr.includes('local.popfile')) {
    return {
      key: 'provider:demo',
      name: t('provider_demo'),
      shortLabel: t('provider_demo_short'),
      subLabel: toAddr || 'me@example.jp',
      color: '#64748b',
      order: 90,
    };
  }

  // 5. Fallback by recipient domain
  const domain = extractDomain(toAddr) || extractDomain(fromAddr) || 'other';
  const niceName = formatDomainProviderName(domain);
  return {
    key: `dom:${domain}`,
    name: niceName,
    shortLabel: niceName,
    subLabel: toAddr || domain,
    color: getProviderColor(domain),
    order: 50,
  };
}

function extractDomain(addr) {
  const s = String(addr || '').toLowerCase().trim();
  const at = s.lastIndexOf('@');
  if (at === -1) return '';
  return s.slice(at + 1).replace(/[>)\s]/g, '');
}

function formatDomainProviderName(domain) {
  const d = String(domain || '').toLowerCase();
  if (d.includes('gmail.com')) return 'Gmail';
  if (d.includes('yahoo.co.jp')) return 'Yahoo!メール';
  if (d.includes('icloud.com') || d.includes('me.com')) return 'iCloud';
  if (d.includes('outlook.com') || d.includes('hotmail.com') || d.includes('live.jp')) return 'Outlook';
  if (d.includes('so-net.ne.jp') || d.includes('vaio.ne.jp')) return 'So-net';
  if (d.includes('mineo.jp')) return 'mineo';
  if (d.includes('ocn.ne.jp')) return 'OCN';
  if (d.includes('nifty.com')) return '@nifty';
  return domain || 'その他';
}

function getProviderColor(seedStr) {
  const s = String(seedStr || '');
  let h = 0;
  for (let i = 0; i < s.length; i++) {
    h = (h * 31 + s.charCodeAt(i)) >>> 0;
  }
  return PROVIDER_PALETTE[h % PROVIDER_PALETTE.length];
}

function collectProviderGroups(emails) {
  const map = new Map();
  // Pre-seed with configured accounts so user always sees their providers even when 0 emails match current filter
  for (const acc of state.accounts || []) {
    const accEmail = String(acc.email_address || '').toLowerCase();
    if (!accEmail) continue;
    const key = `acc:${accEmail}`;
    const proto = String(acc.protocol || 'pop3').toUpperCase();
    map.set(key, {
      key,
      name: acc.name || acc.email_address,
      shortLabel: acc.name || accEmail.split('@')[1] || accEmail,
      subLabel: `${acc.email_address} (${proto})`,
      color: getProviderColor(accEmail),
      order: 10 + (acc.id || 0),
      emails: [],
      unreadCount: 0,
    });
  }

  for (const em of emails) {
    const info = getEmailProviderInfo(em);
    if (!map.has(info.key)) {
      map.set(info.key, {
        ...info,
        emails: [],
        unreadCount: 0,
      });
    }
    const g = map.get(info.key);
    g.emails.push(em);
    if (!em.is_read) g.unreadCount++;
  }

  return Array.from(map.values())
    .filter((g) => g.emails.length > 0 || g.key.startsWith('acc:'))
    .sort((a, b) => a.order - b.order || a.name.localeCompare(b.name));
}

function renderProviderFilterUI() {
  const groups = collectProviderGroups(state.emails);
  const tabsEl = document.getElementById('providerFilterTabs');
  const sideEl = document.getElementById('providerNavList');

  // If current providerFilter no longer exists in groups, reset to 'all'
  if (state.providerFilter !== 'all' && !groups.some((g) => g.key === state.providerFilter)) {
    state.providerFilter = 'all';
  }

  if (tabsEl) {
    const allActive = state.providerFilter === 'all' ? 'active' : '';
    const allTab = `<div class="provider-tab ${allActive}" onclick="setProviderFilter('all')">${t('provider_all')} (${state.emails.length})</div>`;
    const groupTabs = groups
      .map((g) => {
        const isActive = state.providerFilter === g.key ? 'active' : '';
        return `
          <div class="provider-tab ${isActive}" onclick="setProviderFilter('${escapeHTML(g.key)}')" title="${escapeHTML(g.subLabel)}">
            <span class="bucket-dot" style="background:${g.color};width:7px;height:7px;"></span>
            <span>${escapeHTML(g.shortLabel)}</span>
            <span style="opacity:0.8;">(${g.emails.length})</span>
          </div>
        `;
      })
      .join('');
    tabsEl.innerHTML = allTab + groupTabs;
  }

  if (sideEl) {
    sideEl.innerHTML = groups
      .map((g) => {
        const isActive = state.currentView === 'mua' && state.providerFilter === g.key;
        const badgeText = g.unreadCount > 0 ? `${g.unreadCount} / ${g.emails.length}` : `${g.emails.length}`;
        const badgeClass = g.unreadCount > 0 ? 'nav-badge unread' : 'nav-badge';
        return `
          <div class="nav-item ${isActive ? 'active' : ''}" onclick="selectProviderSidebar('${escapeHTML(g.key)}')" title="${escapeHTML(g.subLabel)}">
            <div class="nav-item-left">
              <span class="bucket-dot" style="background:${g.color}"></span>
              <span>${escapeHTML(g.name)}</span>
            </div>
            <span class="${badgeClass}">${badgeText}</span>
          </div>
        `;
      })
      .join('');
  }
}

function setProviderFilter(providerKey) {
  state.providerFilter = providerKey;
  renderProviderFilterUI();
  renderEmailList();

  if (state.visibleEmails.length > 0 && !state.visibleEmails.some((e) => e.id === state.selectedEmailId)) {
    selectEmail(state.visibleEmails[0].id, { keepListFocus: true });
  } else if (state.visibleEmails.length === 0) {
    renderEmptyReader();
  }
  focusMailList();
}

function selectProviderSidebar(providerKey) {
  switchMainView('mua');
  if (state.providerFilter === providerKey) {
    state.providerFilter = 'all';
  } else {
    state.providerFilter = providerKey;
  }
  renderProviderFilterUI();
  renderEmailList();
  if (state.visibleEmails.length > 0 && !state.visibleEmails.some((e) => e.id === state.selectedEmailId)) {
    selectEmail(state.visibleEmails[0].id, { keepListFocus: true });
  }
  focusMailList();
}

function toggleGroupByProvider(checked) {
  state.groupByProvider = Boolean(checked);
  try {
    localStorage.setItem('tunbridge_group_by_provider', state.groupByProvider ? '1' : '0');
  } catch (e) {}
  renderEmailList();
  focusMailList();
}

function toggleProviderSectionCollapse(providerKey) {
  state.collapsedProviders[providerKey] = !state.collapsedProviders[providerKey];
  renderEmailList();
  focusMailList();
}

function renderEmptyReader() {
  document.getElementById('mailReaderPane').innerHTML = `
    <div class="empty-reader">
      <div style="font-size:34px;">📭</div>
      <div>${t('empty_reader_none')}</div>
    </div>
  `;
}

async function loadEmails(options = {}) {
  const { preferredEmailId = 0, keepListFocus = false } = options;
  try {
    const search = document.getElementById('mailSearchInput').value.trim();
    const params = new URLSearchParams({
      folder: state.folder,
      bucket_id: String(state.bucketId),
      unread: state.unreadOnly ? '1' : '0',
      starred: state.starredOnly ? '1' : '0',
      search: search,
    });
    const data = await apiGet('/api/emails?' + params.toString());
    state.emails = data.emails || [];
    renderProviderFilterUI();
    renderEmailList();

    const targetId = preferredEmailId || state.selectedEmailId;
    if (state.visibleEmails.length > 0) {
      if (targetId && state.visibleEmails.some((e) => e.id === targetId)) {
        await selectEmail(targetId, { keepListFocus });
      } else {
        await selectEmail(state.visibleEmails[0].id, { keepListFocus });
      }
    } else {
      state.selectedEmailId = 0;
      state.selectedDetail = null;
      renderEmptyReader();
    }
  } catch (e) {
    showToast(e.message, '#ef4444');
  }
}

function renderSingleMailRowHTML(e, providerInfo) {
  const isSelected = e.id === state.selectedEmailId;
  const sender = e.from_name ? `${e.from_name} <${e.from_address}>` : e.from_address;
  const confPct = Math.round((e.confidence || 0) * 100);
  const starIcon = e.is_starred ? '⭐' : '☆';
  const magnetBadge = e.magnet_id > 0 ? `<span title="${t('magnet_applied_prefix')}${escapeHTML(e.magnet_desc)}">🧲</span>` : '';
  const trainedBadge = e.is_reclassified
    ? `<span style="color:#f59e0b;" title="${t('trained_toe_title')}">${t('trained_toe_badge')}</span>`
    : e.is_trained
    ? `<span style="color:#10b981;" title="${t('trained_corpus_title')}">${t('trained_corpus_badge')}</span>`
    : '';

  const providerPill = !state.groupByProvider
    ? `<span class="provider-badge-pill" style="background:${providerInfo.color}22;color:${providerInfo.color};border-color:${providerInfo.color}55;" title="${escapeHTML(providerInfo.subLabel)}">${escapeHTML(providerInfo.shortLabel)}</span>`
    : '';

  const attachBadge =
    e.attachment_count > 0
      ? `<span style="color:#93c5fd;font-weight:600;" title="${t('attachment_count_title', { count: e.attachment_count })}">📎${e.attachment_count}</span>`
      : '';

  const dispBucketLabel = trBucketLabel(e.bucket_name || e.bucket_label, e.bucket_label);

  return `
    <div class="mail-row ${isSelected ? 'selected' : ''} ${!e.is_read ? 'unread' : ''}" data-email-id="${e.id}" onclick="selectEmail(${e.id}, { keepListFocus: true })">
      <div class="mail-row-top">
        <span class="mail-sender">${escapeHTML(sender)}</span>
        <span class="mail-date">${escapeHTML(e.date.slice(5, 16))}</span>
      </div>
      <div class="mail-subject">${escapeHTML(e.subject)}</div>
      <div class="mail-row-meta">
        <div style="display:flex;align-items:center;gap:5px;flex-wrap:wrap;">
          <span class="bucket-pill" style="background:${e.bucket_color}">
            ${magnetBadge} ${escapeHTML(dispBucketLabel)} (${confPct}%)
          </span>
          ${providerPill}
        </div>
        <div class="meta-flags">
          ${attachBadge}
          ${trainedBadge}
          <span style="cursor:pointer;font-size:14px;" onclick="toggleStar(event, ${e.id}, ${!e.is_starred})">${starIcon}</span>
        </div>
      </div>
    </div>
  `;
}

function renderEmailList() {
  const container = document.getElementById('mailListContainer');
  const allGroups = collectProviderGroups(state.emails);
  const filteredGroups =
    state.providerFilter === 'all'
      ? allGroups.filter((g) => g.emails.length > 0)
      : allGroups.filter((g) => g.key === state.providerFilter && g.emails.length > 0);

  // Build ordered visibleEmails list so keyboard ↑/↓/Home/End/PgUp/PgDn/Delete matches exact visual order
  const ordered = [];
  let html = '';

  if (state.groupByProvider) {
    for (const g of filteredGroups) {
      const isCollapsed = Boolean(state.collapsedProviders[g.key]);
      const arrow = isCollapsed ? '▶' : '▼';
      const unreadText = g.unreadCount > 0 ? `${t('unread_prefix')}${g.unreadCount}` : '';
      html += `
        <div class="provider-section-header" onclick="toggleProviderSectionCollapse('${escapeHTML(g.key)}')" style="border-left:3px solid ${g.color};">
          <div style="display:flex;align-items:center;gap:6px;overflow:hidden;">
            <span style="font-size:10px;color:var(--text-muted);">${arrow}</span>
            <span class="bucket-dot" style="background:${g.color};"></span>
            <span>${escapeHTML(g.name)}</span>
            <span style="font-weight:400;font-size:11px;color:var(--text-muted);overflow:hidden;text-overflow:ellipsis;white-space:nowrap;">${escapeHTML(g.subLabel)}</span>
          </div>
          <span class="nav-badge ${g.unreadCount > 0 ? 'unread' : ''}" style="flex-shrink:0;">${g.emails.length} ${t('count_suffix')}${unreadText}</span>
        </div>
      `;
      if (!isCollapsed) {
        for (const em of g.emails) {
          ordered.push(em);
          html += renderSingleMailRowHTML(em, g);
        }
      }
    }
  } else {
    const flatEmails =
      state.providerFilter === 'all'
        ? state.emails
        : state.emails.filter((e) => getEmailProviderInfo(e).key === state.providerFilter);
    for (const em of flatEmails) {
      const pInfo = getEmailProviderInfo(em);
      ordered.push(em);
      html += renderSingleMailRowHTML(em, pInfo);
    }
  }

  state.visibleEmails = ordered;
  document.getElementById('currentListCount').textContent = `${ordered.length} ${t('count_suffix')}`;

  if (ordered.length === 0 && filteredGroups.length === 0) {
    container.innerHTML = `<div style="padding:28px;text-align:center;color:var(--text-muted);">${t('empty_list_none')}</div>`;
    return;
  }

  container.innerHTML = html;
  scrollSelectedMailRowIntoView();
}

function scrollSelectedMailRowIntoView() {
  const container = document.getElementById('mailListContainer');
  if (!container || !state.selectedEmailId) return;
  const row = container.querySelector(`.mail-row[data-email-id="${state.selectedEmailId}"]`);
  if (row) {
    row.scrollIntoView({ block: 'nearest' });
  }
}

function focusMailList() {
  const container = document.getElementById('mailListContainer');
  if (container && state.currentView === 'mua') {
    container.focus({ preventScroll: true });
  }
}

let selectRequestSeq = 0;

async function selectEmail(id, options = {}) {
  const { keepListFocus = false } = options;
  if (state.selectedEmailId !== id) {
    state.selectedAttachmentId = 0;
  }
  state.selectedEmailId = id;
  const reqId = ++selectRequestSeq;

  // Optimistically mark as read & highlight immediately for snappy keyboard navigation
  const item = state.emails.find((x) => x.id === id);
  if (item && !item.is_read) {
    item.is_read = true;
  }
  renderEmailList();
  if (keepListFocus) {
    focusMailList();
  }

  try {
    const data = await apiGet(`/api/emails/detail?id=${id}`);
    if (reqId !== selectRequestSeq) return; // Superseded by a newer selection
    state.selectedDetail = data;
    refreshOverview();
    renderProviderFilterUI();
    renderMailReader();
    if (keepListFocus) {
      focusMailList();
    }
  } catch (e) {
    showToast(e.message, '#ef4444');
  }
}

// ============================================================================
// Window Size & Position Persistence
// ============================================================================
let saveWindowStateTimer = null;

function saveCurrentWindowState() {
  if (saveWindowStateTimer) clearTimeout(saveWindowStateTimer);
  saveWindowStateTimer = setTimeout(() => {
    try {
      const isMaximized =
        window.outerWidth >= (window.screen.availWidth - 16) &&
        window.outerHeight >= (window.screen.availHeight - 16);

      const winState = {
        x: window.screenX,
        y: window.screenY,
        width: window.outerWidth,
        height: window.outerHeight,
        maximized: isMaximized,
      };

      if (winState.width >= 600 && winState.height >= 450) {
        fetch('/api/window/state', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(winState),
          keepalive: true,
        }).catch(() => {});
      }
    } catch (e) {
      // ignore
    }
  }, 400);
}

function initWindowStateListener() {
  window.addEventListener('resize', saveCurrentWindowState);
  window.addEventListener('beforeunload', () => {
    try {
      const isMaximized =
        window.outerWidth >= (window.screen.availWidth - 16) &&
        window.outerHeight >= (window.screen.availHeight - 16);
      const winState = {
        x: window.screenX,
        y: window.screenY,
        width: window.outerWidth,
        height: window.outerHeight,
        maximized: isMaximized,
      };
      if (winState.width >= 600 && winState.height >= 450) {
        if (navigator.sendBeacon) {
          navigator.sendBeacon('/api/window/state', JSON.stringify(winState));
        }
      }
    } catch (e) {}
  });

  let lastX = window.screenX;
  let lastY = window.screenY;
  setInterval(() => {
    if (window.screenX !== lastX || window.screenY !== lastY) {
      lastX = window.screenX;
      lastY = window.screenY;
      saveCurrentWindowState();
    }
  }, 2000);
}

// ============================================================================
// Mail List & Reader Pane Resizer (Draggable Splitter)
// ============================================================================
function initMailPaneResizer() {
  const resizer = document.getElementById('mailPaneResizer');
  const listPane = document.getElementById('mailListPane');
  if (!resizer || !listPane) return;

  // Restore saved width from localStorage
  try {
    const saved = localStorage.getItem('tunbridge_mail_list_width');
    if (saved) {
      const w = parseInt(saved, 10);
      if (w >= 260 && w <= 950) {
        listPane.style.width = `${w}px`;
      }
    }
  } catch (e) {
    // ignore localStorage errors
  }

  let isDragging = false;
  let startX = 0;
  let startWidth = 0;

  resizer.addEventListener('mousedown', (e) => {
    isDragging = true;
    startX = e.clientX;
    startWidth = listPane.getBoundingClientRect().width;
    resizer.classList.add('resizing');
    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';
    e.preventDefault();
  });

  window.addEventListener('mousemove', (e) => {
    if (!isDragging) return;
    const deltaX = e.clientX - startX;
    let newWidth = startWidth + deltaX;
    if (newWidth < 260) newWidth = 260;
    const maxWidth = Math.max(380, window.innerWidth - 420);
    if (newWidth > maxWidth) newWidth = maxWidth;
    listPane.style.width = `${newWidth}px`;
  });

  window.addEventListener('mouseup', () => {
    if (isDragging) {
      isDragging = false;
      resizer.classList.remove('resizing');
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
      try {
        const finalWidth = Math.round(listPane.getBoundingClientRect().width);
        localStorage.setItem('tunbridge_mail_list_width', String(finalWidth));
      } catch (e) {}
    }
  });

  // Double click to reset to default 400px width
  resizer.addEventListener('dblclick', () => {
    listPane.style.width = '400px';
    try {
      localStorage.setItem('tunbridge_mail_list_width', '400');
    } catch (e) {}
  });
}

// ============================================================================
// Keyboard Navigation (Up/Down/Home/End/PgUp/PgDn/Delete/Ctrl+Z) & Safe Undo
// ============================================================================
function setupKeyboardNavigation() {
  document.addEventListener('keydown', async (ev) => {
    if (state.currentView !== 'mua') return;

    // Ignore when any modal dialog is open
    if (document.querySelector('.modal-backdrop.open')) return;

    const active = document.activeElement;
    const isEditableInput =
      active &&
      (active.tagName === 'INPUT' ||
        active.tagName === 'SELECT' ||
        active.isContentEditable ||
        (active.tagName === 'TEXTAREA' && active.id !== 'emailBodyTextarea'));

    if (isEditableInput) return;

    // Ctrl+Z / Cmd+Z: Undo last delete
    if ((ev.ctrlKey || ev.metaKey) && !ev.shiftKey && !ev.altKey && ev.key.toLowerCase() === 'z') {
      if (state.undoStack.length > 0) {
        ev.preventDefault();
        await undoLastDelete();
      }
      return;
    }

    // If user is focused inside the read-only body textarea and presses navigation keys,
    // allow mail list navigation unless they are holding Shift to select text
    if (ev.shiftKey || ev.ctrlKey || ev.altKey || ev.metaKey) return;

    const list = state.visibleEmails || [];
    const curIdx = list.findIndex((e) => e.id === state.selectedEmailId);

    switch (ev.key) {
      case 'ArrowDown': {
        ev.preventDefault();
        if (list.length === 0) return;
        const nextIdx = curIdx < 0 ? 0 : Math.min(list.length - 1, curIdx + 1);
        if (list[nextIdx]) {
          await selectEmail(list[nextIdx].id, { keepListFocus: true });
        }
        break;
      }
      case 'ArrowUp': {
        ev.preventDefault();
        if (list.length === 0) return;
        const prevIdx = curIdx <= 0 ? 0 : curIdx - 1;
        if (list[prevIdx]) {
          await selectEmail(list[prevIdx].id, { keepListFocus: true });
        }
        break;
      }
      case 'Home': {
        ev.preventDefault();
        if (list.length === 0) return;
        await selectEmail(list[0].id, { keepListFocus: true });
        break;
      }
      case 'End': {
        ev.preventDefault();
        if (list.length === 0) return;
        await selectEmail(list[list.length - 1].id, { keepListFocus: true });
        break;
      }
      case 'PageDown': {
        ev.preventDefault();
        if (list.length === 0) return;
        const step = getMailListPageStep();
        const nextIdx = curIdx < 0 ? 0 : Math.min(list.length - 1, curIdx + step);
        if (list[nextIdx]) {
          await selectEmail(list[nextIdx].id, { keepListFocus: true });
        }
        break;
      }
      case 'PageUp': {
        ev.preventDefault();
        if (list.length === 0) return;
        const step = getMailListPageStep();
        const prevIdx = curIdx < 0 ? 0 : Math.max(0, curIdx - step);
        if (list[prevIdx]) {
          await selectEmail(list[prevIdx].id, { keepListFocus: true });
        }
        break;
      }
      case 'Delete': {
        ev.preventDefault();
        if (!state.selectedEmailId || list.length === 0) return;
        await deleteEmailsWithUndo([state.selectedEmailId]);
        break;
      }
    }
  });
}

function getMailListPageStep() {
  const container = document.getElementById('mailListContainer');
  if (!container) return 6;
  const firstRow = container.querySelector('.mail-row');
  const rowH = firstRow ? firstRow.offsetHeight || 76 : 76;
  const visibleRows = Math.floor((container.clientHeight || 480) / rowH);
  return Math.max(3, visibleRows - 1);
}

function updateUndoButtonUI() {
  const btn = document.getElementById('undoDeleteBtn');
  if (!btn) return;
  if (state.undoStack.length > 0) {
    btn.style.display = 'inline-flex';
    btn.textContent = `${t('undo_btn_label')} (${state.undoStack.length})`;
  } else {
    btn.style.display = 'none';
  }
}

async function deleteEmailsWithUndo(ids) {
  if (!ids || ids.length === 0) return;
  const list = state.visibleEmails || [];
  const firstTargetIdx = list.findIndex((e) => ids.includes(e.id));

  // Determine the next email (below, or above if at the bottom) to receive focus after deletion
  let nextFocusId = 0;
  if (firstTargetIdx >= 0) {
    for (let i = firstTargetIdx + 1; i < list.length; i++) {
      if (!ids.includes(list[i].id)) {
        nextFocusId = list[i].id;
        break;
      }
    }
    if (!nextFocusId) {
      for (let i = firstTargetIdx - 1; i >= 0; i--) {
        if (!ids.includes(list[i].id)) {
          nextFocusId = list[i].id;
          break;
        }
      }
    }
  }

  // Capture snapshots for Undo before deleting
  const snapshots = state.emails.filter((e) => ids.includes(e.id)).map((e) => ({ ...e }));
  if (snapshots.length === 0) return;

  const isPermanent = state.folder === 'trash' || snapshots.every((s) => s.folder === 'trash');
  const action = isPermanent ? 'delete' : 'trash';
  const prevFolder = snapshots[0].folder || 'received';

  try {
    const markRes = await apiPost('/api/emails/mark', { email_ids: ids, action });
    const fullSnapshots =
      markRes && Array.isArray(markRes.deleted_snapshots) && markRes.deleted_snapshots.length > 0
        ? markRes.deleted_snapshots
        : snapshots;

    state.undoStack.push({
      action,
      ids: [...ids],
      prevFolder,
      snapshots: fullSnapshots,
    });
    updateUndoButtonUI();

    const subjPreview =
      snapshots.length === 1
        ? `「${snapshots[0].subject.slice(0, 22)}${snapshots[0].subject.length > 22 ? '…' : ''}」`
        : `${snapshots.length} 件のメール`;
    const msg = isPermanent ? `${subjPreview}を完全削除しました` : `${subjPreview}をゴミ箱へ移動しました`;

    showToast(msg, '#f59e0b', t('undo_toast_action'), () => {
      undoLastDelete();
    });

    await refreshOverview();
    await loadEmails({ preferredEmailId: nextFocusId, keepListFocus: true });
    focusMailList();
  } catch (e) {
    showToast(e.message, '#ef4444');
  }
}

async function undoLastDelete() {
  if (state.undoStack.length === 0) return;
  const entry = state.undoStack.pop();
  updateUndoButtonUI();

  try {
    if (entry.action === 'delete') {
      // Re-insert permanently deleted email snapshots with original IDs
      await apiPost('/api/emails/mark', {
        email_ids: [],
        action: 'restore',
        restore_snapshots: entry.snapshots,
      });
    } else {
      // Restore from trash back to previous folder
      await apiPost('/api/emails/mark', {
        email_ids: entry.ids,
        action: 'restore',
        restore_folder: entry.prevFolder || 'received',
      });
    }

    const restoredId = entry.ids && entry.ids.length > 0 ? entry.ids[0] : 0;
    showToast(t('undo_restored_toast'), '#10b981');
    await refreshOverview();
    await loadEmails({ preferredEmailId: restoredId, keepListFocus: true });
    focusMailList();
  } catch (e) {
    showToast(t('undo_failed_toast') + e.message, '#ef4444');
  }
}

function setReaderTab(tab) {
  state.readerTab = tab;
  renderMailReader();
}

function parseEmailHeaders(em, confPct) {
  const map = new Map(); // lowercase key -> { name, value }
  const order = [];

  const setHeader = (name, val, overwrite = true) => {
    const low = name.toLowerCase().trim();
    if (!low) return;
    if (!map.has(low)) {
      order.push(low);
      map.set(low, { name: name.trim(), value: String(val || '').trim() });
    } else if (overwrite) {
      map.get(low).value = String(val || '').trim();
    }
  };

  if (em.raw_headers) {
    const lines = em.raw_headers.split(/\r?\n/);
    let lastKey = '';
    for (const line of lines) {
      if (!line.trim()) continue;
      if (/^[ \t]+/.test(line) && lastKey && map.has(lastKey)) {
        map.get(lastKey).value += ' ' + line.trim();
        continue;
      }
      const colonIdx = line.indexOf(':');
      if (colonIdx > 0) {
        const k = line.slice(0, colonIdx).trim();
        const v = line.slice(colonIdx + 1).trim();
        lastKey = k.toLowerCase();
        setHeader(k, v, true);
      }
    }
  }

  // Ensure standard & classification headers have fallback values from Email object
  const formattedFrom = em.from_name ? `${em.from_name} <${em.from_address}>` : em.from_address;
  setHeader('From', formattedFrom, true);
  setHeader('To', em.to_address || '', false);
  setHeader('Cc', map.has('cc') ? map.get('cc').value : '', false);
  setHeader('Bcc', map.has('bcc') ? map.get('bcc').value : '', false);
  setHeader('Subject', em.subject || '', true);
  setHeader('Date', em.date || '', false);
  if (em.message_id) {
    setHeader('Message-ID', `<${em.message_id.replace(/^<|>$/g, '')}>`, false);
  }
  setHeader('X-Text-Classification', em.bucket_name || '', true);
  setHeader('X-POPFile-Confidence', `${confPct}% (${(em.confidence || 0).toFixed(4)})`, true);
  setHeader('MIME-Version', '1.0', false);
  setHeader('Content-Type', em.body_html ? 'text/html; charset=UTF-8' : 'text/plain; charset=UTF-8', false);

  return { map, order };
}

function toggleOtherHeaders() {
  state.showAllOtherHeaders = !state.showAllOtherHeaders;
  renderMailReader();
}

function openHeaderConfigModal() {
  renderHeaderConfigModalContents();
  document.getElementById('headerConfigModal').classList.add('open');
}

function savePinnedHeaders() {
  try {
    localStorage.setItem('tunbridge_pinned_headers', JSON.stringify(state.pinnedHeaders));
  } catch (e) {}
}

function renderHeaderConfigModalContents() {
  const checklistEl = document.getElementById('presetHeadersChecklist');
  const pinnedLower = new Set(state.pinnedHeaders.map((h) => h.toLowerCase()));
  const isEn = typeof getCurrentLang === 'function' && getCurrentLang() === 'en';

  checklistEl.innerHTML = PRESET_HEADER_OPTIONS.map((opt) => {
    const checked = pinnedLower.has(opt.name.toLowerCase()) ? 'checked' : '';
    const lbl = isEn && opt.labelEn ? opt.labelEn : opt.label;
    return `
      <label style="display:flex;align-items:center;gap:7px;cursor:pointer;font-size:12.5px;color:var(--text-main);">
        <input type="checkbox" ${checked} onchange="togglePresetPinnedHeader('${escapeHTML(opt.name)}', this.checked)" />
        <span>${escapeHTML(lbl)}</span>
      </label>
    `;
  }).join('');

  const chipsEl = document.getElementById('activePinnedHeadersChips');
  if (state.pinnedHeaders.length === 0) {
    chipsEl.innerHTML = `<span style="font-size:12px;color:var(--text-muted);">${isEn ? 'No additional pinned headers selected' : '常時表示する追加ヘッダは選択されていません'}</span>`;
  } else {
    chipsEl.innerHTML = state.pinnedHeaders
      .map(
        (h) => `
        <span class="header-chip">
          <span style="font-family:monospace;color:#93c5fd;">${escapeHTML(h)}</span>
          <span style="cursor:pointer;color:#fca5a5;font-weight:700;" title="${isEn ? 'Remove' : '削除'}" onclick="removePinnedHeader('${escapeHTML(h)}')">✕</span>
        </span>
      `
      )
      .join('');
  }
}

function togglePresetPinnedHeader(headerName, enabled) {
  const low = headerName.toLowerCase();
  if (enabled) {
    if (!state.pinnedHeaders.some((h) => h.toLowerCase() === low)) {
      state.pinnedHeaders.push(headerName);
    }
  } else {
    state.pinnedHeaders = state.pinnedHeaders.filter((h) => h.toLowerCase() !== low);
  }
  savePinnedHeaders();
  renderHeaderConfigModalContents();
  renderMailReader();
}

function addCustomPinnedHeader() {
  const input = document.getElementById('customHeaderInput');
  const raw = input.value.trim().replace(/:$/, '');
  if (!raw) return;
  const isEn = typeof getCurrentLang === 'function' && getCurrentLang() === 'en';
  const core5 = ['from', 'to', 'cc', 'bcc', 'subject'];
  if (core5.includes(raw.toLowerCase())) {
    showToast(isEn ? `${raw} is always displayed by default` : `${raw} は標準で常時表示されています`, '#f59e0b');
    input.value = '';
    return;
  }
  if (!state.pinnedHeaders.some((h) => h.toLowerCase() === raw.toLowerCase())) {
    state.pinnedHeaders.push(raw);
    savePinnedHeaders();
    renderHeaderConfigModalContents();
    renderMailReader();
    showToast(isEn ? `Added header "${raw}" to pinned headers` : `ヘッダ「${raw}」を常時表示に追加しました`, '#10b981');
  }
  input.value = '';
}

function removePinnedHeader(headerName) {
  const low = headerName.toLowerCase();
  state.pinnedHeaders = state.pinnedHeaders.filter((h) => h.toLowerCase() !== low);
  savePinnedHeaders();
  renderHeaderConfigModalContents();
  renderMailReader();
}

function resetPinnedHeadersDefault() {
  state.pinnedHeaders = [...DEFAULT_PINNED_HEADERS];
  savePinnedHeaders();
  renderHeaderConfigModalContents();
  renderMailReader();
  const isEn = typeof getCurrentLang === 'function' && getCurrentLang() === 'en';
  showToast(isEn ? 'Pinned headers reset to default' : '常時表示ヘッダを標準設定に戻しました');
}

function renderMailReader() {
  const pane = document.getElementById('mailReaderPane');
  if (!state.selectedDetail || !state.selectedDetail.email) return;

  const { email: em, explanation: exp } = state.selectedDetail;
  const activeBuckets = state.buckets.filter((b) => b.name !== 'unclassified');
  const confPct = ((em.confidence || 0) * 100).toFixed(2);

  // Parse all headers
  const parsedHeaders = parseEmailHeaders(em, confPct);
  const coreHeaderNames = ['From', 'To', 'Cc', 'Bcc', 'Subject'];
  const displayedKeys = new Set(coreHeaderNames.map((k) => k.toLowerCase()));

  // Build Core 5 Header rows (From / To / Cc / Bcc / Subject)
  const coreRowsHTML = coreHeaderNames
    .map((hName) => {
      const entry = parsedHeaders.map.get(hName.toLowerCase());
      const val = entry ? entry.value : '';
      const isSubject = hName === 'Subject';
      const valHTML = val
        ? `<div class="header-detail-val ${isSubject ? 'subject-val' : ''}">${escapeHTML(val)}</div>`
        : `<div class="header-detail-val empty-val">${t('header_empty')}</div>`;
      return `
        <div class="header-detail-key">${escapeHTML(hName)}:</div>
        ${valHTML}
      `;
    })
    .join('');

  // Build User-Customizable Always-Shown Header rows
  const pinnedRowsHTML = state.pinnedHeaders
    .map((hName) => {
      const low = hName.toLowerCase();
      if (displayedKeys.has(low)) return '';
      displayedKeys.add(low);
      const entry = parsedHeaders.map.get(low);
      const val = entry ? entry.value : '';
      const valHTML = val
        ? `<div class="header-detail-val">${escapeHTML(val)}</div>`
        : `<div class="header-detail-val empty-val">${t('header_unset')}</div>`;
      return `
        <div class="header-detail-key custom-pinned" title="${escapeHTML(hName)}">${escapeHTML(hName)}:</div>
        ${valHTML}
      `;
    })
    .join('');

  // Collect remaining "Other" headers that are in the email but not in Core 5 or Pinned
  const otherHeaders = [];
  for (const lowKey of parsedHeaders.order) {
    if (!displayedKeys.has(lowKey)) {
      const item = parsedHeaders.map.get(lowKey);
      if (item && item.value !== '') {
        otherHeaders.push(item);
      }
    }
  }

  const otherRowsHTML =
    state.showAllOtherHeaders && otherHeaders.length > 0
      ? `
        <div class="extra-headers-box">
          <div class="header-detail-table">
            ${otherHeaders
              .map(
                (item) => `
                  <div class="header-detail-key">${escapeHTML(item.name)}:</div>
                  <div class="header-detail-val" style="color:var(--text-sub);">${escapeHTML(item.value)}</div>
                `
              )
              .join('')}
          </div>
        </div>
      `
      : '';

  // Reclassification buttons (TOE Bar)
  const reclassBtns = activeBuckets
    .map((b) => {
      const isCurrent = b.id === em.bucket_id;
      const dispLabel = trBucketLabel(b);
      return `
        <button class="reclass-btn ${isCurrent ? 'current' : ''}" 
                style="border-color:${b.color};"
                onclick="reclassifyCurrentEmail(${b.id}, '${escapeHTML(dispLabel)}')">
          <span class="bucket-dot" style="background:${b.color}"></span>
          <span>${escapeHTML(dispLabel)}</span>
          ${isCurrent ? '✓' : ''}
        </button>
      `;
    })
    .join('');

  const magnetNotice =
    exp && exp.magnet_matched
      ? `<span style="background:rgba(245,158,11,0.2);color:#fcd34d;padding:3px 9px;border-radius:999px;font-size:11.5px;">
           ${t('magnet_matched_badge', { field: escapeHTML(exp.magnet_matched.field), pattern: escapeHTML(exp.magnet_matched.pattern) })}
         </span>`
      : '';

  let tabContentHTML = '';
  const isBodyTab = state.readerTab === 'body';
  if (isBodyTab) {
    tabContentHTML = `<textarea id="emailBodyTextarea" class="email-body-textarea" readonly></textarea>`;
  } else if (state.readerTab === 'colored') {
    tabContentHTML = renderColoredMessageView(em, exp);
  } else if (state.readerTab === 'matrix') {
    tabContentHTML = renderWordScoreMatrix(em, exp);
  } else if (state.readerTab === 'headers') {
    tabContentHTML = `
      <div class="card" style="margin-bottom:12px;">
        <div style="font-weight:700;margin-bottom:6px;">${t('raw_rfc822_title')}</div>
        <div style="font-family:var(--header-font-family);color:#38bdf8;font-size:var(--header-font-size);">
          X-Text-Classification: ${escapeHTML(em.bucket_name)}<br/>
          X-POPFile-Confidence: ${confPct}%
        </div>
      </div>
      <div class="email-plain-body">${escapeHTML(em.raw_headers)}\n\n${escapeHTML(em.body_text)}</div>
    `;
  }

  const emBucketLabel = trBucketLabel(em.bucket_name || em.bucket_label, em.bucket_label);

  pane.innerHTML = `
    <div class="reader-header">
      <div class="reader-top-toolbar">
        <div style="display:flex;align-items:center;gap:6px;flex-wrap:wrap;">
          <button class="btn-secondary" onclick="openHeaderConfigModal()" title="${t('header_config_btn_title')}">
            ${t('header_config_btn', { count: state.pinnedHeaders.length })}
          </button>
          <button class="btn-secondary" onclick="openFontConfigModal()" title="${t('font_config_btn_title')}">
            ${t('font_config_btn')}
          </button>
          ${
            otherHeaders.length > 0
              ? `<button class="btn-secondary" onclick="toggleOtherHeaders()">
                   ${state.showAllOtherHeaders ? t('other_headers_hide') : t('other_headers_show', { count: otherHeaders.length })}
                 </button>`
              : ''
          }
        </div>
        <div class="reader-actions">
          <button class="btn-secondary" onclick="replyCurrentEmail()" title="${t('btn_reply_sender_title')}">${t('btn_reply_sender')}</button>
          <button class="btn-secondary" onclick="replyAllCurrentEmail()" title="${t('btn_reply_all_title')}">${t('btn_reply_all')}</button>
          <button class="btn-secondary" onclick="forwardCurrentEmail()" title="${t('btn_forward_title')}">${t('btn_forward')}</button>
          <button class="btn-secondary" onclick="createMagnetFromEmail()">${t('btn_register_magnet')}</button>
          <button class="btn-secondary" onclick="window.open('/api/emails/export?id=${em.id}', '_blank')">${t('btn_export_eml')}</button>
          ${
            em.folder === 'trash'
              ? `<button class="btn-danger" onclick="markEmails([${em.id}], 'delete')">${t('btn_delete_perm')}</button>`
              : `<button class="btn-danger" onclick="markEmails([${em.id}], 'trash')">${t('btn_delete_trash')}</button>`
          }
        </div>
      </div>

      <!-- Structured Mail Header Table: From / To / Cc / Bcc / Subject + Custom Pinned Headers -->
      <div class="header-detail-table">
        ${coreRowsHTML}
        ${pinnedRowsHTML}
      </div>

      ${otherRowsHTML}
    </div>

    <!-- Classification & TOE Bar -->
    <div class="popfile-bar">
      <div class="popfile-bar-top">
        <div class="popfile-status-info">
          <span><strong>${t('classification_judgment')}</strong></span>
          <span class="bucket-pill" style="background:${em.bucket_color};font-size:12px;padding:3px 10px;">
            ${escapeHTML(emBucketLabel)} (${confPct}%)
          </span>
          ${magnetNotice}
          ${
            em.is_reclassified
              ? `<span style="color:#fcd34d;font-size:12px;">${t('toe_reclassified_note')}</span>`
              : `<span style="color:var(--text-muted);font-size:12px;">${t('toe_hint_note')}</span>`
          }
        </div>
      </div>
      <div class="popfile-bucket-buttons">
        <span style="font-size:11.5px;color:var(--text-muted);margin-right:4px;">${t('bucket_change_label')}</span>
        ${reclassBtns}
      </div>
    </div>

    <!-- Reader View Switcher Tabs -->
    <div class="reader-tabs">
      <div class="reader-tab ${state.readerTab === 'body' ? 'active' : ''}" onclick="setReaderTab('body')">
        ${t('tab_body')}
      </div>
      <div class="reader-tab ${state.readerTab === 'colored' ? 'active' : ''}" onclick="setReaderTab('colored')">
        ${t('tab_colored')}
      </div>
      <div class="reader-tab ${state.readerTab === 'matrix' ? 'active' : ''}" onclick="setReaderTab('matrix')">
        ${t('tab_matrix', { count: exp && exp.top_words ? exp.top_words.length : 0 })}
      </div>
      <div class="reader-tab ${state.readerTab === 'headers' ? 'active' : ''}" onclick="setReaderTab('headers')">
        ${t('tab_headers')}
      </div>
    </div>

    <!-- Split Workspace: Left (Body + Bottom Attachment Medium Icons) | Right (Attachment Preview Pane) -->
    <div class="reader-workspace-split">
      <div class="reader-main-column">
        <div class="reader-body-container ${isBodyTab ? 'textarea-mode' : ''}">
          ${tabContentHTML}
        </div>
        ${renderBottomAttachmentsPane(em.attachments)}
      </div>
      ${renderRightAttachmentPreviewPane(em.attachments)}
    </div>
  `;

  if (isBodyTab) {
    const ta = document.getElementById('emailBodyTextarea');
    if (ta) {
      ta.value = em.body_text || em.body_html || '';
    }
  }
}

// Colored Message View: Highlights recognized corpus tokens in subject & body with their strongest bucket's color!
function renderColoredMessageView(em, exp) {
  const words = (exp && exp.top_words) || [];
  // Build lookup of raw words (without prefix like subject:)
  const wordMap = new Map();
  for (const w of words) {
    let raw = w.word;
    if (raw.includes(':')) {
      const parts = raw.split(':');
      raw = parts[parts.length - 1];
    }
    if (raw.length >= 2 && !wordMap.has(raw.toLowerCase())) {
      wordMap.set(raw.toLowerCase(), w);
    }
  }

  const highlightText = (text) => {
    let safe = escapeHTML(text);
    // Sort keys by length descending to avoid partial nested replacements
    const sortedKeys = Array.from(wordMap.keys()).sort((a, b) => b.length - a.length);
    if (sortedKeys.length === 0) return safe;

    // Replace using placeholder tokens so we never corrupt HTML tags
    const placeholders = [];
    for (const key of sortedKeys.slice(0, 60)) {
      const info = wordMap.get(key);
      const escapedKey = key.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
      const regex = new RegExp(`(${escapedKey})`, 'gi');
      safe = safe.replace(regex, (match) => {
        const idx = placeholders.length;
        const pct = (info.max_prob * 100).toFixed(1);
        const dispBest = trBucketLabel(info.best_bucket || info.best_label, info.best_label);
        placeholders.push(
          `<span class="colored-token" style="background:${info.best_color}33;color:${info.best_color};" title="${t('colored_token_title', { label: escapeHTML(dispBest), pct, freq: info.frequency_in_msg })}">${match}</span>`
        );
        return `\x00TOK_${idx}_\x00`;
      });
    }
    return safe.replace(/\x00TOK_(\d+)_\x00/g, (_, idx) => placeholders[Number(idx)]);
  };

  const legend = state.buckets
    .filter((b) => b.name !== 'unclassified')
    .map(
      (b) =>
        `<span class="bucket-pill" style="background:${b.color}33;color:${b.color};border:1px solid ${b.color};">${escapeHTML(trBucketLabel(b))}</span>`
    )
    .join(' ');

  return `
    <div class="card" style="margin-bottom:14px;">
      <div style="display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:8px;">
        <div style="font-size:12.5px;color:var(--text-muted);">
          ${t('colored_view_desc')}
        </div>
        <div style="display:flex;gap:6px;flex-wrap:wrap;">${legend}</div>
      </div>
    </div>
    <div class="card" style="margin-bottom:12px;">
      <div style="font-size:12px;color:var(--text-muted);margin-bottom:4px;">${t('colored_subject_label')}</div>
      <div style="font-size:15px;font-weight:700;line-height:1.5;">${highlightText(em.subject)}</div>
    </div>
    <div class="email-plain-body">${highlightText(em.body_text || em.body_html)}</div>
  `;
}

// Word Score Matrix & Posterior Probabilities
function renderWordScoreMatrix(em, exp) {
  if (!exp) return '';
  const details = exp.bucket_details || [];
  const topWords = exp.top_words || [];
  const activeBuckets = state.buckets.filter((b) => b.name !== 'unclassified');

  const probBars = details
    .map((d) => {
      const pct = (d.probability * 100).toFixed(2);
      const dispLabel = trBucketLabel(d.bucket_name || d.bucket_label, d.bucket_label);
      return `
        <div class="prob-bar-row">
          <div style="display:flex;align-items:center;gap:6px;font-weight:600;">
            <span class="bucket-dot" style="background:${d.bucket_color}"></span>
            <span>${escapeHTML(dispLabel)}</span>
          </div>
          <div class="prob-bar-track">
            <div class="prob-bar-fill" style="width:${Math.max(2, d.probability * 100)}%;background:${d.bucket_color};"></div>
          </div>
          <div style="text-align:right;font-weight:700;">${pct}%</div>
          <div style="text-align:right;color:var(--text-muted);font-size:11.5px;">${t('matrix_match_words', { count: d.match_words })}</div>
        </div>
      `;
    })
    .join('');

  const bucketHeaders = activeBuckets
    .map((b) => `<th style="color:${b.color}">${escapeHTML(b.name)}</th>`)
    .join('');

  const wordRows = topWords
    .map((w) => {
      const cols = activeBuckets
        .map((b) => {
          const p = ((w.bucket_probs[b.name] || 0) * 100).toFixed(1);
          const c = w.bucket_counts[b.name] || 0;
          const isBest = b.name === w.best_bucket;
          return `<td style="${isBest ? `font-weight:700;color:${b.color};` : 'color:var(--text-muted);'}">${p}% <span style="font-size:10.5px;">(${c})</span></td>`;
        })
        .join('');

      const dispBest = trBucketLabel(w.best_bucket || w.best_label, w.best_label);

      return `
        <tr>
          <td style="font-family:monospace;font-weight:600;">${escapeHTML(w.word)}</td>
          <td>${w.frequency_in_msg}</td>
          <td>
            <span class="bucket-pill" style="background:${w.best_color}">
              ${escapeHTML(dispBest)} (${(w.max_prob * 100).toFixed(1)}%)
            </span>
          </td>
          ${cols}
        </tr>
      `;
    })
    .join('');

  return `
    <div class="prob-bars-card">
      <div style="font-weight:700;margin-bottom:10px;">${t('matrix_prob_title')}</div>
      ${probBars}
    </div>

    <div style="font-weight:700;margin-bottom:8px;">${t('matrix_table_title')}</div>
    <table class="data-table">
      <thead>
        <tr>
          <th>${t('matrix_col_token')}</th>
          <th>${t('matrix_col_freq')}</th>
          <th>${t('matrix_col_best')}</th>
          ${bucketHeaders}
        </tr>
      </thead>
      <tbody>
        ${wordRows || `<tr><td colspan="8" style="text-align:center;">${t('matrix_empty')}</td></tr>`}
      </tbody>
    </table>
  `;
}

async function reclassifyCurrentEmail(bucketId, bucketLabel) {
  if (!state.selectedEmailId) return;
  try {
    await apiPost('/api/emails/reclassify', {
      email_id: state.selectedEmailId,
      bucket_id: bucketId,
    });
    showToast(`「${bucketLabel}」へ再分類し、コーパスをTOE学習しました！`, '#10b981');
    await refreshOverview();
    await loadEmails();
    await selectEmail(state.selectedEmailId);
  } catch (e) {
    showToast(e.message, '#ef4444');
  }
}

async function toggleStar(ev, id, starred) {
  ev.stopPropagation();
  try {
    await apiPost('/api/emails/mark', {
      email_ids: [id],
      action: starred ? 'star' : 'unstar',
    });
    await refreshOverview();
    await loadEmails();
  } catch (e) {
    showToast(e.message, '#ef4444');
  }
}

async function markEmails(ids, action) {
  if (action === 'trash' || action === 'delete') {
    await deleteEmailsWithUndo(ids);
    return;
  }
  try {
    await apiPost('/api/emails/mark', { email_ids: ids, action });
    showToast('メールを更新しました');
    await refreshOverview();
    await loadEmails({ preferredEmailId: state.selectedEmailId, keepListFocus: true });
  } catch (e) {
    showToast(e.message, '#ef4444');
  }
}

async function quickSimulate(count, category) {
  try {
    const res = await apiPost('/api/emails/simulate', { count, category });
    showToast(`${res.count} 通の新着サンプルメールを受信・自動分類しました！`, '#10b981');
    await refreshOverview();
    await loadEmails();
    if (res.emails && res.emails.length > 0) {
      selectEmail(res.emails[0].id);
    }
  } catch (e) {
    showToast(e.message, '#ef4444');
  }
}

function openSimulateModal() {
  document.getElementById('simulateModal').classList.add('open');
}

async function runModalSimulate() {
  const category = document.getElementById('simCategory').value;
  const count = Number(document.getElementById('simCount').value) || 3;
  closeModal('simulateModal');
  await quickSimulate(count, category);
}

async function rescanEmails(onlyUnclassified) {
  try {
    const res = await apiPost('/api/emails/rescan', { only_unclassified: onlyUnclassified });
    showToast(`${res.updated} 件のメールを最新コーパスとマグネットで再評価しました`, '#3b82f6');
    await refreshOverview();
    await loadEmails();
    if (state.selectedEmailId) {
      await selectEmail(state.selectedEmailId);
    }
  } catch (e) {
    showToast(e.message, '#ef4444');
  }
}

// Compose & Reply (with Attachment Support)
let composeAttachments = [];

function formatFileSize(bytes) {
  const n = Number(bytes) || 0;
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / (1024 * 1024)).toFixed(2)} MB`;
}

function getAttachmentIcon(contentType, filename) {
  const ct = String(contentType || '').toLowerCase();
  const fn = String(filename || '').toLowerCase();
  if (ct.startsWith('image/') || /\.(png|jpe?g|gif|webp|bmp|svg)$/.test(fn)) return '🖼️';
  if (ct === 'application/pdf' || fn.endsWith('.pdf')) return '📕';
  if (/\.(zip|rar|7z|tar|gz)$/.test(fn)) return '🗜️';
  if (/\.(xlsx?|csv)$/.test(fn)) return '📊';
  if (/\.(docx?|rtf)$/.test(fn)) return '📝';
  if (/\.(pptx?)$/.test(fn)) return '📽️';
  if (ct.startsWith('text/') || /\.(txt|md|json|xml|log|eml)$/.test(fn)) return '📄';
  return '📎';
}

function isImageAttachment(contentType, filename) {
  const ct = String(contentType || '').toLowerCase();
  const fn = String(filename || '').toLowerCase();
  return ct.startsWith('image/') || /\.(png|jpe?g|gif|webp|bmp|svg|ico)$/.test(fn);
}

function isIframePreviewableAttachment(contentType, filename) {
  const ct = String(contentType || '').toLowerCase();
  const fn = String(filename || '').toLowerCase();
  return (
    ct === 'application/pdf' ||
    fn.endsWith('.pdf') ||
    ct.startsWith('text/') ||
    /\.(txt|md|csv|json|xml|log|html?|eml)$/.test(fn)
  );
}

function selectAttachmentPreview(attId) {
  const ta = document.getElementById('emailBodyTextarea');
  const prevScroll = ta ? ta.scrollTop : 0;
  state.selectedAttachmentId = Number(attId) || 0;
  renderMailReader();
  const nextTa = document.getElementById('emailBodyTextarea');
  if (nextTa) {
    nextTa.scrollTop = prevScroll;
  }
}

function closeAttachmentPreview() {
  const ta = document.getElementById('emailBodyTextarea');
  const prevScroll = ta ? ta.scrollTop : 0;
  state.selectedAttachmentId = 0;
  renderMailReader();
  const nextTa = document.getElementById('emailBodyTextarea');
  if (nextTa) {
    nextTa.scrollTop = prevScroll;
  }
}

function openAttachmentInNewWindow(attId) {
  const id = Number(attId) || 0;
  if (!id) return;
  const url = `/api/emails/attachment?id=${id}&inline=1`;
  window.open(
    url,
    `attachment_${id}`,
    'width=1020,height=780,resizable=yes,scrollbars=yes,menubar=no,toolbar=no,location=no,status=no'
  );
}

function renderBottomAttachmentsPane(attachments) {
  if (!Array.isArray(attachments) || attachments.length === 0) return '';

  const cardsHTML = attachments
    .map((att) => {
      const icon = getAttachmentIcon(att.content_type, att.filename);
      const sizeStr = formatFileSize(att.size);
      const isSelected = Number(state.selectedAttachmentId) === Number(att.id);
      const isImg = isImageAttachment(att.content_type, att.filename);
      const inlineUrl = `/api/emails/attachment?id=${att.id}&inline=1`;

      const visualHTML = isImg
        ? `<div class="attachment-medium-thumb-wrap">
             <img src="${inlineUrl}" alt="${escapeHTML(att.filename)}" class="attachment-medium-thumb" onerror="this.style.display='none';this.nextElementSibling.style.display='flex';" />
             <div class="attachment-medium-icon" style="display:none;">${icon}</div>
           </div>`
        : `<div class="attachment-medium-icon">${icon}</div>`;

      return `
        <div class="attachment-medium-card ${isSelected ? 'active' : ''}"
             onclick="selectAttachmentPreview(${att.id})"
             title="${escapeHTML(att.filename)} (${sizeStr} / ${escapeHTML(att.content_type || 'application/octet-stream')})${t('att_card_title_suffix')}">
          ${visualHTML}
          <div class="attachment-medium-name">${escapeHTML(att.filename)}</div>
          <div class="attachment-medium-meta">${sizeStr}</div>
        </div>
      `;
    })
    .join('');

  return `
    <div class="reader-attachments-bottom-pane">
      <div class="reader-attachments-bottom-header">
        <div style="display:flex;align-items:center;gap:8px;">
          <span style="font-weight:700;color:var(--link-accent);font-size:12px;">${t('att_bottom_title', { count: attachments.length })}</span>
          <span style="font-size:11.5px;color:var(--text-muted);">${t('att_bottom_hint')}</span>
        </div>
        ${
          state.selectedAttachmentId
            ? `<button class="btn-secondary" style="padding:2px 8px;font-size:11px;" onclick="closeAttachmentPreview()">${t('att_close_preview')}</button>`
            : ''
        }
      </div>
      <div class="attachment-medium-grid">
        ${cardsHTML}
      </div>
    </div>
  `;
}

function renderRightAttachmentPreviewPane(attachments) {
  if (!Array.isArray(attachments) || attachments.length === 0 || !state.selectedAttachmentId) {
    return '';
  }
  const att = attachments.find((a) => Number(a.id) === Number(state.selectedAttachmentId));
  if (!att) return '';

  const icon = getAttachmentIcon(att.content_type, att.filename);
  const sizeStr = formatFileSize(att.size);
  const inlineUrl = `/api/emails/attachment?id=${att.id}&inline=1`;
  const downloadUrl = `/api/emails/attachment?id=${att.id}`;
  const isImg = isImageAttachment(att.content_type, att.filename);
  const isIframe = !isImg && isIframePreviewableAttachment(att.content_type, att.filename);

  let bodyHTML = '';
  if (isImg) {
    bodyHTML = `
      <div class="attachment-preview-image-container">
        <img src="${inlineUrl}" alt="${escapeHTML(att.filename)}" class="attachment-preview-img" />
      </div>
    `;
  } else if (isIframe) {
    bodyHTML = `
      <iframe src="${inlineUrl}" class="attachment-preview-iframe" title="${escapeHTML(att.filename)}"></iframe>
    `;
  } else {
    bodyHTML = `
      <div class="attachment-preview-fallback">
        <div class="attachment-preview-fallback-icon">${icon}</div>
        <div style="font-size:15px;font-weight:700;color:#f8fafc;word-break:break-all;text-align:center;">
          ${escapeHTML(att.filename)}
        </div>
        <div style="font-size:12px;color:var(--text-muted);">
          ${t('att_type_size', { type: escapeHTML(att.content_type || 'application/octet-stream'), size: sizeStr })}
        </div>
        <div style="display:flex;gap:10px;margin-top:10px;flex-wrap:wrap;justify-content:center;">
          <button class="btn-primary" onclick="openAttachmentInNewWindow(${att.id})">
            ${t('att_open_newwin')}
          </button>
          <a class="btn-secondary" style="text-decoration:none;display:inline-flex;align-items:center;" href="${downloadUrl}" download="${escapeHTML(att.filename)}">
            ${t('att_save_full')}
          </a>
        </div>
      </div>
    `;
  }

  return `
    <div class="reader-attachment-preview-pane">
      <div class="attachment-preview-header">
        <div class="attachment-preview-title" title="${escapeHTML(att.filename)} (${sizeStr})">
          <span style="font-size:16px;flex-shrink:0;">${icon}</span>
          <span class="attachment-preview-filename">${escapeHTML(att.filename)}</span>
          <span class="attachment-preview-filesize">(${sizeStr})</span>
        </div>
        <div class="attachment-preview-actions">
          <button class="btn-primary attachment-open-window-btn" onclick="openAttachmentInNewWindow(${att.id})" title="${t('att_open_newwin_title')}">
            ${t('att_open_newwin')}
          </button>
          <a class="btn-secondary attachment-save-btn" href="${downloadUrl}" download="${escapeHTML(att.filename)}" title="${t('att_save_title')}">
            ${t('att_save')}
          </a>
          <button class="btn-secondary attachment-close-btn" onclick="closeAttachmentPreview()" title="${t('att_close_title')}">
            ✕
          </button>
        </div>
      </div>
      <div class="attachment-preview-body">
        ${bodyHTML}
      </div>
    </div>
  `;
}

function renderComposeAttachmentsList() {
  const listEl = document.getElementById('composeAttachmentsList');
  const hintEl = document.getElementById('composeAttachmentHint');
  if (!listEl) return;

  if (composeAttachments.length === 0) {
    listEl.innerHTML = '';
    if (hintEl) hintEl.style.display = 'block';
    return;
  }
  if (hintEl) hintEl.style.display = 'none';

  listEl.innerHTML = composeAttachments
    .map((att, idx) => {
      const icon = getAttachmentIcon(att.content_type, att.filename);
      return `
        <div class="attachment-item-chip">
          <span>${icon}</span>
          <span style="font-weight:600;max-width:210px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;">${escapeHTML(att.filename)}</span>
          <span style="color:var(--text-muted);font-size:11px;">(${formatFileSize(att.size)})</span>
          <span style="cursor:pointer;color:#fca5a5;font-weight:700;padding:0 3px;" title="添付を解除" onclick="removeComposeAttachment(${idx})">✕</span>
        </div>
      `;
    })
    .join('');
}

function addFilesToComposeAttachments(fileList) {
  if (!fileList || fileList.length === 0) return;
  Array.from(fileList).forEach((file) => {
    const reader = new FileReader();
    reader.onload = () => {
      const b64 = arrayBufferToBase64(reader.result);
      composeAttachments.push({
        filename: file.name || 'attachment.bin',
        content_type: file.type || 'application/octet-stream',
        size: file.size || 0,
        data_base64: b64,
      });
      renderComposeAttachmentsList();
    };
    reader.readAsArrayBuffer(file);
  });
}

function onComposeFilesSelected(ev) {
  if (ev.target && ev.target.files) {
    addFilesToComposeAttachments(ev.target.files);
    ev.target.value = '';
  }
}

function onComposeFilesDropped(ev) {
  ev.preventDefault();
  const dz = document.getElementById('composeAttachmentDropzone');
  if (dz) dz.classList.remove('dragover');
  if (ev.dataTransfer && ev.dataTransfer.files) {
    addFilesToComposeAttachments(ev.dataTransfer.files);
  }
}

function removeComposeAttachment(idx) {
  composeAttachments.splice(idx, 1);
  renderComposeAttachmentsList();
}

function detectAccountIdForEmail(em) {
  if (!em || !Array.isArray(state.accounts) || state.accounts.length === 0) return 0;
  const uidl = String(em.uidl || '').toLowerCase();
  if (uidl.startsWith('pop3:') || uidl.startsWith('imap:')) {
    const accEmail = (uidl.split(':')[1] || '').trim();
    const matched = state.accounts.find((a) => String(a.email_address || '').toLowerCase() === accEmail);
    if (matched) return matched.id;
  }
  const toLow = String(em.to_address || '').toLowerCase();
  for (const a of state.accounts) {
    const accEmail = String(a.email_address || '').toLowerCase();
    if (accEmail && toLow.includes(accEmail)) {
      return a.id;
    }
  }
  return 0;
}

function splitAddressHeaderList(headerVal) {
  const s = String(headerVal || '').trim();
  if (!s) return [];
  const res = [];
  let cur = '';
  let inQuotes = false;
  for (let i = 0; i < s.length; i++) {
    const ch = s[i];
    if (ch === '"') {
      inQuotes = !inQuotes;
      cur += ch;
    } else if ((ch === ',' || ch === ';') && !inQuotes) {
      if (cur.trim()) res.push(cur.trim());
      cur = '';
    } else {
      cur += ch;
    }
  }
  if (cur.trim()) res.push(cur.trim());
  return res;
}

function extractBareEmail(addrToken) {
  const s = String(addrToken || '').trim();
  const angleMatch = s.match(/<([^>]+)>/);
  if (angleMatch && angleMatch[1]) {
    return angleMatch[1].trim().toLowerCase();
  }
  return s.replace(/^["'\s]+|["'\s]+$/g, '').toLowerCase();
}

function formatQuotedReplyBody(em) {
  const senderLabel = em.from_name ? `${em.from_name} <${em.from_address}>` : em.from_address;
  const rawBody = String(em.body_text || '').replace(/\r\n/g, '\n');
  const quotedLines = rawBody
    .split('\n')
    .map((line) => `> ${line}`)
    .join('\n');
  return `\n\n--- On ${em.date}, ${senderLabel} wrote: ---\n${quotedLines}`;
}

function openComposeModal(to = '', subject = '', body = '', cc = '', bcc = '', options = {}) {
  const {
    title = t('compose_title_new'),
    accountId = 0,
    preserveAttachments = false,
    focusField = '',
  } = options;

  if (!preserveAttachments) {
    composeAttachments = [];
  }
  renderComposeAttachmentsList();
  const fileInput = document.getElementById('composeAttachmentInput');
  if (fileInput) fileInput.value = '';

  const titleEl = document.getElementById('composeModalTitle');
  if (titleEl) titleEl.textContent = title;

  const accSelect = document.getElementById('composeAccount');
  if (state.accounts.length > 0) {
    accSelect.innerHTML =
      `<option value="0">${escapeHTML(t('compose_local_demo'))}</option>` +
      state.accounts
        .map((a) => `<option value="${a.id}">${escapeHTML(a.name)} (&lt;${escapeHTML(a.email_address)}&gt;)</option>`)
        .join('');
    if (accountId && state.accounts.some((a) => Number(a.id) === Number(accountId))) {
      accSelect.value = String(accountId);
    }
  } else {
    accSelect.innerHTML = `<option value="0">${escapeHTML(t('compose_local_addr'))}</option>`;
  }
  document.getElementById('composeTo').value = to;
  document.getElementById('composeCc').value = cc;
  document.getElementById('composeBcc').value = bcc;
  document.getElementById('composeSubject').value = subject;
  document.getElementById('composeBody').value = body;
  document.getElementById('composeModal').classList.add('open');

  setTimeout(() => {
    if (focusField === 'to') {
      const toEl = document.getElementById('composeTo');
      if (toEl) toEl.focus();
    } else if (focusField === 'body') {
      const bodyEl = document.getElementById('composeBody');
      if (bodyEl) {
        bodyEl.focus();
        bodyEl.setSelectionRange(0, 0);
        bodyEl.scrollTop = 0;
      }
    }
  }, 30);
}

// 1. 差出人に返信 (Reply to Sender)
function replyCurrentEmail() {
  if (!state.selectedDetail || !state.selectedDetail.email) return;
  const em = state.selectedDetail.email;
  const parsedHeaders = parseEmailHeaders(em, '0');
  const replyToEntry = parsedHeaders.map.get('reply-to');
  const replyTarget = replyToEntry && replyToEntry.value ? replyToEntry.value : em.from_address;

  const subj = /^re:/i.test(em.subject || '') ? em.subject : `Re: ${em.subject}`;
  const quote = formatQuotedReplyBody(em);
  const accountId = detectAccountIdForEmail(em);

  openComposeModal(replyTarget, subj, quote, '', '', {
    title: t('compose_title_reply'),
    accountId,
    focusField: 'body',
  });
}

// 2. 全員に返信 (Reply All)
function replyAllCurrentEmail() {
  if (!state.selectedDetail || !state.selectedDetail.email) return;
  const em = state.selectedDetail.email;
  const parsedHeaders = parseEmailHeaders(em, '0');

  const replyToEntry = parsedHeaders.map.get('reply-to');
  const primaryTo = replyToEntry && replyToEntry.value ? replyToEntry.value : em.from_address;

  // Build set of own addresses and primaryTo addresses to exclude from Cc
  const excludedEmails = new Set(['me@example.jp', 'me@local.tunbridge', 'me@local.popfile']);
  for (const a of state.accounts || []) {
    if (a.email_address) excludedEmails.add(String(a.email_address).toLowerCase());
  }
  const uidl = String(em.uidl || '').toLowerCase();
  if (uidl.startsWith('pop3:') || uidl.startsWith('imap:')) {
    const accAddr = (uidl.split(':')[1] || '').trim();
    if (accAddr) excludedEmails.add(accAddr);
  }

  const toTokens = [];
  const seenEmails = new Set();
  for (const item of splitAddressHeaderList(primaryTo)) {
    const bare = extractBareEmail(item);
    if (bare && !seenEmails.has(bare)) {
      seenEmails.add(bare);
      toTokens.push(item);
    }
  }
  // If From was different from Reply-To, also include From in To unless excluded
  if (em.from_address) {
    const bareFrom = extractBareEmail(em.from_address);
    if (bareFrom && !seenEmails.has(bareFrom) && !excludedEmails.has(bareFrom)) {
      seenEmails.add(bareFrom);
      toTokens.push(em.from_address);
    }
  }

  // Collect original To and Cc headers for Reply-All Cc list
  const origToRaw = (parsedHeaders.map.get('to') && parsedHeaders.map.get('to').value) || em.to_address || '';
  const origCcRaw = (parsedHeaders.map.get('cc') && parsedHeaders.map.get('cc').value) || '';
  const ccTokens = [];
  for (const item of [...splitAddressHeaderList(origToRaw), ...splitAddressHeaderList(origCcRaw)]) {
    const bare = extractBareEmail(item);
    if (!bare || seenEmails.has(bare) || excludedEmails.has(bare)) continue;
    seenEmails.add(bare);
    ccTokens.push(item);
  }

  const subj = /^re:/i.test(em.subject || '') ? em.subject : `Re: ${em.subject}`;
  const quote = formatQuotedReplyBody(em);
  const accountId = detectAccountIdForEmail(em);

  openComposeModal(toTokens.join(', '), subj, quote, ccTokens.join(', '), '', {
    title: t('compose_title_reply_all'),
    accountId,
    focusField: 'body',
  });
}

// 3. 転送 (Forward, including original attachments)
async function forwardCurrentEmail() {
  if (!state.selectedDetail || !state.selectedDetail.email) return;
  const em = state.selectedDetail.email;
  const parsedHeaders = parseEmailHeaders(em, '0');

  const fromDisplay = em.from_name ? `${em.from_name} <${em.from_address}>` : em.from_address;
  const toVal = (parsedHeaders.map.get('to') && parsedHeaders.map.get('to').value) || em.to_address || '';
  const ccVal = (parsedHeaders.map.get('cc') && parsedHeaders.map.get('cc').value) || '';

  let fwdHeaderBlock =
    `\n\n---------- Forwarded message ---------\n` +
    `From: ${fromDisplay}\n` +
    `Date: ${em.date}\n` +
    `Subject: ${em.subject}\n` +
    `To: ${toVal}\n`;
  if (ccVal) {
    fwdHeaderBlock += `Cc: ${ccVal}\n`;
  }
  fwdHeaderBlock += `\n${em.body_text || ''}`;

  const subj = /^(fwd?|fw):/i.test(em.subject || '') ? em.subject : `Fwd: ${em.subject}`;
  const accountId = detectAccountIdForEmail(em);

  composeAttachments = [];
  openComposeModal('', subj, fwdHeaderBlock, '', '', {
    title: t('compose_title_forward'),
    accountId,
    preserveAttachments: true,
    focusField: 'to',
  });

  // Carry over attachments from the original email asynchronously
  if (Array.isArray(em.attachments) && em.attachments.length > 0) {
    for (const att of em.attachments) {
      try {
        const resp = await fetch(`/api/emails/attachment?id=${att.id}`);
        if (resp.ok) {
          const buf = await resp.arrayBuffer();
          composeAttachments.push({
            filename: att.filename || 'attachment.bin',
            content_type: att.content_type || 'application/octet-stream',
            size: att.size || buf.byteLength,
            data_base64: arrayBufferToBase64(buf),
          });
        }
      } catch (e) {
        console.error('Failed to carry over attachment for forward:', e);
      }
    }
    renderComposeAttachmentsList();
  }
}

async function sendComposedEmail() {
  const accountId = Number(document.getElementById('composeAccount').value) || 0;
  const to = document.getElementById('composeTo').value.trim();
  const cc = document.getElementById('composeCc').value.trim();
  const bcc = document.getElementById('composeBcc').value.trim();
  const subject = document.getElementById('composeSubject').value.trim();
  const body = document.getElementById('composeBody').value;
  try {
    const res = await apiPost('/api/emails/send', {
      account_id: accountId,
      to,
      cc,
      bcc,
      subject,
      body,
      attachments: composeAttachments,
    });
    composeAttachments = [];
    clearFailedNetLogs();
    closeModal('composeModal');
    showToast(res.message || '送信しました', '#10b981');
    await refreshOverview();
    if (state.folder === 'sent') {
      await loadEmails();
    }
  } catch (e) {
    if (e.netLogs && e.netLogs.length > 0) {
      recordFailedNetLogs(e.netLogs);
      showToast(e.message, '#ef4444', t('netlog_btn_view'), () => openNetLogModal(), 9000);
    } else {
      showToast(e.message, '#ef4444');
    }
  }
}

// Import EML
let selectedEMLBase64 = '';

function openImportModal() {
  selectedEMLBase64 = '';
  document.getElementById('importRawText').value = '';
  const fileInput = document.getElementById('importFileInput');
  if (fileInput) fileInput.value = '';
  document.getElementById('importModal').classList.add('open');
}

function arrayBufferToBase64(buffer) {
  let binary = '';
  const bytes = new Uint8Array(buffer);
  const chunkSize = 0x8000;
  for (let i = 0; i < bytes.length; i += chunkSize) {
    binary += String.fromCharCode.apply(null, bytes.subarray(i, i + chunkSize));
  }
  return btoa(binary);
}

function onEMLFileSelected(ev) {
  const file = ev.target.files && ev.target.files[0];
  if (!file) return;
  const binReader = new FileReader();
  binReader.onload = () => {
    selectedEMLBase64 = arrayBufferToBase64(binReader.result);
  };
  binReader.readAsArrayBuffer(file);

  const textReader = new FileReader();
  textReader.onload = () => {
    document.getElementById('importRawText').value = textReader.result;
  };
  textReader.readAsText(file);
}

async function submitImportEML() {
  const raw = document.getElementById('importRawText').value.trim();
  if (!raw && !selectedEMLBase64) {
    showToast('EMLテキストを入力してください', '#ef4444');
    return;
  }
  try {
    const payload = selectedEMLBase64
      ? { raw_eml_base64: selectedEMLBase64, raw_eml: raw }
      : { raw_eml: raw };
    const res = await apiPost('/api/emails/import', payload);
    selectedEMLBase64 = '';
    closeModal('importModal');
    showToast(`インポート完了: バケット「${res.classification.bucket_label}」に自動分類されました`, '#10b981');
    await refreshOverview();
    await loadEmails();
    if (res.email) selectEmail(res.email.id);
  } catch (e) {
    showToast(e.message, '#ef4444');
  }
}

function closeModal(id) {
  document.getElementById(id).classList.remove('open');
}

// Magnets View
async function loadMagnetsView() {
  try {
    const data = await apiGet('/api/magnets');
    const magnets = data.magnets || [];
    const tbody = document.getElementById('magnetsTableBody');
    const isEn = getCurrentLang() === 'en';
    const fieldNames = isEn
      ? {
          from: 'From (Sender)',
          domain: 'Domain',
          subject: 'Subject',
          to: 'To (Recipient)',
        }
      : {
          from: 'From (差出人)',
          domain: 'Domain (ドメイン)',
          subject: 'Subject (件名)',
          to: 'To (宛先)',
        };
    tbody.innerHTML =
      magnets
        .map(
          (m) => `
        <tr>
          <td>
            <input type="checkbox" ${m.enabled ? 'checked' : ''} onchange="toggleMagnet(${m.id}, this.checked)" />
            ${m.enabled ? t('magnets_active') : t('magnets_inactive')}
          </td>
          <td><strong>${escapeHTML(fieldNames[m.field] || m.field)}</strong></td>
          <td style="font-family:monospace;color:#38bdf8;">${escapeHTML(m.pattern)}</td>
          <td><span class="bucket-pill" style="background:${m.bucket_color}">${escapeHTML(trBucketLabel(m.bucket_name || m.bucket_label, m.bucket_label))}</span></td>
          <td>${m.hit_count} ${t('stat_magnet_unit')}</td>
          <td style="color:var(--text-muted);">${escapeHTML(m.description)}</td>
          <td>
            <button class="btn-danger" onclick="deleteMagnet(${m.id})">${t('buckets_delete_btn')}</button>
          </td>
        </tr>
      `
        )
        .join('') || `<tr><td colspan="7" style="text-align:center;">${t('magnets_empty')}</td></tr>`;
  } catch (e) {
    showToast(e.message, '#ef4444');
  }
}

async function createMagnet() {
  const field = document.getElementById('magField').value;
  const pattern = document.getElementById('magPattern').value.trim();
  const bucketId = Number(document.getElementById('magBucket').value);
  const desc = document.getElementById('magDesc').value.trim();
  try {
    await apiPost('/api/magnets/create', {
      bucket_id: bucketId,
      field,
      pattern,
      description: desc,
    });
    document.getElementById('magPattern').value = '';
    document.getElementById('magDesc').value = '';
    showToast('マグネットを追加しました', '#10b981');
    await loadMagnetsView();
    await refreshOverview();
  } catch (e) {
    showToast(e.message, '#ef4444');
  }
}

async function createMagnetFromEmail() {
  if (!state.selectedDetail || !state.selectedDetail.email) return;
  const em = state.selectedDetail.email;
  switchMainView('magnets');
  document.getElementById('magField').value = 'from';
  document.getElementById('magPattern').value = em.from_address;
  document.getElementById('magBucket').value = String(em.bucket_id);
  document.getElementById('magDesc').value = `${em.from_name || em.from_address} からのメールを固定振り分け`;
  showToast('差出人アドレスをマグネット作成フォームにセットしました');
}

async function toggleMagnet(id, enabled) {
  await apiPost('/api/magnets/toggle', { id, enabled });
  await loadMagnetsView();
}

async function deleteMagnet(id) {
  await apiPost('/api/magnets/delete', { id });
  showToast('マグネットを削除しました');
  await loadMagnetsView();
}

// Corpus & Word Lookup View
async function loadCorpusWords() {
  const sel = document.getElementById('corpusBucketSelect');
  if (!sel || !sel.value) return;
  const bucketId = Number(sel.value);
  const search = document.getElementById('corpusSearchInput').value.trim();
  try {
    const data = await apiGet(`/api/corpus?bucket_id=${bucketId}&search=${encodeURIComponent(search)}&limit=120`);
    const words = data.words || [];
    document.getElementById('corpusCountLabel').textContent = t('corpus_showing_count', { count: words.length });
    const tbody = document.getElementById('corpusTableBody');
    tbody.innerHTML =
      words
        .map((w) => {
          const dist = Object.entries(w.all_buckets || {})
            .map(([bname, cnt]) => {
              const bObj = state.buckets.find((x) => x.name === bname);
              const col = bObj ? bObj.color : '#64748b';
              return `<span class="bucket-pill" style="background:${col};margin-right:4px;">${escapeHTML(bname)}: ${cnt}</span>`;
            })
            .join('');
          return `
            <tr>
              <td style="font-family:monospace;font-weight:700;cursor:pointer;color:#38bdf8;" onclick="document.getElementById('lookupWordInput').value='${escapeHTML(w.word)}';lookupWord();">${escapeHTML(w.word)}</td>
              <td><strong>${w.count}</strong></td>
              <td>${(w.prob * 100).toFixed(3)}%</td>
              <td>${dist}</td>
            </tr>
          `;
        })
        .join('') || `<tr><td colspan="4" style="text-align:center;">${t('corpus_empty')}</td></tr>`;
  } catch (e) {
    console.error(e);
  }
}

async function lookupWord() {
  const word = document.getElementById('lookupWordInput').value.trim();
  if (!word) return;
  try {
    const data = await apiGet(`/api/corpus/lookup?word=${encodeURIComponent(word)}`);
    const box = document.getElementById('lookupWordResult');
    const rows = (data.breakdown || [])
      .map((b) => {
        const pct = (b.affinity * 100).toFixed(1);
        const dispLabel = trBucketLabel(b.bucket_name || b.bucket_label, b.bucket_label);
        return `
          <div style="display:flex;align-items:center;justify-content:space-between;margin-top:6px;">
            <span class="bucket-pill" style="background:${b.bucket_color}">${escapeHTML(dispLabel)}</span>
            <span>${t('lookup_result_row', { count: b.count, pct })}</span>
          </div>
        `;
      })
      .join('');
    box.innerHTML = `<div style="margin-top:6px;"><strong>${t('lookup_result_title', { word: escapeHTML(data.word) })}</strong>${rows}</div>`;
  } catch (e) {
    showToast(e.message, '#ef4444');
  }
}

async function runSandboxTest() {
  const subject = document.getElementById('sandboxSubject').value.trim();
  const body = document.getElementById('sandboxBody').value.trim();
  if (!subject && !body) {
    showToast(getCurrentLang() === 'en' ? 'Enter a test subject or body' : 'テスト用の件名または本文を入力してください', '#f59e0b');
    return;
  }
  try {
    const res = await apiPost('/api/corpus/test', { from: 'test@example.com', subject, body });
    const pct = (res.confidence * 100).toFixed(1);
    const dispLabel = trBucketLabel(res.bucket_name || res.bucket_label, res.bucket_label);
    document.getElementById('sandboxResultSummary').innerHTML = `
      ${t('sandbox_result_prefix')} <span class="bucket-pill" style="background:${res.bucket_color};font-size:12.5px;">
        ${escapeHTML(dispLabel)} (${pct}%)
      </span>
    `;
  } catch (e) {
    showToast(e.message, '#ef4444');
  }
}

// Buckets Management View
function renderBucketsTable() {
  const tbody = document.getElementById('bucketsTableBody');
  tbody.innerHTML = state.buckets
    .map((b) => {
      const priorPct = (b.prior_prob * 100).toFixed(1);
      const dispLabel = trBucketLabel(b);
      return `
        <tr>
          <td>
            <span class="bucket-pill" style="background:${b.color};font-size:12px;">
              ${escapeHTML(dispLabel)}
            </span>
          </td>
          <td style="font-family:monospace;">${escapeHTML(b.name)}</td>
          <td>${t('buckets_emails_col', { total: b.email_count, unread: b.unread_count })}</td>
          <td>${t('buckets_docs_col', { docs: b.trained_docs })}</td>
          <td>${t('buckets_words_col', { total: b.total_words, unique: b.unique_words })}</td>
          <td>${priorPct}%</td>
          <td>FP: ${b.false_pos} / FN: ${b.false_neg}</td>
          <td style="display:flex;gap:6px;">
            <button class="btn-secondary" onclick="clearBucketCorpus(${b.id}, '${escapeHTML(dispLabel)}')">${t('buckets_clear_btn')}</button>
            ${
              !b.is_system
                ? `<button class="btn-danger" onclick="deleteBucket(${b.id})">${t('buckets_delete_btn')}</button>`
                : ''
            }
          </td>
        </tr>
      `;
    })
    .join('');
}

async function createBucket() {
  const name = document.getElementById('newBucketName').value.trim();
  const label = document.getElementById('newBucketLabel').value.trim();
  const color = document.getElementById('newBucketColor').value;
  const description = document.getElementById('newBucketDesc').value.trim();
  try {
    await apiPost('/api/buckets/create', { name, label, color, description });
    document.getElementById('newBucketName').value = '';
    document.getElementById('newBucketLabel').value = '';
    document.getElementById('newBucketDesc').value = '';
    showToast(`バケット「${label}」を作成しました`, '#10b981');
    await refreshOverview();
    renderBucketsTable();
  } catch (e) {
    showToast(e.message, '#ef4444');
  }
}

async function clearBucketCorpus(id, label) {
  try {
    await apiPost('/api/buckets/clear', { id });
    showToast(`「${label}」の学習コーパスをリセットしました`);
    await refreshOverview();
    renderBucketsTable();
  } catch (e) {
    showToast(e.message, '#ef4444');
  }
}

async function deleteBucket(id) {
  try {
    await apiPost('/api/buckets/delete', { id });
    showToast('バケットを削除しました');
    await refreshOverview();
    renderBucketsTable();
  } catch (e) {
    showToast(e.message, '#ef4444');
  }
}

// History Log View
async function loadHistoryView() {
  try {
    const data = await apiGet('/api/history');
    const events = data.events || [];
    const actionLabels = {
      classified: t('history_classified'),
      magnet: t('history_magnet'),
      reclassified: t('history_reclassified'),
    };
    document.getElementById('historyTableBody').innerHTML =
      events
        .map(
          (ev) => `
        <tr>
          <td style="color:var(--text-muted);">${escapeHTML(ev.created_at)}</td>
          <td>${actionLabels[ev.action] || escapeHTML(ev.action)}</td>
          <td><strong>${escapeHTML(ev.subject)}</strong></td>
          <td>${escapeHTML(ev.from_addr)}</td>
          <td style="color:var(--text-muted);">${escapeHTML(ev.from_bucket ? trBucketLabel(ev.from_bucket, ev.from_bucket) : '-')}</td>
          <td><strong>${escapeHTML(trBucketLabel(ev.to_bucket, ev.to_bucket))}</strong></td>
        </tr>
      `
        )
        .join('') || `<tr><td colspan="6" style="text-align:center;">${t('history_empty')}</td></tr>`;
  } catch (e) {
    showToast(e.message, '#ef4444');
  }
}

// Accounts Management (POP3 / IMAP / SMTP)
function onProtocolChange() {
  const proto = document.getElementById('accProtocol').value;
  const portEl = document.getElementById('accInPort');
  const hostEl = document.getElementById('accInHost');
  const tlsEl = document.getElementById('accInTLS');
  if (proto === 'jmap') {
    portEl.value = 443;
    if (tlsEl) tlsEl.value = '1';
    if (hostEl) hostEl.placeholder = 'api.fastmail.com / mail.example.com';
  } else if (proto === 'imap') {
    portEl.value = 993;
    if (tlsEl) tlsEl.value = '1';
    if (hostEl) hostEl.placeholder = 'imap.example.com';
  } else {
    portEl.value = 995;
    if (tlsEl) tlsEl.value = '1';
    if (hostEl) hostEl.placeholder = 'pop.example.com';
  }
}

function updateAccountEditorModeUI() {
  const idEl = document.getElementById('accId');
  const editingId = idEl ? Number(idEl.value) || 0 : 0;
  const badgeEl = document.getElementById('accountFormModeBadge');
  const resetBtn = document.getElementById('btnResetAccountForm');
  const saveBtn = document.getElementById('btnSaveAccount');
  const copyBtn = document.getElementById('btnSaveAccountAsCopy');
  const cardEl = document.getElementById('accountEditorCard');

  if (editingId > 0) {
    if (badgeEl) {
      badgeEl.style.display = 'inline-block';
      badgeEl.textContent = t('accounts_editing_badge', { id: editingId });
    }
    if (resetBtn) resetBtn.style.display = 'inline-flex';
    if (copyBtn) copyBtn.style.display = 'inline-flex';
    if (saveBtn) saveBtn.textContent = t('accounts_btn_save_overwrite');
    if (cardEl) {
      cardEl.style.borderColor = 'var(--accent)';
      cardEl.style.boxShadow = '0 0 0 2px rgba(59, 130, 246, 0.2)';
    }
  } else {
    if (badgeEl) badgeEl.style.display = 'none';
    if (resetBtn) resetBtn.style.display = 'none';
    if (copyBtn) copyBtn.style.display = 'none';
    if (saveBtn) saveBtn.textContent = t('accounts_btn_save_new');
    if (cardEl) {
      cardEl.style.borderColor = '';
      cardEl.style.boxShadow = '';
    }
  }

  // Highlight active row in table
  const tbody = document.getElementById('accountsTableBody');
  if (tbody) {
    tbody.querySelectorAll('tr[data-acc-id]').forEach((tr) => {
      const rowId = Number(tr.getAttribute('data-acc-id')) || 0;
      if (editingId > 0 && rowId === editingId) {
        tr.classList.add('account-row-editing');
      } else {
        tr.classList.remove('account-row-editing');
      }
    });
  }
}

function editAccount(id, silent = false) {
  const acc = state.accounts.find((a) => Number(a.id) === Number(id));
  if (!acc) return;

  document.getElementById('accId').value = String(acc.id);
  document.getElementById('accName').value = acc.name || '';
  document.getElementById('accEmail').value = acc.email_address || '';
  document.getElementById('accProtocol').value = acc.protocol || 'pop3';
  document.getElementById('accInHost').value = acc.incoming_host || '';
  document.getElementById('accInPort').value = acc.incoming_port || (acc.protocol === 'jmap' ? 443 : (acc.protocol === 'imap' ? 993 : 995));
  document.getElementById('accInTLS').value = acc.incoming_tls ? '1' : '0';
  document.getElementById('accUser').value = acc.username || '';
  document.getElementById('accPass').value = acc.password || '';
  document.getElementById('accSmtpHost').value = acc.smtp_host || '';
  document.getElementById('accSmtpPort').value = acc.smtp_port || 587;
  document.getElementById('accEnabled').value = acc.enabled ? '1' : '0';
  const intervalEl = document.getElementById('accCheckInterval');
  if (intervalEl) intervalEl.value = String(acc.check_interval ?? 0);

  updateAccountEditorModeUI();
  const cardEl = document.getElementById('accountEditorCard');
  if (cardEl && typeof cardEl.scrollIntoView === 'function') {
    cardEl.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
  }
  if (!silent) {
    showToast(t('accounts_toast_loaded', { name: acc.name }), '#3b82f6');
  }
}

function resetAccountForm() {
  const idEl = document.getElementById('accId');
  if (!idEl) return;
  idEl.value = '0';
  document.getElementById('accName').value = '';
  document.getElementById('accEmail').value = '';
  document.getElementById('accProtocol').value = 'pop3';
  document.getElementById('accInHost').value = '';
  document.getElementById('accInPort').value = '995';
  document.getElementById('accInTLS').value = '1';
  document.getElementById('accUser').value = '';
  document.getElementById('accPass').value = '';
  document.getElementById('accSmtpHost').value = '';
  document.getElementById('accSmtpPort').value = '587';
  document.getElementById('accEnabled').value = '1';
  const intervalEl = document.getElementById('accCheckInterval');
  if (intervalEl) intervalEl.value = '0';
  updateAccountEditorModeUI();
}

async function duplicateAccount(id, ev) {
  if (ev && typeof ev.stopPropagation === 'function') ev.stopPropagation();
  const src = state.accounts.find((a) => Number(a.id) === Number(id));
  if (!src) return;

  const copySuffix = t('accounts_copy_suffix');
  const dupAcc = {
    id: 0,
    name: `${src.name}${copySuffix}`,
    email_address: src.email_address,
    protocol: src.protocol,
    incoming_host: src.incoming_host,
    incoming_port: Number(src.incoming_port) || 995,
    incoming_tls: Boolean(src.incoming_tls),
    username: src.username,
    password: src.password,
    smtp_host: src.smtp_host || '',
    smtp_port: Number(src.smtp_port) || 587,
    smtp_tls: src.smtp_tls !== undefined ? Boolean(src.smtp_tls) : true,
    enabled: false, // Default duplicated entry to disabled so it's safe as a backup or template for editing
    check_interval: Number(src.check_interval) || 0,
  };

  try {
    const res = await apiPost('/api/accounts/save', dupAcc);
    await loadAccountsList();
    renderProviderFilterUI();
    if (res && res.account && res.account.id) {
      editAccount(res.account.id, true);
    }
    showToast(t('accounts_toast_duplicated', { name: dupAcc.name }), '#10b981');
  } catch (e) {
    showToast(e.message, '#ef4444');
  }
}

async function loadAccountsList() {
  try {
    const data = await apiGet('/api/accounts');
    state.accounts = data.accounts || [];
    const tbody = document.getElementById('accountsTableBody');
    if (!tbody) return;
    const editingId = Number(document.getElementById('accId')?.value) || 0;

    tbody.innerHTML =
      state.accounts
        .map((a) => {
          const isEditing = editingId > 0 && Number(a.id) === editingId;
          const statusHTML = a.enabled
            ? `<span style="color:#10b981;font-weight:700;font-size:12px;">${t('accounts_status_enabled')}</span>`
            : `<span style="color:var(--text-muted);font-size:12px;">${t('accounts_status_disabled')}</span>`;
          const protoDisplay = (a.protocol || 'pop3').toUpperCase();
          const tlsLabel = a.protocol === 'jmap' ? (a.incoming_tls ? 'HTTPS' : 'HTTP') : (a.incoming_tls ? 'TLS' : 'Plain');

          let intervalBadge = '';
          const intervalNum = Number(a.check_interval) || 0;
          if (intervalNum > 0) {
            let label = '';
            if (intervalNum === 60) {
              label = t('accounts_interval_1h');
            } else if (intervalNum === 120) {
              label = t('accounts_interval_2h');
            } else if (intervalNum === 180) {
              label = t('accounts_interval_3h');
            } else {
              label = t('accounts_interval_mins', { min: intervalNum });
            }
            intervalBadge = `<span style="display:inline-flex;align-items:center;gap:3px;padding:2px 8px;border-radius:999px;font-size:11px;font-weight:700;background:rgba(16,185,129,0.14);color:#10b981;border:1px solid rgba(16,185,129,0.35);">
              ⏱️ ${escapeHTML(label)}
            </span>`;
          } else {
            intervalBadge = `<span style="font-size:11px;color:var(--text-muted);">${t('accounts_interval_manual')}</span>`;
          }

          let lastCheckedHTML = '';
          if (a.last_checked_at) {
            const isErr = a.last_check_status && a.last_check_status.includes('エラー');
            const statusColor = isErr ? 'var(--danger)' : 'var(--text-muted)';
            lastCheckedHTML = `
              <div style="font-size:11px;color:var(--text-muted);margin-top:3px;">
                ${escapeHTML(a.last_checked_at)}
              </div>
              ${a.last_check_status ? `<div style="font-size:10px;color:${statusColor};line-height:1.2;margin-top:1px;">${escapeHTML(a.last_check_status)}</div>` : ''}
            `;
          }

          return `
        <tr class="account-row-clickable ${isEditing ? 'account-row-editing' : ''}" data-acc-id="${a.id}" onclick="editAccount(${a.id})" style="${!a.enabled ? 'opacity:0.75;' : ''}">
          <td>${statusHTML}</td>
          <td><strong>${escapeHTML(a.name)}</strong></td>
          <td>${escapeHTML(a.email_address)}</td>
          <td>${escapeHTML(protoDisplay)} (${tlsLabel})</td>
          <td>${escapeHTML(a.incoming_host)}:${a.incoming_port}</td>
          <td>${a.smtp_host ? `${escapeHTML(a.smtp_host)}:${a.smtp_port}` : t('accounts_local_save')}</td>
          <td>${intervalBadge}${lastCheckedHTML}</td>
          <td style="display:flex;gap:6px;flex-wrap:wrap;" onclick="event.stopPropagation()">
            <button class="btn-secondary" onclick="editAccount(${a.id})">${t('accounts_edit_btn')}</button>
            <button class="btn-secondary" onclick="duplicateAccount(${a.id}, event)">${t('accounts_duplicate_btn')}</button>
            <button class="btn-primary" onclick="fetchSingleAccount(${a.id}, event)">${t('accounts_fetch_btn')}</button>
            <button class="btn-danger" onclick="deleteAccount(${a.id}, event)">${t('accounts_delete_btn')}</button>
          </td>
        </tr>
      `;
        })
        .join('') ||
      `<tr><td colspan="8" style="text-align:center;color:var(--text-muted);">${t('accounts_empty')}</td></tr>`;
    updateAccountEditorModeUI();
  } catch (e) {
    console.error(e);
  }
}

async function saveAccount(asCopy = false) {
  const origId = Number(document.getElementById('accId').value) || 0;
  let nameVal = document.getElementById('accName').value.trim();
  if (asCopy && origId > 0) {
    const origAcc = state.accounts.find((a) => Number(a.id) === origId);
    if (origAcc && origAcc.name === nameVal) {
      nameVal = `${nameVal}${t('accounts_copy_suffix')}`;
    }
  }
  const intervalVal = Number(document.getElementById('accCheckInterval')?.value) || 0;
  const acc = {
    id: asCopy ? 0 : origId,
    name: nameVal,
    email_address: document.getElementById('accEmail').value.trim(),
    protocol: document.getElementById('accProtocol').value,
    incoming_host: document.getElementById('accInHost').value.trim(),
    incoming_port: Number(document.getElementById('accInPort').value) || 995,
    incoming_tls: document.getElementById('accInTLS').value === '1',
    username: document.getElementById('accUser').value.trim(),
    password: document.getElementById('accPass').value,
    smtp_host: document.getElementById('accSmtpHost').value.trim(),
    smtp_port: Number(document.getElementById('accSmtpPort').value) || 587,
    smtp_tls: true,
    enabled: document.getElementById('accEnabled').value === '1',
    check_interval: intervalVal,
  };
  if (!acc.name || !acc.incoming_host) {
    const isEn = typeof getCurrentLang === 'function' && getCurrentLang() === 'en';
    showToast(isEn ? 'Please enter Account Name and Incoming Server Host' : 'アカウント名と受信サーバーホストを入力してください', '#f59e0b');
    return;
  }
  try {
    const res = await apiPost('/api/accounts/save', acc);
    await loadAccountsList();
    renderProviderFilterUI();
    if (acc.id > 0) {
      showToast(t('accounts_toast_updated', { name: acc.name }), '#10b981');
      editAccount(acc.id, true);
    } else {
      showToast(t('accounts_toast_created', { name: acc.name }), '#10b981');
      if (asCopy && res && res.account && res.account.id) {
        editAccount(res.account.id, true);
      } else {
        resetAccountForm();
      }
    }
  } catch (e) {
    showToast(e.message, '#ef4444');
  }
}

async function deleteAccount(id, ev) {
  if (ev && typeof ev.stopPropagation === 'function') ev.stopPropagation();
  await apiPost('/api/accounts/delete', { id });
  const currentEditingId = Number(document.getElementById('accId')?.value) || 0;
  if (currentEditingId === Number(id)) {
    resetAccountForm();
  }
  const isEn = typeof getCurrentLang === 'function' && getCurrentLang() === 'en';
  showToast(isEn ? 'Account deleted' : 'アカウントを削除しました');
  await loadAccountsList();
  renderProviderFilterUI();
}

function exportAccounts() {
  const isEn = typeof getCurrentLang === 'function' && getCurrentLang() === 'en';
  showToast(isEn ? 'Exporting account configurations...' : 'アカウント設定をエクスポート中...');
  const a = document.createElement('a');
  a.href = '/api/accounts/export';
  a.download = `tunbridge-accounts-${new Date().toISOString().slice(0, 10)}.json`;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
}

function triggerImportAccounts() {
  const fileInput = document.getElementById('accountsImportFileInput');
  if (fileInput) {
    fileInput.value = '';
    fileInput.click();
  }
}

async function handleAccountsFileSelected(ev) {
  const file = ev.target.files && ev.target.files[0];
  if (!file) return;

  const isEn = typeof getCurrentLang === 'function' && getCurrentLang() === 'en';
  try {
    const text = await file.text();
    let parsed;
    try {
      parsed = JSON.parse(text);
    } catch (parseErr) {
      showToast(t('accounts_import_err_invalid'), '#ef4444');
      return;
    }
    const res = await apiPost('/api/accounts/import', parsed);
    if (res && res.imported !== undefined) {
      showToast(t('accounts_imported_toast', { count: res.imported }), '#10b981');
      await loadAccountsList();
      renderProviderFilterUI();
    } else {
      showToast(isEn ? 'Import completed' : 'インポートを完了しました', '#10b981');
    }
  } catch (e) {
    showToast(e.message, '#ef4444');
  }
}

async function fetchSingleAccount(id, ev) {
  if (ev && typeof ev.stopPropagation === 'function') ev.stopPropagation();
  const isEn = typeof getCurrentLang === 'function' && getCurrentLang() === 'en';
  showToast(isEn ? 'Connecting to mail server...' : 'サーバーに接続してメールを受信中...');
  try {
    const res = await apiPost('/api/accounts/fetch', { id });
    if (res.errors && res.errors.length > 0) {
      if (Array.isArray(res.net_logs) && res.net_logs.length > 0) {
        recordFailedNetLogs(res.net_logs);
        showToast(res.errors.join(' / '), '#ef4444', t('netlog_btn_view'), () => openNetLogModal(), 9000);
      } else {
        showToast(res.errors.join(' / '), '#ef4444');
      }
    } else {
      clearFailedNetLogs();
      showToast(isEn ? `Fetched and classified ${res.fetched} new message(s)` : `${res.fetched} 通の新着メールを受信・自動分類しました`, '#10b981');
    }
    await refreshOverview();
    await loadEmails();
  } catch (e) {
    showToast(e.message, '#ef4444');
  }
}

function updateFailedNetLogsIndicator() {
  const hasLogs = Array.isArray(state.lastFailedNetLogs) && state.lastFailedNetLogs.length > 0;
  const topBtn = document.getElementById('btnTopNetErrorLog');
  if (topBtn) {
    topBtn.style.display = hasLogs ? 'inline-flex' : 'none';
  }
  const banner = document.getElementById('accountsNetErrorBanner');
  if (banner) {
    banner.style.display = hasLogs ? 'block' : 'none';
  }
}

function recordFailedNetLogs(logs) {
  state.lastFailedNetLogs = Array.isArray(logs) ? logs : [];
  updateFailedNetLogsIndicator();
  if (state.lastFailedNetLogs.length > 0) {
    openNetLogModal();
  }
}

function clearFailedNetLogs() {
  state.lastFailedNetLogs = [];
  updateFailedNetLogsIndicator();
  closeModal('netLogModal');
}

function getStageBadgeInfo(stage) {
  switch (stage) {
    case 'dns':
      return { text: t('netlog_stage_dns'), color: '#a855f7', bg: 'rgba(168, 85, 247, 0.16)' };
    case 'tcp_connect':
      return { text: t('netlog_stage_tcp'), color: '#ef4444', bg: 'rgba(239, 68, 68, 0.16)' };
    case 'tls_handshake':
      return { text: t('netlog_stage_tls'), color: '#f59e0b', bg: 'rgba(245, 158, 11, 0.16)' };
    case 'greeting':
      return { text: t('netlog_stage_greeting'), color: '#f97316', bg: 'rgba(249, 115, 22, 0.16)' };
    case 'auth':
      return { text: t('netlog_stage_auth'), color: '#ec4899', bg: 'rgba(236, 72, 153, 0.16)' };
    default:
      return { text: t('netlog_stage_command'), color: '#ef4444', bg: 'rgba(239, 68, 68, 0.16)' };
  }
}

function getTagStyle(tag) {
  switch (tag) {
    case 'DNS':
      return 'background:rgba(168,85,247,0.2);color:#c084fc;border:1px solid rgba(168,85,247,0.4);';
    case 'TCP':
      return 'background:rgba(16,185,129,0.2);color:#34d399;border:1px solid rgba(16,185,129,0.4);';
    case 'TLS':
      return 'background:rgba(6,182,212,0.2);color:#22d3ee;border:1px solid rgba(6,182,212,0.4);';
    case 'C->S':
      return 'background:rgba(59,130,246,0.2);color:#60a5fa;border:1px solid rgba(59,130,246,0.4);';
    case 'S->C':
      return 'background:rgba(245,158,11,0.2);color:#fbbf24;border:1px solid rgba(245,158,11,0.4);';
    case 'ERROR':
      return 'background:rgba(239,68,68,0.25);color:#fca5a5;border:1px solid rgba(239,68,68,0.55);font-weight:700;';
    default:
      return 'background:rgba(148,163,184,0.2);color:#94a3b8;border:1px solid rgba(148,163,184,0.4);';
  }
}

function openNetLogModal() {
  const logs = state.lastFailedNetLogs || [];
  if (logs.length === 0) return;

  const isEn = typeof getCurrentLang === 'function' && getCurrentLang() === 'en';
  const bodyEl = document.getElementById('netLogModalBody');
  if (!bodyEl) return;

  bodyEl.innerHTML = logs
    .map((rep) => {
      const stageBadge = getStageBadgeInfo(rep.error_stage);
      const socketPill = rep.socket_connected
        ? `<span style="padding:3px 10px;border-radius:999px;font-size:12px;font-weight:700;background:rgba(16,185,129,0.16);color:#10b981;border:1px solid rgba(16,185,129,0.4);">
             ${escapeHTML(t('netlog_socket_connected', { local: rep.local_addr || 'local', remote: rep.remote_addr || `${rep.host}:${rep.port}` }))}
           </span>`
        : `<span style="padding:3px 10px;border-radius:999px;font-size:12px;font-weight:700;background:rgba(239,68,68,0.16);color:#ef4444;border:1px solid rgba(239,68,68,0.4);">
             ${escapeHTML(t('netlog_socket_disconnected'))}
           </span>`;

      let tlsPill = '';
      if (rep.use_tls) {
        tlsPill = rep.tls_completed
          ? `<span style="padding:3px 10px;border-radius:999px;font-size:12px;font-weight:600;background:rgba(6,182,212,0.16);color:#06b6d4;border:1px solid rgba(6,182,212,0.4);">
               ${escapeHTML(t('netlog_tls_ok', { info: rep.tls_info || 'TLS' }))}
             </span>`
          : `<span style="padding:3px 10px;border-radius:999px;font-size:12px;font-weight:600;background:rgba(245,158,11,0.16);color:#f59e0b;border:1px solid rgba(245,158,11,0.4);">
               ${escapeHTML(t('netlog_tls_fail'))}
             </span>`;
      } else {
        tlsPill = `<span style="padding:3px 10px;border-radius:999px;font-size:12px;background:rgba(148,163,184,0.15);color:var(--text-muted);border:1px solid var(--border);">
                     ${escapeHTML(t('netlog_tls_plain'))}
                   </span>`;
      }

      const diagText = isEn ? rep.diagnosis_en || rep.diagnosis_ja : rep.diagnosis_ja || rep.diagnosis_en;
      const linesHTML = (rep.lines || [])
        .map((ln) => {
          const isErr = ln.tag === 'ERROR';
          return `
            <div style="display:flex;align-items:baseline;gap:8px;padding:3px 6px;border-radius:4px;${isErr ? 'background:rgba(239,68,68,0.12);' : ''}">
              <span style="color:#64748b;font-size:11px;flex-shrink:0;user-select:none;">${escapeHTML(ln.time)}</span>
              <span style="font-size:10.5px;padding:1px 6px;border-radius:4px;min-width:48px;text-align:center;flex-shrink:0;${getTagStyle(ln.tag)}">${escapeHTML(ln.tag)}</span>
              <span style="word-break:break-all;color:${isErr ? '#fca5a5' : '#e2e8f0'};">${escapeHTML(ln.text)}</span>
            </div>
          `;
        })
        .join('');

      const editBtnHTML =
        rep.account_id > 0
          ? `<button class="btn-secondary" onclick="jumpToEditAccountFromNetLog(${rep.account_id})">${escapeHTML(t('netlog_edit_account_btn'))}</button>`
          : '';

      return `
        <div class="card" style="padding:14px;border-color:var(--border);display:flex;flex-direction:column;gap:10px;">
          <div style="display:flex;align-items:center;justify-content:space-between;flex-wrap:wrap;gap:8px;">
            <div style="display:flex;align-items:center;gap:8px;flex-wrap:wrap;">
              <span style="font-weight:700;font-size:14.5px;">${escapeHTML(rep.account_name || 'Mail Account')}</span>
              <span style="font-family:Consolas,monospace;font-size:12.5px;padding:2px 8px;border-radius:6px;background:var(--bg-app);border:1px solid var(--border);">
                ${escapeHTML(rep.protocol)} — ${escapeHTML(rep.host)}:${rep.port} (user: ${escapeHTML(rep.username || '-')})
              </span>
            </div>
            ${editBtnHTML}
          </div>

          <div style="display:flex;flex-wrap:wrap;gap:8px;align-items:center;">
            ${socketPill}
            ${tlsPill}
            <span style="padding:3px 10px;border-radius:999px;font-size:12px;font-weight:700;background:${stageBadge.bg};color:${stageBadge.color};border:1px solid ${stageBadge.color};">
              ${escapeHTML(stageBadge.text)}
            </span>
          </div>

          <div style="background:rgba(239,68,68,0.09);border-left:4px solid var(--danger);padding:10px 14px;border-radius:6px;line-height:1.6;">
            <div style="font-weight:700;font-size:13px;margin-bottom:4px;">${escapeHTML(diagText)}</div>
            <div style="font-family:Consolas,monospace;font-size:12px;color:var(--text-muted);word-break:break-all;">
              Raw Error: ${escapeHTML(rep.error_message || '')}
            </div>
          </div>

          <div>
            <div style="font-size:12px;font-weight:700;color:var(--text-muted);margin-bottom:6px;">
              ${escapeHTML(t('netlog_transcript_title'))}
            </div>
            <div style="background:#090e1a;color:#e2e8f0;border:1px solid var(--border);border-radius:8px;padding:10px;font-family:Consolas,monospace;font-size:12px;max-height:260px;overflow-y:auto;display:flex;flex-direction:column;gap:2px;">
              ${linesHTML}
            </div>
          </div>
        </div>
      `;
    })
    .join('');

  document.getElementById('netLogModal').classList.add('open');
}

function jumpToEditAccountFromNetLog(accId) {
  closeModal('netLogModal');
  closeModal('composeModal');
  switchMainView('accounts');
  setTimeout(() => {
    editAccount(accId);
  }, 50);
}

async function copyNetLogToClipboard() {
  const logs = state.lastFailedNetLogs || [];
  if (logs.length === 0) return;
  const textBlocks = logs.map((rep) => {
    const header = [
      `=== [${rep.protocol}] ${rep.account_name} (${rep.host}:${rep.port}, TLS=${rep.use_tls}, user=${rep.username}) ===`,
      `Socket Connected: ${rep.socket_connected} (Local=${rep.local_addr || '-'} -> Remote=${rep.remote_addr || '-'})`,
      `TLS Completed: ${rep.tls_completed} (${rep.tls_info || '-'})`,
      `Error Stage: ${rep.error_stage}`,
      `Diagnosis: ${rep.diagnosis_ja}`,
      `Raw Error: ${rep.error_message}`,
      `--- Trace Log ---`,
    ];
    const lines = (rep.lines || []).map((ln) => `${ln.time} [${ln.tag}] ${ln.text}`);
    return [...header, ...lines].join('\n');
  });
  try {
    await navigator.clipboard.writeText(textBlocks.join('\n\n'));
    showToast(t('netlog_copied_toast'), '#10b981');
  } catch (e) {
    showToast(e.message, '#ef4444');
  }
}

async function fetchAllAccounts() {
  if (state.accounts.length === 0) {
    // If no external POP3/IMAP server is configured, offer immediate demo reception
    await quickSimulate(2, 'random');
    return;
  }
  await fetchSingleAccount(0);
}
