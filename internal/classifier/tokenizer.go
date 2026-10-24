package classifier

import (
	"html"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

var (
	reHTMLTag       = regexp.MustCompile(`(?i)<\s*(a|img|form|script|iframe|style|table)\b[^>]*>`)
	reHTMLComment   = regexp.MustCompile(`(?s)<!--.*?-->`)
	reHTMLStyle     = regexp.MustCompile(`(?is)<\s*(style|script|noscript|title)\b[^>]*>.*?</\s*(style|script|noscript|title)\s*>`)
	reHTMLBlockEnd  = regexp.MustCompile(`(?i)<\s*br\s*/?\s*>|</\s*(p|div|tr|li|h[1-6]|table|blockquote|section|article|header|footer)\s*>`)
	reStripTags     = regexp.MustCompile(`<[^>]+>`)
	reURL           = regexp.MustCompile(`https?://[^\s"'<>()]+`)
	reASCIIWord     = regexp.MustCompile(`[\p{Latin}][\p{Latin}0-9_\-\$%]{1,35}`)
	reKanji         = regexp.MustCompile(`[\p{Han}]{2,10}`)
	reKatakana      = regexp.MustCompile(`[\p{Katakana}ー]{2,16}`)
	reHangul        = regexp.MustCompile(`[\p{Hangul}]{2,12}`)
	reCyrillic      = regexp.MustCompile(`[\p{Cyrillic}]{2,24}`)
	reOtherScripts  = regexp.MustCompile(`[\p{Arabic}\p{Hebrew}\p{Greek}\p{Thai}\p{Devanagari}]{2,20}`)
	reHiraganaKw    = regexp.MustCompile(`(お支払い|ご確認|ご案内|ご注意|お知らせ|無料|当選|口座|振込|至急|重要|請求|停止|登録|解除|割引|限定|還付|確定|申告|領収書)`)
)

var stopWords = map[string]bool{
	"the": true, "and": true, "for": true, "that": true, "this": true,
	"with": true, "from": true, "your": true, "have": true, "are": true,
	"was": true, "were": true, "will": true, "has": true, "been": true,
	"not": true, "but": true, "you": true, "all": true, "can": true,
	"http": true, "https": true, "com": true, "net": true, "org": true,
	"www": true, "html": true, "mailto": true,
}

// TokenizeEmail extracts token frequencies from an email's headers and body.
// Header tokens are prefixed (e.g., "from:...", "domain:...", "subject:...").
func TokenizeEmail(fromName, fromAddr, toAddr, subject, bodyText, bodyHTML string) map[string]int {
	counts := make(map[string]int)

	addToken := func(tok string, weight int) {
		tok = strings.ToLower(strings.TrimSpace(tok))
		if tok == "" || stopWords[tok] {
			return
		}
		counts[tok] += weight
	}

	// 1. From / Domain pseudo-tokens
	fromAddrClean := strings.ToLower(strings.TrimSpace(fromAddr))
	if fromAddrClean != "" {
		addToken("from:"+fromAddrClean, 2)
		if at := strings.LastIndex(fromAddrClean, "@"); at != -1 && at+1 < len(fromAddrClean) {
			domain := fromAddrClean[at+1:]
			addToken("domain:"+domain, 2)
		}
	}
	for _, w := range ExtractRawWords(fromName) {
		addToken("from:"+w, 1)
	}

	// 2. To pseudo-token
	toAddrClean := strings.ToLower(strings.TrimSpace(toAddr))
	if toAddrClean != "" {
		addToken("to:"+toAddrClean, 1)
	}

	// 3. Subject tokens (both "subject:<word>" and "<word>")
	for _, w := range ExtractRawWords(subject) {
		addToken("subject:"+w, 2)
		addToken(w, 1)
	}

	// 4. HTML structural pseudo-tokens
	if bodyHTML != "" {
		matches := reHTMLTag.FindAllStringSubmatch(bodyHTML, -1)
		for _, m := range matches {
			if len(m) > 1 {
				addToken("html:"+strings.ToLower(m[1]), 1)
			}
		}
	}

	// 5. URL host pseudo-tokens
	combinedBody := bodyText
	if combinedBody == "" && bodyHTML != "" {
		combinedBody = StripHTML(bodyHTML)
	}
	urls := reURL.FindAllString(combinedBody+" "+bodyHTML, -1)
	for _, rawU := range urls {
		if parsed, err := url.Parse(rawU); err == nil && parsed.Host != "" {
			host := strings.ToLower(parsed.Hostname())
			if host != "" {
				addToken("url:"+host, 2)
			}
		}
	}

	// 6. Body content tokens
	for _, w := range ExtractRawWords(combinedBody) {
		addToken(w, 1)
	}

	return counts
}

// ExtractRawWords extracts Latin/ASCII words, Kanji/Hanzi compounds (+ bigrams), Katakana, Hangul, Cyrillic, and other multilingual tokens.
func ExtractRawWords(text string) []string {
	if text == "" {
		return nil
	}
	text = stripZeroWidthForTokenize(text)
	var tokens []string

	// Latin / ASCII words (including European accented characters)
	for _, m := range reASCIIWord.FindAllString(text, -1) {
		low := strings.ToLower(m)
		if !stopWords[low] && len([]rune(low)) >= 2 {
			tokens = append(tokens, low)
		}
	}

	// Katakana sequences
	for _, m := range reKatakana.FindAllString(text, -1) {
		runes := []rune(m)
		if len(runes) >= 2 {
			tokens = append(tokens, m)
		}
	}

	// Kanji / Hanzi sequences + overlapping 2-gram for compound words
	for _, m := range reKanji.FindAllString(text, -1) {
		runes := []rune(m)
		if len(runes) >= 2 && len(runes) <= 6 {
			tokens = append(tokens, m)
		}
		if len(runes) > 2 {
			for i := 0; i+2 <= len(runes); i++ {
				bigram := string(runes[i : i+2])
				tokens = append(tokens, bigram)
			}
		}
	}

	// Korean Hangul sequences
	for _, m := range reHangul.FindAllString(text, -1) {
		runes := []rune(m)
		if len(runes) >= 2 {
			tokens = append(tokens, m)
		}
	}

	// Cyrillic words
	for _, m := range reCyrillic.FindAllString(text, -1) {
		low := strings.ToLower(m)
		if len([]rune(low)) >= 2 {
			tokens = append(tokens, low)
		}
	}

	// Other multilingual scripts (Arabic, Hebrew, Greek, Thai, Devanagari)
	for _, m := range reOtherScripts.FindAllString(text, -1) {
		low := strings.ToLower(m)
		if len([]rune(low)) >= 2 {
			tokens = append(tokens, low)
		}
	}

	// Key Japanese phrases that include Hiragana
	for _, m := range reHiraganaKw.FindAllString(text, -1) {
		tokens = append(tokens, m)
	}

	return tokens
}

func stripZeroWidthForTokenize(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\u200b', '\u200c', '\u200d', '\u200e', '\u200f',
			'\u202a', '\u202b', '\u202c', '\u202d', '\u202e',
			'\u2060', '\u2061', '\u2062', '\u2063', '\u2064',
			'\ufeff', '\u00ad', '\u0330':
			continue
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// StripHTML removes comments, <style>/<script> blocks, HTML tags, unescapes HTML entities, and normalizes whitespace.
func StripHTML(htmlStr string) string {
	if htmlStr == "" {
		return ""
	}
	s := reHTMLComment.ReplaceAllString(htmlStr, "")
	s = reHTMLStyle.ReplaceAllString(s, " ")
	s = reHTMLBlockEnd.ReplaceAllString(s, "\n")
	s = reStripTags.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	s = stripZeroWidthForTokenize(s)

	var b strings.Builder
	b.Grow(len(s))
	lastSpace := false
	consecutiveNewlines := 0

	for _, r := range s {
		if r == '\r' {
			continue
		}
		if r == '\n' {
			consecutiveNewlines++
			if consecutiveNewlines <= 2 {
				b.WriteRune('\n')
			}
			lastSpace = true
			continue
		}
		if unicode.IsSpace(r) {
			if !lastSpace {
				b.WriteRune(' ')
				lastSpace = true
			}
		} else {
			b.WriteRune(r)
			lastSpace = false
			consecutiveNewlines = 0
		}
	}
	return strings.TrimSpace(b.String())
}
