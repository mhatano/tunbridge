package mail

import (
	"bytes"
	"encoding/base64"
	"html"
	"io"
	"mime"
	"mime/quotedprintable"
	"regexp"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

var (
	reEncodedWord     = regexp.MustCompile(`=\?([^?\s]+)\?([bBqQ])\?([^?]*)\?=`)
	reBetweenWords    = regexp.MustCompile(`\?=[ \t\r\n]+=\?`)
	reMetaCharset1    = regexp.MustCompile(`(?i)<meta\b[^>]*?\bcharset\s*=\s*["']?\s*([a-zA-Z0-9_\-.:]+)\s*["']?`)
	reMetaCharset2    = regexp.MustCompile(`(?i)<\?xml\b[^>]*?\bencoding\s*=\s*["']\s*([a-zA-Z0-9_\-.:]+)\s*["']`)
	reParamCharset    = regexp.MustCompile(`(?i)\bcharset\s*=\s*["']?([^\s"';]+)["']?`)
	reParamBoundary   = regexp.MustCompile(`(?i)\bboundary\s*=\s*(?:"([^"]+)"|'([^']+)'|([^\s;]+))`)
	reUTF7Block       = regexp.MustCompile(`\+([A-Za-z0-9+/]+)-?`)
)

// LookupEncoding resolves an IANA/MIME/HTML charset name (including common aliases and supersets) to an encoding.Encoding.
func LookupEncoding(charset string) encoding.Encoding {
	cs := strings.TrimSpace(charset)
	cs = strings.Trim(cs, `"'`)
	if idx := strings.IndexByte(cs, '*'); idx != -1 {
		// Strip RFC 2231 language tag e.g. UTF-8*ja
		cs = cs[:idx]
	}
	low := strings.ToLower(strings.TrimSpace(cs))
	lowNorm := strings.NewReplacer("_", "-", " ", "-").Replace(low)

	switch lowNorm {
	case "", "utf-8", "utf8", "unicode-1-1-utf-8", "us-ascii", "ascii", "ansi-x3.4-1968", "iso646-us", "cp367", "ibm367":
		return unicode.UTF8
	case "iso-2022-jp", "iso-2022-jp-1", "iso-2022-jp-2", "iso-2022-jp-ms", "cp50220", "cp50221", "cp50222", "ms50220", "ms50221", "jis", "csiso2022jp", "x-iso2022jp", "iso2022jp":
		return japanese.ISO2022JP
	case "shift-jis", "shiftjis", "sjis", "s-jis", "x-sjis", "ms-kanji", "csshiftjis", "windows-31j", "cp932", "ms932", "cswindows31j", "x-ms-cp932":
		return japanese.ShiftJIS
	case "euc-jp", "eucjp", "x-euc-jp", "cseucpkdfmtjapanese", "euc-jp-ms", "cp51932":
		return japanese.EUCJP
	case "gb18030", "gbk", "cp936", "ms936", "windows-936", "gb2312", "gb-2312", "gb-2312-80", "csgb2312", "chinese", "cn-gb", "x-gbk", "iso-ir-58":
		// GB18030 is a superset of GBK and GB2312
		return simplifiedchinese.GB18030
	case "hz-gb-2312", "hz":
		return simplifiedchinese.HZGB2312
	case "big5", "big-5", "big5-hkscs", "cp950", "ms950", "csbig5", "x-x-big5", "cn-big5":
		return traditionalchinese.Big5
	case "euc-kr", "euckr", "cseuckr", "ksc5601", "ks-c-5601-1987", "ks-c-5601-1989", "csksc56011987", "cp949", "ms949", "uhc", "windows-949", "x-windows-949", "iso-2022-kr", "csiso2022kr":
		return korean.EUCKR
	case "koi8-r", "cskoi8r", "koi8":
		return charmap.KOI8R
	case "koi8-u", "koi8-ru":
		return charmap.KOI8U
	case "windows-1251", "cp1251", "x-cp1251":
		return charmap.Windows1251
	case "iso-8859-5", "cyrillic":
		return charmap.ISO8859_5
	case "ibm866", "cp866", "866", "csibm866":
		return charmap.CodePage866
	case "windows-1252", "cp1252", "x-cp1252", "iso-8859-1", "iso-8859-1:1987", "latin1", "l1", "cp819", "ibm819":
		return charmap.Windows1252
	case "iso-8859-15", "latin9":
		return charmap.ISO8859_15
	case "windows-1250", "cp1250", "x-cp1250":
		return charmap.Windows1250
	case "iso-8859-2", "latin2":
		return charmap.ISO8859_2
	case "windows-1253", "cp1253":
		return charmap.Windows1253
	case "iso-8859-7", "greek":
		return charmap.ISO8859_7
	case "windows-1254", "cp1254", "iso-8859-9", "latin5":
		return charmap.Windows1254
	case "windows-1255", "cp1255":
		return charmap.Windows1255
	case "iso-8859-8", "iso-8859-8-i", "hebrew":
		return charmap.ISO8859_8
	case "windows-1256", "cp1256":
		return charmap.Windows1256
	case "iso-8859-6", "arabic":
		return charmap.ISO8859_6
	case "windows-1257", "cp1257":
		return charmap.Windows1257
	case "iso-8859-13":
		return charmap.ISO8859_13
	case "windows-1258", "cp1258":
		return charmap.Windows1258
	case "tis-620", "windows-874", "cp874", "iso-8859-11", "dos-874":
		return charmap.Windows874
	case "macintosh", "mac", "x-mac-roman":
		return charmap.Macintosh
	case "x-mac-cyrillic", "mac-cyrillic":
		return charmap.MacintoshCyrillic
	case "utf-16le", "utf16le":
		return unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	case "utf-16be", "utf16be":
		return unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
	case "utf-16", "utf16", "ucs-2":
		return unicode.UTF16(unicode.BigEndian, unicode.UseBOM)
	}

	if enc, err := htmlindex.Get(low); err == nil && enc != nil {
		return enc
	}
	if enc, err := ianaindex.MIME.Encoding(low); err == nil && enc != nil {
		return enc
	}
	if enc, err := ianaindex.IANA.Encoding(low); err == nil && enc != nil {
		return enc
	}
	return nil
}

// NewWordDecoder returns a mime.WordDecoder configured with our multilingual CharsetReader.
func NewWordDecoder() *mime.WordDecoder {
	return &mime.WordDecoder{
		CharsetReader: func(charset string, input io.Reader) (io.Reader, error) {
			raw, err := io.ReadAll(input)
			if err != nil {
				return nil, err
			}
			decoded := DecodeBytesWithCharset(raw, charset)
			return strings.NewReader(decoded), nil
		},
	}
}

// DecodeMIMEHeader decodes RFC 2047 encoded-words (=?charset?B/Q?...?=), handles adjacent split multi-byte words,
// repairs malformed base64/Q-encoding, auto-detects unencoded non-UTF-8 bytes or ISO-2022 escape sequences,
// and normalizes spoofed directional/zero-width characters.
func DecodeMIMEHeader(v string) string {
	if v == "" {
		return ""
	}

	s := v
	// Unfold CRLF + WSP if any
	s = strings.ReplaceAll(s, "\r\n ", " ")
	s = strings.ReplaceAll(s, "\r\n\t", " ")
	s = strings.ReplaceAll(s, "\n ", " ")
	s = strings.ReplaceAll(s, "\n\t", " ")

	if strings.Contains(s, "=?") {
		s = decodeEncodedWordsRobust(s)
	}

	// Check if remaining string has ISO-2022 escape sequences or non-UTF8 bytes
	if HasISO2022Escapes(s) || !utf8.ValidString(s) {
		s = DetectAndDecodeBytes([]byte(s), "")
	}

	s = NormalizeMultilingualText(s, true)
	return s
}

type encodedToken struct {
	isEncoded bool
	charset   string
	encoding  string
	payload   string
	rawText   string
}

func decodeEncodedWordsRobust(input string) string {
	matches := reEncodedWord.FindAllStringSubmatchIndex(input, -1)
	if len(matches) == 0 {
		return input
	}

	var tokens []encodedToken
	lastEnd := 0
	for _, m := range matches {
		start, end := m[0], m[1]
		if start > lastEnd {
			between := input[lastEnd:start]
			// Per RFC 2047 Section 6.2, linear whitespace between adjacent encoded-words is ignored
			if len(tokens) > 0 && tokens[len(tokens)-1].isEncoded && strings.Trim(between, " \t\r\n") == "" {
				// skip whitespace between encoded-words
			} else {
				tokens = append(tokens, encodedToken{isEncoded: false, rawText: between})
			}
		}
		cs := input[m[2]:m[3]]
		enc := strings.ToUpper(input[m[4]:m[5]])
		pay := input[m[6]:m[7]]
		tokens = append(tokens, encodedToken{
			isEncoded: true,
			charset:   cs,
			encoding:  enc,
			payload:   pay,
			rawText:   input[start:end],
		})
		lastEnd = end
	}
	if lastEnd < len(input) {
		tokens = append(tokens, encodedToken{isEncoded: false, rawText: input[lastEnd:]})
	}

	// Merge adjacent encoded tokens that share the same normalized charset so split multi-byte characters
	// or stateful ISO-2022 sequences across adjacent words decode seamlessly.
	var out strings.Builder
	i := 0
	for i < len(tokens) {
		if !tokens[i].isEncoded {
			out.WriteString(tokens[i].rawText)
			i++
			continue
		}
		baseCS := normalizeCharsetKey(tokens[i].charset)
		var combinedBytes []byte
		j := i
		for j < len(tokens) && tokens[j].isEncoded && normalizeCharsetKey(tokens[j].charset) == baseCS {
			b, ok := decodeSingleWordPayload(tokens[j].encoding, tokens[j].payload, baseCS)
			if !ok {
				b = []byte(tokens[j].rawText)
			}
			combinedBytes = append(combinedBytes, b...)
			j++
		}
		out.WriteString(DecodeBytesWithCharset(combinedBytes, tokens[i].charset))
		i = j
	}
	return out.String()
}

func normalizeCharsetKey(cs string) string {
	if idx := strings.IndexByte(cs, '*'); idx != -1 {
		cs = cs[:idx]
	}
	return strings.ToLower(strings.TrimSpace(cs))
}

func decodeSingleWordPayload(enc, payload, charsetKey string) ([]byte, bool) {
	switch enc {
	case "B":
		return decodeBase64Tolerant(payload, charsetKey == "utf-8" || charsetKey == "utf8")
	case "Q":
		// In RFC 2047 Q-encoding, '_' represents space (0x20)
		qStr := strings.ReplaceAll(payload, "_", " ")
		b, err := io.ReadAll(quotedprintable.NewReader(strings.NewReader(qStr)))
		if err != nil && len(b) == 0 {
			return decodeQPTolerant(qStr), true
		}
		return b, true
	default:
		return nil, false
	}
}

// decodeBase64Tolerant decodes Base64 even if padding '=' is missing, whitespace is present,
// or a single stray character was injected into a UTF-8 base64 stream (as seen in spam subjects).
func decodeBase64Tolerant(s string, expectUTF8 bool) ([]byte, bool) {
	var cleanBuf strings.Builder
	cleanBuf.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '+' || c == '/' {
			cleanBuf.WriteByte(c)
		} else if c == '-' {
			cleanBuf.WriteByte('+')
		} else if c == '_' {
			cleanBuf.WriteByte('/')
		}
	}
	clean := cleanBuf.String()
	if len(clean) == 0 {
		return nil, true
	}

	// Wait: if len(clean)%4 == 1, in valid Base64 a 1-char remainder is impossible (6 bits < 1 byte).
	// This happens when a 1-char typo was inserted into the base64 string or appended at the end.
	if len(clean)%4 == 1 && expectUTF8 && len(clean) > 8 {
		if repaired, ok := tryRepairShiftedUTF8Base64(clean); ok {
			return repaired, true
		}
		// Fallback: drop trailing 1 char
		clean = clean[:len(clean)-1]
	}

	if rem := len(clean) % 4; rem != 0 {
		clean += strings.Repeat("=", 4-rem)
	}
	b, err := base64.StdEncoding.DecodeString(clean)
	if err == nil {
		return b, true
	}
	return b, len(b) > 0
}

// tryRepairShiftedUTF8Base64 finds a single stray character in a Base64 string of length 4k+1
// that caused subsequent 4-char blocks to misalign and break UTF-8 validity.
func tryRepairShiftedUTF8Base64(clean string) ([]byte, bool) {
	bestInvalid := 1 << 30
	bestScore := -1e18
	var bestBytes []byte

	// Test removing 1 character at candidate positions to restore 4-char alignment
	for pos := 0; pos < len(clean); pos++ {
		cand := clean[:pos] + clean[pos+1:]
		b, err := base64.StdEncoding.DecodeString(cand)
		if err != nil {
			continue
		}
		inv := countInvalidUTF8Bytes(b)
		sc := scoreCandidateDecoded(string(b), "utf-8", "utf-8")
		if inv < bestInvalid || (inv == bestInvalid && sc >= bestScore) {
			bestInvalid = inv
			bestScore = sc
			bestBytes = b
		}
	}
	if bestBytes != nil && bestInvalid <= 3 {
		return bestBytes, true
	}
	return nil, false
}

func countInvalidUTF8Bytes(b []byte) int {
	invalid := 0
	for len(b) > 0 {
		r, size := utf8.DecodeRune(b)
		if r == utf8.RuneError && size == 1 {
			invalid++
		}
		b = b[size:]
	}
	return invalid
}

func decodeQPTolerant(s string) []byte {
	var out bytes.Buffer
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			if i+2 < len(s) {
				if s[i+1] == '\r' && s[i+2] == '\n' {
					i += 2
					continue
				}
				h1 := fromHexChar(s[i+1])
				h2 := fromHexChar(s[i+2])
				if h1 >= 0 && h2 >= 0 {
					out.WriteByte(byte(h1<<4 | h2))
					i += 2
					continue
				}
			} else if i+1 < len(s) && s[i+1] == '\n' {
				i++
				continue
			}
		}
		out.WriteByte(s[i])
	}
	return out.Bytes()
}

func fromHexChar(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'A' && c <= 'F':
		return int(c - 'A' + 10)
	case c >= 'a' && c <= 'f':
		return int(c - 'a' + 10)
	default:
		return -1
	}
}

// HasISO2022Escapes returns true if s contains 7-bit ISO-2022-JP / JIS / ISO-2022-KR escape sequences.
func HasISO2022Escapes(s string) bool {
	if !strings.Contains(s, "\x1b") {
		return false
	}
	return strings.Contains(s, "\x1b$B") ||
		strings.Contains(s, "\x1b$@") ||
		strings.Contains(s, "\x1b$(D") ||
		strings.Contains(s, "\x1b$A") ||
		strings.Contains(s, "\x1b(B") ||
		strings.Contains(s, "\x1b(J") ||
		strings.Contains(s, "\x1b(I") ||
		strings.Contains(s, "\x1b$)C") ||
		strings.Contains(s, "\x1b$(C")
}

// DecodeISO2022Segments decodes all 7-bit ISO-2022-JP / JIS escape-bracketed segments in raw
// while leaving any surrounding UTF-8 or 8-bit bytes intact!
func DecodeISO2022Segments(raw []byte) []byte {
	if !bytes.Contains(raw, []byte{'\x1b'}) {
		return raw
	}

	var out bytes.Buffer
	out.Grow(len(raw))
	i := 0
	n := len(raw)

	isStartEsc := func(pos int) (int, bool) {
		if pos+2 >= n || raw[pos] != '\x1b' {
			return 0, false
		}
		// \x1b$B, \x1b$@, \x1b$A, \x1b(I
		if (raw[pos+1] == '$' && (raw[pos+2] == 'B' || raw[pos+2] == '@' || raw[pos+2] == 'A')) ||
			(raw[pos+1] == '(' && raw[pos+2] == 'I') {
			return 3, true
		}
		// \x1b$(B, \x1b$(D, \x1b$)C, \x1b$(C
		if pos+3 < n && raw[pos+1] == '$' && (raw[pos+2] == '(' || raw[pos+2] == ')') &&
			(raw[pos+3] == 'B' || raw[pos+3] == 'D' || raw[pos+3] == 'C') {
			return 4, true
		}
		return 0, false
	}

	isEndEsc := func(pos int) (int, bool) {
		if pos+2 < n && raw[pos] == '\x1b' && raw[pos+1] == '(' &&
			(raw[pos+2] == 'B' || raw[pos+2] == 'J' || raw[pos+2] == 'H') {
			return 3, true
		}
		return 0, false
	}

	for i < n {
		if escLen, ok := isStartEsc(i); ok {
			// Collect the 7-bit JIS segment until end escape, newline, or non-ASCII byte
			segStart := i
			j := i + escLen
			endedWithEsc := false
			for j < n {
				if endLen, okEnd := isEndEsc(j); okEnd {
					j += endLen
					endedWithEsc = true
					break
				}
				if subLen, okSub := isStartEsc(j); okSub {
					j += subLen
					continue
				}
				b := raw[j]
				if b == '\r' || b == '\n' || b >= 0x80 || b == '\x1b' {
					break
				}
				j++
			}
			seg := make([]byte, j-segStart)
			copy(seg, raw[segStart:j])
			if !endedWithEsc {
				seg = append(seg, '\x1b', '(', 'B')
			}
			decoded, _, err := transform.Bytes(japanese.ISO2022JP.NewDecoder(), seg)
			if err == nil && len(decoded) > 0 {
				out.Write(decoded)
			} else {
				// Strip escape sequences if decoding partially failed
				cleaned := bytes.ReplaceAll(seg, []byte("\x1b$B"), nil)
				cleaned = bytes.ReplaceAll(cleaned, []byte("\x1b$@"), nil)
				cleaned = bytes.ReplaceAll(cleaned, []byte("\x1b(B"), nil)
				cleaned = bytes.ReplaceAll(cleaned, []byte("\x1b(J"), nil)
				out.Write(cleaned)
			}
			i = j
			continue
		}

		// Strip stray closing escape sequences (\x1b(B, \x1b(J) if preceded by broken JIS
		if endLen, okEnd := isEndEsc(i); okEnd {
			i += endLen
			continue
		}

		out.WriteByte(raw[i])
		i++
	}

	return out.Bytes()
}

// DecodeBytesWithCharset decodes raw bytes using a declared charset, falling back to automatic detection
// when the charset is missing, mislabelled, or produces invalid text.
func DecodeBytesWithCharset(raw []byte, declaredCharset string) string {
	return DetectAndDecodeBytes(raw, declaredCharset)
}

// DetectAndDecodeBytes inspects raw bytes + optional charset hint (and HTML <meta> charset),
// automatically detects the character encoding if unencoded or mislabelled, and returns clean UTF-8.
func DetectAndDecodeBytes(raw []byte, declaredCharset string) string {
	if len(raw) == 0 {
		return ""
	}

	// 1. Check BOMs
	if bytes.HasPrefix(raw, []byte{0xEF, 0xBB, 0xBF}) {
		raw = raw[3:]
	} else if bytes.HasPrefix(raw, []byte{0xFE, 0xFF}) {
		if dec, _, err := transform.Bytes(unicode.UTF16(unicode.BigEndian, unicode.UseBOM).NewDecoder(), raw); err == nil {
			return string(dec)
		}
	} else if bytes.HasPrefix(raw, []byte{0xFF, 0xFE}) {
		if dec, _, err := transform.Bytes(unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder(), raw); err == nil {
			return string(dec)
		}
	}

	// 2. Check BOM-less UTF-16LE / UTF-16BE
	if enc := detectUTF16WithoutBOM(raw); enc != nil {
		if dec, _, err := transform.Bytes(enc.NewDecoder(), raw); err == nil {
			return string(dec)
		}
	}

	// 3. Always decode 7-bit ISO-2022-JP / JIS escape sequences first (works even on mixed UTF-8 + JIS)
	if bytes.Contains(raw, []byte{'\x1b'}) && HasISO2022Escapes(string(raw)) {
		raw = DecodeISO2022Segments(raw)
	}

	// 4. Resolve charset hint from declaredCharset or HTML <meta> tag
	hint := normalizeCharsetKey(declaredCharset)
	if hint == "" || hint == "utf-8" || hint == "utf8" || hint == "us-ascii" || hint == "ascii" || hint == "iso-8859-1" || hint == "latin1" {
		if metaCS := extractCharsetFromMeta(raw); metaCS != "" {
			hint = normalizeCharsetKey(metaCS)
		}
	}

	// 5. Check 7-bit ASCII / UTF-7 / HZ-GB-2312
	hasHighByte := false
	for _, b := range raw {
		if b >= 0x80 {
			hasHighByte = true
			break
		}
	}
	if !hasHighByte {
		if hint == "utf-7" || hint == "unicode-1-1-utf-7" {
			return decodeUTF7(string(raw))
		}
		if hint == "hz-gb-2312" || hint == "hz" || bytes.Contains(raw, []byte("~{")) {
			if dec, _, err := transform.Bytes(simplifiedchinese.HZGB2312.NewDecoder(), raw); err == nil {
				return string(dec)
			}
		}
		return string(raw)
	}

	// 6. Check if raw is already valid UTF-8!
	// Note: genuine multi-byte UTF-8 sequences almost never occur by accident in legacy 8-bit encodings.
	if utf8.Valid(raw) {
		return string(raw)
	}

	// 7. Raw contains non-UTF-8 high bytes (>= 0x80).
	// What if raw is mostly valid UTF-8 with a few corrupt bytes (e.g., broken spam base64 or truncated tail)?
	if isMostlyValidUTF8(raw) && (hint == "" || hint == "utf-8" || hint == "utf8") {
		return sanitizeUTF8(raw)
	}

	// 8. If a specific non-Latin1 charset hint was provided, try it first
	if hint != "" && hint != "utf-8" && hint != "utf8" && hint != "us-ascii" && hint != "ascii" &&
		hint != "iso-8859-1" && hint != "latin1" && hint != "windows-1252" {
		if enc := LookupEncoding(hint); enc != nil && enc != unicode.UTF8 {
			if dec, _, err := transform.Bytes(enc.NewDecoder(), raw); err == nil && utf8.Valid(dec) {
				repl := bytes.Count(dec, []byte("\xef\xbf\xbd"))
				if repl == 0 {
					return string(dec)
				}
			}
		}
	}

	// 9. Run Multi-Language Statistical & Structural Auto-Detection across candidate encodings
	type candidate struct {
		name string
		enc  encoding.Encoding
	}
	candidates := []candidate{
		{"shift_jis", japanese.ShiftJIS},
		{"euc-jp", japanese.EUCJP},
		{"gb18030", simplifiedchinese.GB18030},
		{"big5", traditionalchinese.Big5},
		{"euc-kr", korean.EUCKR},
		{"windows-1251", charmap.Windows1251},
		{"koi8-r", charmap.KOI8R},
		{"windows-1252", charmap.Windows1252},
	}
	if hint != "" {
		if enc := LookupEncoding(hint); enc != nil && enc != unicode.UTF8 {
			already := false
			for _, c := range candidates {
				if c.enc == enc {
					already = true
					break
				}
			}
			if !already {
				candidates = append([]candidate{{hint, enc}}, candidates...)
			}
		}
	}

	// Sample up to 16KB for fast scoring
	sample := raw
	if len(sample) > 16384 {
		sample = sample[:16384]
	}

	bestScore := -1e18
	var bestEncoding encoding.Encoding
	for _, c := range candidates {
		decSample, _, err := transform.Bytes(c.enc.NewDecoder(), sample)
		if err != nil {
			continue
		}
		sc := scoreCandidateDecoded(string(decSample), c.name, hint)
		if sc > bestScore {
			bestScore = sc
			bestEncoding = c.enc
		}
	}

	if bestEncoding != nil {
		if dec, _, err := transform.Bytes(bestEncoding.NewDecoder(), raw); err == nil {
			return string(dec)
		}
	}

	return sanitizeUTF8(raw)
}

func detectUTF16WithoutBOM(raw []byte) encoding.Encoding {
	if len(raw) < 6 || len(raw)%2 != 0 {
		return nil
	}
	checkLen := len(raw)
	if checkLen > 512 {
		checkLen = 512
	}
	evenZero, oddZero := 0, 0
	pairs := checkLen / 2
	for i := 0; i < checkLen; i += 2 {
		if raw[i] == 0 && raw[i+1] >= 0x09 && raw[i+1] < 0x7F {
			evenZero++
		}
		if raw[i+1] == 0 && raw[i] >= 0x09 && raw[i] < 0x7F {
			oddZero++
		}
	}
	if evenZero > pairs/2 && oddZero == 0 {
		return unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
	}
	if oddZero > pairs/2 && evenZero == 0 {
		return unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	}
	return nil
}

func extractCharsetFromMeta(raw []byte) string {
	head := raw
	if len(head) > 4096 {
		head = head[:4096]
	}
	if m := reMetaCharset1.FindSubmatch(head); len(m) > 1 {
		return string(m[1])
	}
	if m := reMetaCharset2.FindSubmatch(head); len(m) > 1 {
		return string(m[1])
	}
	return ""
}

func isMostlyValidUTF8(raw []byte) bool {
	validMultiByteRunes := 0
	invalidBytes := 0
	b := raw
	for len(b) > 0 {
		r, size := utf8.DecodeRune(b)
		if r == utf8.RuneError && size == 1 {
			invalidBytes++
		} else if size >= 2 {
			validMultiByteRunes++
		}
		b = b[size:]
	}
	return validMultiByteRunes >= 8 && validMultiByteRunes > invalidBytes*4
}

func sanitizeUTF8(raw []byte) string {
	return strings.ToValidUTF8(string(raw), "")
}

var (
	commonSimplifiedChinese = buildRuneSet("的是了不在人有我他这中大来上国个到说们为子和你地出道也时年得就那要下以生会自着去之过家学对可里后小么心多天而能好都然没日于起还发成事只作当想看文无开手十用主行方又如前所本见经头面公同三已老从动两长知民样现分将外但身些与高意进把法此实回二理美点月明其种声全工己话儿者向情部正名定女问力机给等几很业最间新什打便位因重被走电四第门相次东政海口使教西再平真听世气信北少关并内加化由却代军产入先山五太水万市眼体别处总才场师书比住员九笑性通目华报立马命张活难神数件安表原车白应路期叫死常提感金何更反合放做系计或司利受光王果亲界及今京务制解各任至清物台象记边共风战干接它许题商品発送価格品質購入無料会員登録您好感谢使用服订单已经货请录个人中心查看物流信息电子票")
	commonTraditionalChinese = buildRuneSet("的在不人有我他這為之大來以個中上們到說國時要就出會可也你對生能而子那得於著下自後年過發作裡用道行所然家種事成方多經麼去法學如都同現當沒動面起看定天分還進好小部其些主樣理心她本前開但因只從想實日軍者意無力它與長把機十民第公此已工使情明性全三又關點正業外將兩高間由問很最重並物手應戰向頭文體政美相見被利什二等產或新己制身果加西斯月話合回特代內信表化老給世位次度門任常先海通教兒原東聲提立及比員解水名真論處走義各入幾口認條平系氣題活爾更別打女變四神總何電數安少報才結反受目太量再感建務做接必場件計管期市直德資命山金指克許統區保至隊形社便空決治展馬科司五基眼書非則聽白卻界達光放強即像難且權思王象完設式色路記南品住告類求據程北邊死張該交規萬取拉格望覺術領共確傳師觀清今切院讓識候帶導爭運笑飛風步改收根干造言聯持組每濟車親極林服快辦議往元英士證近失轉夫令准布始怎呢存未遠叫台單影具羅字愛擊流備兵連調深商算質團集百需價花黨華城石級整府離況亞請技際約示復病息究線似官火斷精滿支視消越器容照須九增研寫稱企八功嗎包片史委乎查輕易早曾除農找裝廣顯吧阿李標談吃圖念六引歷首醫局突專費號盡另周較注語僅考落青隨選列武紅響雖推勢參希古眾構房半節土投某案維革划敵致陳律足態護七責訴客戶網銀行確認帳單明細與轉紀錄")
	commonJapaneseKanji = buildRuneSet("日一国会人年大十二本中長出三同時政事自行社見月分議後前民生連五発間対上部東者党地合市業内相方四定今回新場金員九入選立開手米力学問高代明実円関決子動京全目表戦経通外最言氏現理調体化田当八六約主題下首意法不来作性的要用制治度務強気小七成期公持野協取都和統以機平総加山思家話世受区領保係済指べて確認案内支払口座振替利用明細請求金額登録解除停止重要至急当選無料連絡")
	commonKoreanHangul = buildRuneSet("이그저것수있하되년월일시분초에의가을를은는로고다한지사기대정리자전생국우리문보성원주동구학민해법경실만개중명제여발업상과부방원품게점내통영비교신관도력회마하서아어오우으음니다습까요세안녕감사메일확인결제계좌고객님역과체결과를안내해드립부탁")
)

func buildRuneSet(s string) map[rune]bool {
	m := make(map[rune]bool, len(s))
	for _, r := range s {
		m[r] = true
	}
	return m
}

func scoreCandidateDecoded(decoded, encName, hint string) float64 {
	totalNonASCII := 0
	score := 0.0
	hiragana := 0
	katakana := 0
	halfKana := 0
	commonHangul := 0
	rareHangul := 0
	replCount := 0

	for _, r := range decoded {
		if r < 0x80 {
			if r < 0x20 && r != '\t' && r != '\n' && r != '\r' {
				score -= 20.0
			}
			continue
		}
		totalNonASCII++
		if r == '\uFFFD' {
			replCount++
			score -= 60.0
			continue
		}
		if r >= 0x7F && r <= 0x9F {
			score -= 35.0
			continue
		}
		// Private Use Area or unassigned/weird symbols
		if r >= 0xE000 && r <= 0xF8FF {
			score -= 25.0
			continue
		}

		switch {
		case r >= 0x3041 && r <= 0x3096: // Hiragana
			hiragana++
			score += 5.5
		case (r >= 0x30A1 && r <= 0x30FA) || r == 0x30FC: // Full-width Katakana
			katakana++
			score += 4.5
		case r >= 0xFF61 && r <= 0xFF9F: // Half-width Katakana (often false-positive in Shift_JIS on EUC/GBK/Big5)
			halfKana++
			score -= 2.5
		case r == '、' || r == '。' || r == '，' || r == '「' || r == '」' || r == '『' || r == '』' || r == '【' || r == '】' || r == '・' || r == '（' || r == '）' || r == '〜' || r == '！' || r == '？' || r == '：' || r == '；' || r == '　':
			score += 3.5
		case r >= 0x4E00 && r <= 0x9FFF: // CJK Unified Ideographs
			score += 1.8
			if (encName == "shift_jis" || encName == "euc-jp") && commonJapaneseKanji[r] {
				score += 2.8
			} else if encName == "gb18030" && commonSimplifiedChinese[r] {
				score += 3.5
			} else if encName == "big5" && commonTraditionalChinese[r] {
				score += 3.5
			}
		case r >= 0xAC00 && r <= 0xD7A3: // Korean Hangul Syllables
			if commonKoreanHangul[r] {
				commonHangul++
				score += 6.0
			} else {
				rareHangul++
				score += 0.3
			}
		case r >= 0x0430 && r <= 0x044F || r == 0x0451: // Lowercase Russian Cyrillic
			score += 3.2
		case r >= 0x0410 && r <= 0x042F || r == 0x0401: // Uppercase Russian Cyrillic
			score += 1.4
		case r >= 0x00C0 && r <= 0x00FF: // Western Latin-1 letters
			score += 1.2
		case r >= 0x0100 && r <= 0x024F: // Latin Extended
			score += 1.2
		case r >= 0x0600 && r <= 0x06FF: // Arabic
			score += 2.5
		case r >= 0x0E00 && r <= 0x0E7F: // Thai
			score += 2.5
		case r >= 0x0370 && r <= 0x03FF: // Greek
			score += 2.5
		case r >= 0x0590 && r <= 0x05FF: // Hebrew
			score += 2.5
		default:
			// Box drawing, rare symbols, etc. often produced by single-byte Windows-125x on CJK bytes
			score -= 1.0
		}
	}

	if totalNonASCII == 0 {
		return 0
	}

	avg := score / float64(totalNonASCII)

	// Bonus for natural Japanese mix (Hiragana/Katakana + Kanji)
	if (encName == "shift_jis" || encName == "euc-jp") && (hiragana+katakana) > 0 && halfKana == 0 {
		avg += 1.5
	}
	// Bonus for Korean EUC-KR only when genuine common Hangul syllables are present
	if encName == "euc-kr" {
		if commonHangul > totalNonASCII/4 {
			avg += 2.0
		} else if rareHangul > commonHangul*2 {
			// False-positive CP949 extended Hangul produced from GB18030/Big5/Shift_JIS bytes
			avg -= 2.5
		}
	}
	// Bonus if matches explicit hint
	if hint != "" && strings.HasPrefix(encName, hint[:min(len(hint), 4)]) {
		avg += 1.5
	}
	// Penalize single-byte fallback encodings when multi-byte decodes cleanly with zero errors
	if encName == "windows-1252" || encName == "windows-1251" || encName == "koi8-r" {
		avg -= 0.4
	}
	if replCount > 0 {
		avg -= float64(replCount) * 5.0
	}

	return avg
}

// decodeUTF7 decodes RFC 2152 UTF-7 modified base64 sequences (+...-).
func decodeUTF7(s string) string {
	if !strings.Contains(s, "+") {
		return s
	}
	s = strings.ReplaceAll(s, "+-", "\x00PLUS\x00")
	res := reUTF7Block.ReplaceAllStringFunc(s, func(m string) string {
		inner := strings.TrimPrefix(m, "+")
		inner = strings.TrimSuffix(inner, "-")
		if rem := len(inner) % 4; rem != 0 {
			inner += strings.Repeat("=", 4-rem)
		}
		b, err := base64.StdEncoding.DecodeString(inner)
		if err != nil || len(b) < 2 {
			return m
		}
		u16 := make([]uint16, len(b)/2)
		for i := 0; i < len(u16); i++ {
			u16[i] = uint16(b[2*i])<<8 | uint16(b[2*i+1])
		}
		return string(utf16.Decode(u16))
	})
	return strings.ReplaceAll(res, "\x00PLUS\x00", "+")
}

// NormalizeMultilingualText resolves bidirectional Right-To-Left Overrides (\u202e...\u202c)
// and removes invisible zero-width spoofing characters (\u200b, \u200c, \u200d, \u2060, \u2063, \ufeff)
// so spoofed/obfuscated subjects, sender names, and bodies display and tokenize cleanly.
func NormalizeMultilingualText(s string, isSingleLineHeader bool) string {
	if s == "" {
		return ""
	}

	// Unescape HTML entities if present in header/text
	if strings.Contains(s, "&#") || strings.Contains(s, "&amp;") || strings.Contains(s, "&lt;") || strings.Contains(s, "&gt;") || strings.Contains(s, "&quot;") || strings.Contains(s, "&nbsp;") {
		s = html.UnescapeString(s)
	}

	// Resolve Right-To-Left Override (\u202e) blocks into normal logical reading order
	if strings.ContainsRune(s, '\u202e') {
		s = resolveRLOBlocks(s)
	}

	// Strip zero-width & directional formatting characters and combining spam noise
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\u200b', '\u200c', '\u200d', '\u200e', '\u200f',
			'\u202a', '\u202b', '\u202c', '\u202d', '\u202e',
			'\u2060', '\u2061', '\u2062', '\u2063', '\u2064',
			'\u2066', '\u2067', '\u2068', '\u2069',
			'\ufeff', '\u00ad', '\u0330', '\ufffd':
			continue
		}
		if isSingleLineHeader && (r == '\r' || r == '\n') {
			b.WriteRune(' ')
			continue
		}
		b.WriteRune(r)
	}
	res := b.String()
	if isSingleLineHeader {
		return strings.TrimSpace(res)
	}
	return res
}

var bracketMirror = map[rune]rune{
	'(': ')', ')': '(',
	'[': ']', ']': '[',
	'{': '}', '}': '{',
	'<': '>', '>': '<',
	'（': '）', '）': '（',
	'【': '】', '】': '【',
	'「': '」', '」': '「',
	'『': '』', '』': '『',
	'［': '］', '］': '［',
	'≪': '≫', '≫': '≪',
	'〈': '〉', '〉': '〈',
	'《': '》', '》': '《',
}

func resolveRLOBlocks(s string) string {
	// First strip zero-width joiners inside so RLO reversal operates on clean characters
	var clean []rune
	for _, r := range s {
		switch r {
		case '\u200b', '\u200c', '\u200d', '\u2060', '\u2063', '\ufeff':
			continue
		default:
			clean = append(clean, r)
		}
	}

	var out []rune
	i := 0
	for i < len(clean) {
		if clean[i] == '\u202e' {
			j := i + 1
			for j < len(clean) && clean[j] != '\u202c' && clean[j] != '\n' && clean[j] != '\r' {
				j++
			}
			segment := clean[i+1 : j]
			for k := len(segment) - 1; k >= 0; k-- {
				r := segment[k]
				if m, ok := bracketMirror[r]; ok {
					r = m
				}
				out = append(out, r)
			}
			if j < len(clean) && clean[j] == '\u202c' {
				j++
			}
			i = j
			continue
		}
		out = append(out, clean[i])
		i++
	}
	return string(out)
}
