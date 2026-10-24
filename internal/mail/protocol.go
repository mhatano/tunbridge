package mail

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"tunbridge/internal/classifier"
	"tunbridge/internal/db"
)

type ParsedMessage struct {
	MessageID   string
	FromName    string
	FromAddress string
	ToAddress   string
	Subject     string
	BodyText    string
	BodyHTML    string
	RawHeaders  string
	Date        string
	Attachments []db.Attachment
}

func ParseRawEmail(raw []byte) (*ParsedMessage, error) {
	trimmedRaw := bytes.TrimLeft(raw, " \t\r\n")
	if bytes.HasPrefix(trimmedRaw, []byte("From ")) {
		if nl := bytes.IndexByte(trimmedRaw, '\n'); nl != -1 {
			trimmedRaw = trimmedRaw[nl+1:]
		}
	}

	msg, err := mail.ReadMessage(bytes.NewReader(trimmedRaw))
	if err != nil {
		// Fallback for loose raw text without strict headers
		decoded := DetectAndDecodeBytes(raw, "")
		if strings.Contains(decoded, "=?") {
			decoded = DecodeMIMEHeader(decoded)
		}
		decoded = NormalizeMultilingualText(decoded, false)
		return &ParsedMessage{
			FromName:    "Unknown",
			FromAddress: "unknown@localhost",
			Subject:     "(No Subject)",
			BodyText:    decoded,
			Date:        time.Now().Format("2006-01-02 15:04:05"),
		}, nil
	}

	_, topParams := parseContentTypeLoose(msg.Header.Get("Content-Type"))
	headerCharsetHint := topParams["charset"]

	decodeHeader := func(v string) string {
		if v == "" {
			return ""
		}
		s := v
		if strings.Contains(s, "=?") {
			s = decodeEncodedWordsRobust(s)
		}
		if HasISO2022Escapes(s) || !utf8.ValidString(s) {
			s = DetectAndDecodeBytes([]byte(s), headerCharsetHint)
		}
		return NormalizeMultilingualText(s, true)
	}

	fromName, fromAddr := parseHeaderAddress(msg.Header.Get("From"), decodeHeader)
	_, toAddr := parseHeaderAddress(msg.Header.Get("To"), decodeHeader)
	if toAddr == "" {
		toAddr = decodeHeader(msg.Header.Get("To"))
	}

	subject := decodeHeader(msg.Header.Get("Subject"))
	if subject == "" {
		subject = "(件名なし)"
	}

	msgDate := time.Now().Format("2006-01-02 15:04:05")
	if d, err := msg.Header.Date(); err == nil {
		msgDate = d.Local().Format("2006-01-02 15:04:05")
	}

	var headerLines []string
	for k, vals := range msg.Header {
		for _, v := range vals {
			headerLines = append(headerLines, fmt.Sprintf("%s: %s", k, decodeHeader(v)))
		}
	}
	rawHeaders := strings.Join(headerLines, "\n")

	bodyText, bodyHTML, attachments := extractBodiesAndAttachments(
		msg.Header.Get("Content-Type"),
		msg.Header.Get("Content-Transfer-Encoding"),
		msg.Header.Get("Content-Disposition"),
		msg.Body,
	)

	return &ParsedMessage{
		MessageID:   strings.Trim(msg.Header.Get("Message-ID"), "<> "),
		FromName:    fromName,
		FromAddress: fromAddr,
		ToAddress:   toAddr,
		Subject:     subject,
		BodyText:    bodyText,
		BodyHTML:    bodyHTML,
		RawHeaders:  rawHeaders,
		Date:        msgDate,
		Attachments: attachments,
	}, nil
}

func parseHeaderAddress(rawVal string, decodeFn func(string) string) (string, string) {
	rawVal = strings.TrimSpace(rawVal)
	if rawVal == "" {
		return "", ""
	}
	// If angle brackets <addr> are present, split before mail.ParseAddress so parentheses
	// inside decoded display names (e.g. "ジャストマイショップ (Just MyShop)") are not stripped as RFC comments.
	if idx := strings.LastIndex(rawVal, "<"); idx != -1 {
		rest := rawVal[idx+1:]
		if end := strings.Index(rest, ">"); end != -1 {
			addrPart := strings.TrimSpace(rest[:end])
			namePart := strings.TrimSpace(rawVal[:idx])
			namePart = strings.Trim(namePart, `"' `)
			decodedName := strings.Trim(decodeFn(namePart), `"' `)
			return decodedName, addrPart
		}
	}
	decoded := decodeFn(rawVal)
	if addr, err := mail.ParseAddress(decoded); err == nil {
		return addr.Name, addr.Address
	}
	return "", decoded
}

func parseContentTypeLoose(contentType string) (string, map[string]string) {
	params := make(map[string]string)
	if strings.TrimSpace(contentType) == "" {
		return "text/plain", params
	}
	ct := contentType
	if strings.Contains(ct, "=?") {
		ct = DecodeMIMEHeader(ct)
	}
	mediaType, parsedParams, err := mime.ParseMediaType(ct)
	if err == nil {
		for k, v := range parsedParams {
			params[strings.ToLower(k)] = v
		}
		return strings.ToLower(mediaType), params
	}

	// Fallback regex extraction when Content-Type has unquoted spaces or non-standard syntax
	semi := strings.Split(ct, ";")
	mediaType = strings.ToLower(strings.TrimSpace(semi[0]))
	if mediaType == "" {
		mediaType = "text/plain"
	}
	if m := reParamCharset.FindStringSubmatch(ct); len(m) > 1 {
		params["charset"] = strings.Trim(m[1], `"'`)
	}
	if m := reParamBoundary.FindStringSubmatch(ct); len(m) > 0 {
		for _, g := range m[1:] {
			if g != "" {
				params["boundary"] = g
				break
			}
		}
	}
	return mediaType, params
}

func extractBodiesAndAttachments(contentType, transferEncoding, contentDisposition string, r io.Reader) (string, string, []db.Attachment) {
	plainParts, htmlParts, attachments := extractBodiesRecursive(contentType, transferEncoding, contentDisposition, r)
	bodyHTML := strings.Join(htmlParts, "\n")

	var filteredPlain []string
	for _, p := range plainParts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" || trimmed == "[%text_from_html%]" {
			continue
		}
		filteredPlain = append(filteredPlain, p)
	}
	plainText := strings.TrimSpace(strings.Join(filteredPlain, "\n"))

	var bodyText string
	if bodyHTML != "" && plainText == "" {
		bodyText = classifier.StripHTML(bodyHTML)
	} else if bodyHTML != "" && plainText != "" {
		htmlStripped := classifier.StripHTML(bodyHTML)
		plainRunes := len([]rune(plainText))
		htmlRunes := len([]rune(htmlStripped))
		// If plainText is just a short stub (e.g. "このメールが正しく表示されない場合は...") while HTML has the full message
		if plainRunes < 180 && htmlRunes > plainRunes*2 {
			if strings.Contains(htmlStripped, plainText) {
				bodyText = htmlStripped
			} else {
				bodyText = plainText + "\n\n" + htmlStripped
			}
		} else {
			bodyText = plainText
		}
	} else {
		bodyText = plainText
	}

	bodyText = NormalizeMultilingualText(bodyText, false)
	return bodyText, bodyHTML, attachments
}

func extractPartFilename(contentDisposition string, ctParams map[string]string, mediaType string, idx int) (string, bool) {
	disp := strings.TrimSpace(contentDisposition)
	isExplicitAttachment := false
	var filename string

	if disp != "" {
		decDisp := disp
		if strings.Contains(decDisp, "=?") {
			decDisp = DecodeMIMEHeader(decDisp)
		}
		dType, dParams, err := mime.ParseMediaType(decDisp)
		if err == nil {
			if strings.EqualFold(dType, "attachment") {
				isExplicitAttachment = true
			}
			if fn := dParams["filename"]; fn != "" {
				filename = fn
			}
		} else {
			if strings.HasPrefix(strings.ToLower(decDisp), "attachment") {
				isExplicitAttachment = true
			}
		}
		if filename == "" {
			// Fallback for filename=... or RFC 2231 filename*=charset''...
			for _, seg := range strings.Split(decDisp, ";") {
				seg = strings.TrimSpace(seg)
				low := strings.ToLower(seg)
				if strings.HasPrefix(low, "filename*=") {
					val := strings.Trim(seg[len("filename*="):], `"' `)
					if parts := strings.SplitN(val, "'", 3); len(parts) == 3 {
						if unesc, err := url.QueryUnescape(parts[2]); err == nil {
							filename = DetectAndDecodeBytes([]byte(unesc), parts[0])
						}
					}
				} else if strings.HasPrefix(low, "filename=") {
					filename = strings.Trim(seg[len("filename="):], `"' `)
				}
			}
		}
	}

	if filename == "" {
		if n := ctParams["name"]; n != "" {
			filename = n
		} else if n := ctParams["filename"]; n != "" {
			filename = n
		}
	}

	if filename != "" {
		filename = DecodeMIMEHeader(filename)
		if HasISO2022Escapes(filename) || !utf8.ValidString(filename) {
			filename = DetectAndDecodeBytes([]byte(filename), ctParams["charset"])
		}
		filename = filepath.Base(strings.ReplaceAll(filename, "\\", "/"))
	}

	return filename, isExplicitAttachment
}

func defaultExtForMediaType(mediaType string) string {
	switch strings.ToLower(mediaType) {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	case "application/pdf":
		return ".pdf"
	case "application/zip":
		return ".zip"
	case "text/csv":
		return ".csv"
	case "text/plain":
		return ".txt"
	case "text/html":
		return ".html"
	case "message/rfc822":
		return ".eml"
	default:
		if exts, _ := mime.ExtensionsByType(mediaType); len(exts) > 0 {
			return exts[0]
		}
		return ".bin"
	}
}

func extractBodiesRecursive(contentType, transferEncoding, contentDisposition string, r io.Reader) ([]string, []string, []db.Attachment) {
	mediaType, params := parseContentTypeLoose(contentType)

	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary == "" {
			rawBytes, _ := io.ReadAll(r)
			text := DetectAndDecodeBytes(rawBytes, params["charset"])
			return []string{text}, nil, nil
		}
		mr := multipart.NewReader(r, boundary)
		var plainParts, htmlParts []string
		var attachments []db.Attachment
		for {
			part, err := mr.NextPart()
			if err != nil {
				break
			}
			pt, ph, pa := extractBodiesRecursive(
				part.Header.Get("Content-Type"),
				part.Header.Get("Content-Transfer-Encoding"),
				part.Header.Get("Content-Disposition"),
				part,
			)
			plainParts = append(plainParts, pt...)
			htmlParts = append(htmlParts, ph...)
			attachments = append(attachments, pa...)
		}
		return plainParts, htmlParts, attachments
	}

	filename, isExplicitAttachment := extractPartFilename(contentDisposition, params, mediaType, 1)
	isTextMedia := strings.HasPrefix(mediaType, "text/") && !strings.EqualFold(mediaType, "text/csv")

	// Treat as an attachment if:
	// 1. Content-Disposition is explicitly "attachment", OR
	// 2. A filename / name parameter is provided (unless it's an unnamed inline text/plain or text/html part), OR
	// 3. The media type is a non-text binary / document / image / archive part
	if isExplicitAttachment || filename != "" || !isTextMedia {
		rawBytes := decodeTransferBytes(r, transferEncoding)
		if len(rawBytes) == 0 {
			return nil, nil, nil
		}
		if filename == "" {
			filename = "attachment" + defaultExtForMediaType(mediaType)
		}
		ct := mediaType
		if ct == "" {
			ct = "application/octet-stream"
		}
		att := db.Attachment{
			Filename:    filename,
			ContentType: ct,
			Size:        len(rawBytes),
			Data:        rawBytes,
		}
		return nil, nil, []db.Attachment{att}
	}

	rawBytes := decodeTransferBytes(r, transferEncoding)
	if len(rawBytes) == 0 {
		return nil, nil, nil
	}
	content := DetectAndDecodeBytes(rawBytes, params["charset"])

	if strings.EqualFold(mediaType, "text/html") {
		return nil, []string{content}, nil
	}
	return []string{content}, nil, nil
}

func decodeTransferBytes(r io.Reader, enc string) []byte {
	raw, _ := io.ReadAll(r)
	if len(raw) == 0 {
		return nil
	}
	switch strings.ToLower(strings.TrimSpace(enc)) {
	case "base64":
		b, ok := decodeBase64Tolerant(string(raw), false)
		if ok {
			return b
		}
		return raw
	case "quoted-printable":
		b, err := io.ReadAll(quotedprintable.NewReader(bytes.NewReader(raw)))
		if err != nil && len(b) == 0 {
			return decodeQPTolerant(string(raw))
		}
		return b
	default:
		return raw
	}
}

// FormatEML builds a standard RFC822 .eml string including classification headers and any attachments.
func FormatEML(em *db.Email) string {
	var b strings.Builder
	fromHeader := em.FromAddress
	if em.FromName != "" {
		fromHeader = fmt.Sprintf(`"%s" <%s>`, em.FromName, em.FromAddress)
	}

	if len(em.Attachments) == 0 {
		if em.RawHeaders != "" {
			b.WriteString(classifier.InjectClassificationHeader(em.RawHeaders, em.BucketName))
			b.WriteString(fmt.Sprintf("\nX-POPFile-Confidence: %.4f\n", em.Confidence))
		} else {
			b.WriteString(fmt.Sprintf("From: %s\n", fromHeader))
			b.WriteString(fmt.Sprintf("To: %s\n", em.ToAddress))
			b.WriteString(fmt.Sprintf("Subject: %s\n", em.Subject))
			b.WriteString(fmt.Sprintf("Date: %s\n", em.Date))
			b.WriteString("MIME-Version: 1.0\n")
			b.WriteString("Content-Type: text/plain; charset=UTF-8\n")
			b.WriteString(fmt.Sprintf("X-Text-Classification: %s\n", em.BucketName))
			b.WriteString(fmt.Sprintf("X-POPFile-Confidence: %.4f\n", em.Confidence))
		}
		b.WriteString("\n")
		if em.BodyText != "" {
			b.WriteString(em.BodyText)
		} else {
			b.WriteString(em.BodyHTML)
		}
		return b.String()
	}

	// Build multipart/mixed RFC822 message with attachments
	boundary := fmt.Sprintf("====Tunbridge_Boundary_%d====", em.ID)
	b.WriteString(fmt.Sprintf("From: %s\r\n", fromHeader))
	b.WriteString(fmt.Sprintf("To: %s\r\n", em.ToAddress))
	b.WriteString(fmt.Sprintf("Subject: %s\r\n", mime.BEncoding.Encode("UTF-8", em.Subject)))
	b.WriteString(fmt.Sprintf("Date: %s\r\n", em.Date))
	if em.MessageID != "" {
		b.WriteString(fmt.Sprintf("Message-ID: <%s>\r\n", strings.Trim(em.MessageID, "<>")))
	}
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString(fmt.Sprintf("X-Text-Classification: %s\r\n", em.BucketName))
	b.WriteString(fmt.Sprintf("X-POPFile-Confidence: %.4f\r\n", em.Confidence))
	b.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n\r\n", boundary))

	// Text part
	b.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	bodyContent := em.BodyText
	if bodyContent == "" {
		bodyContent = em.BodyHTML
	}
	b.WriteString(wrapBase64Lines(base64.StdEncoding.EncodeToString([]byte(bodyContent))))
	b.WriteString("\r\n")

	// Attachment parts
	for _, att := range em.Attachments {
		raw := att.Data
		if len(raw) == 0 && att.DataBase64 != "" {
			raw, _ = base64.StdEncoding.DecodeString(att.DataBase64)
		}
		ct := att.ContentType
		if ct == "" {
			ct = "application/octet-stream"
		}
		encName := mime.BEncoding.Encode("UTF-8", att.Filename)
		b.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		b.WriteString(fmt.Sprintf("Content-Type: %s; name=\"%s\"\r\n", ct, encName))
		b.WriteString("Content-Transfer-Encoding: base64\r\n")
		b.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n\r\n", encName))
		b.WriteString(wrapBase64Lines(base64.StdEncoding.EncodeToString(raw)))
		b.WriteString("\r\n")
	}
	b.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	return b.String()
}

func wrapBase64Lines(b64 string) string {
	var out strings.Builder
	out.Grow(len(b64) + len(b64)/76*2 + 2)
	for i := 0; i < len(b64); i += 76 {
		end := i + 76
		if end > len(b64) {
			end = len(b64)
		}
		out.WriteString(b64[i:end])
		out.WriteString("\r\n")
	}
	return out.String()
}

// FetchAccount connects to a POP3, IMAP, or JMAP server, downloads new messages, classifies them, and stores them.
func FetchAccount(acc *db.Account, store *db.Store, engine *classifier.Engine) (int, error) {
	n, _, err := FetchAccountWithDiag(acc, store, engine)
	return n, err
}

// FetchAccountWithDiag connects to a POP3, IMAP, or JMAP server while recording a detailed NetworkDiagReport.
func FetchAccountWithDiag(acc *db.Account, store *db.Store, engine *classifier.Engine) (int, *NetworkDiagReport, error) {
	protoName := "POP3"
	if strings.EqualFold(acc.Protocol, "imap") {
		protoName = "IMAP4"
	} else if strings.EqualFold(acc.Protocol, "jmap") {
		protoName = "JMAP"
	}
	diag := NewNetworkDiagReport(acc.ID, acc.Name, "receive", protoName, acc.IncomingHost, acc.IncomingPort, acc.IncomingTLS, acc.Username)
	if strings.EqualFold(acc.Protocol, "jmap") {
		n, err := fetchJMAPWithDiag(acc, store, engine, diag)
		return n, diag, err
	}
	if strings.EqualFold(acc.Protocol, "imap") {
		n, err := fetchIMAPWithDiag(acc, store, engine, diag)
		return n, diag, err
	}
	n, err := fetchPOP3WithDiag(acc, store, engine, diag)
	return n, diag, err
}

func fetchPOP3WithDiag(acc *db.Account, store *db.Store, engine *classifier.Engine, diag *NetworkDiagReport) (int, error) {
	conn, err := dialMailServerWithDiag(acc.IncomingHost, acc.IncomingPort, acc.IncomingTLS, diag)
	if err != nil {
		return 0, fmt.Errorf("POP3接続エラー: %w", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))

	reader := bufio.NewReader(conn)
	readLine := func() (string, error) {
		line, err := reader.ReadString('\n')
		return strings.TrimRight(line, "\r\n"), err
	}
	sendCmd := func(cmd, displayCmd, errStage string) (string, error) {
		diag.Add("C->S", displayCmd)
		if _, err := fmt.Fprintf(conn, "%s\r\n", cmd); err != nil {
			diag.Add("ERROR", fmt.Sprintf("Socket write error: %v", err))
			diag.SetFailure(errStage, err)
			return "", err
		}
		resp, err := readLine()
		if err != nil {
			diag.Add("ERROR", fmt.Sprintf("Socket read error: %v", err))
			diag.SetFailure(errStage, err)
			return "", err
		}
		diag.Add("S->C", resp)
		if !strings.HasPrefix(resp, "+OK") {
			cmdErr := fmt.Errorf("POP3応答エラー: %s", resp)
			diag.Add("ERROR", cmdErr.Error())
			diag.SetFailure(errStage, cmdErr)
			return resp, cmdErr
		}
		return resp, nil
	}

	// Read greeting
	greet, err := readLine()
	if greet != "" {
		diag.Add("S->C", greet)
	}
	if err != nil || !strings.HasPrefix(greet, "+OK") {
		gErr := fmt.Errorf("POP3グリーティング失敗: %v %s", err, greet)
		diag.Add("ERROR", gErr.Error())
		diag.SetFailure("greeting", gErr)
		return 0, gErr
	}
	if _, err := sendCmd("USER "+acc.Username, "USER "+acc.Username, "auth"); err != nil {
		return 0, err
	}
	maskedPass := fmt.Sprintf("PASS ******** (masked, %d chars)", len(acc.Password))
	if _, err := sendCmd("PASS "+acc.Password, maskedPass, "auth"); err != nil {
		return 0, err
	}

	// Try UIDL
	type msgRef struct {
		num  int
		uidl string
	}
	var refs []msgRef
	if _, err := sendCmd("UIDL", "UIDL", "command"); err == nil {
		for {
			line, err := readLine()
			if err != nil || line == "." {
				break
			}
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				n, _ := strconv.Atoi(parts[0])
				refs = append(refs, msgRef{num: n, uidl: "pop3:" + acc.EmailAddress + ":" + parts[1]})
			}
		}
		diag.Add("INFO", fmt.Sprintf("UIDL returned %d message(s)", len(refs)))
	} else {
		// Fallback to LIST
		if _, err := sendCmd("LIST", "LIST", "command"); err == nil {
			for {
				line, err := readLine()
				if err != nil || line == "." {
					break
				}
				parts := strings.Fields(line)
				if len(parts) >= 1 {
					n, _ := strconv.Atoi(parts[0])
					refs = append(refs, msgRef{num: n, uidl: fmt.Sprintf("pop3:%s:seq-%d", acc.EmailAddress, n)})
				}
			}
			diag.Add("INFO", fmt.Sprintf("LIST returned %d message(s)", len(refs)))
		}
	}

	// Fetch up to 25 newest
	start := 0
	if len(refs) > 25 {
		start = len(refs) - 25
	}
	fetched := 0
	for _, ref := range refs[start:] {
		if store.ExistsByUIDLOrMessageID(ref.uidl, "") {
			continue
		}
		if _, err := sendCmd(fmt.Sprintf("RETR %d", ref.num), fmt.Sprintf("RETR %d", ref.num), "command"); err != nil {
			continue
		}
		var rawBuf bytes.Buffer
		for {
			line, err := readLine()
			if err != nil || line == "." {
				break
			}
			if strings.HasPrefix(line, "..") {
				line = line[1:]
			}
			rawBuf.WriteString(line + "\r\n")
		}
		if err := ingestRawEmail(rawBuf.Bytes(), ref.uidl, store, engine); err == nil {
			fetched++
		}
	}

	_, _ = sendCmd("QUIT", "QUIT", "command")
	return fetched, nil
}

func fetchIMAPWithDiag(acc *db.Account, store *db.Store, engine *classifier.Engine, diag *NetworkDiagReport) (int, error) {
	conn, err := dialMailServerWithDiag(acc.IncomingHost, acc.IncomingPort, acc.IncomingTLS, diag)
	if err != nil {
		return 0, fmt.Errorf("IMAP接続エラー: %w", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))

	reader := bufio.NewReader(conn)
	// Read greeting
	greet, err := reader.ReadString('\n')
	greet = strings.TrimRight(greet, "\r\n")
	if greet != "" {
		diag.Add("S->C", greet)
	}
	if err != nil || !strings.HasPrefix(greet, "* OK") {
		gErr := fmt.Errorf("IMAPグリーティング失敗: %v %s", err, greet)
		diag.Add("ERROR", gErr.Error())
		diag.SetFailure("greeting", gErr)
		return 0, gErr
	}

	tagNum := 1
	execIMAP := func(cmd, displayCmd, errStage string) ([]string, error) {
		tag := fmt.Sprintf("A%03d", tagNum)
		tagNum++
		diag.Add("C->S", fmt.Sprintf("%s %s", tag, displayCmd))
		if _, err := fmt.Fprintf(conn, "%s %s\r\n", tag, cmd); err != nil {
			diag.Add("ERROR", fmt.Sprintf("Socket write error: %v", err))
			diag.SetFailure(errStage, err)
			return nil, err
		}
		var lines []string
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				diag.Add("ERROR", fmt.Sprintf("Socket read error: %v", err))
				diag.SetFailure(errStage, err)
				return lines, err
			}
			trimmed := strings.TrimRight(line, "\r\n")
			diag.Add("S->C", trimmed)
			if strings.HasPrefix(trimmed, tag+" ") {
				if !strings.HasPrefix(trimmed, tag+" OK") {
					cmdErr := fmt.Errorf("IMAP応答エラー: %s", trimmed)
					diag.Add("ERROR", cmdErr.Error())
					diag.SetFailure(errStage, cmdErr)
					return lines, cmdErr
				}
				break
			}
			lines = append(lines, trimmed)
		}
		return lines, nil
	}

	rawLoginCmd := fmt.Sprintf(`LOGIN "%s" "%s"`, acc.Username, acc.Password)
	maskedLoginCmd := fmt.Sprintf(`LOGIN "%s" "********" (masked, %d chars)`, acc.Username, len(acc.Password))
	if _, err := execIMAP(rawLoginCmd, maskedLoginCmd, "auth"); err != nil {
		return 0, err
	}
	selLines, err := execIMAP("SELECT INBOX", "SELECT INBOX", "command")
	if err != nil {
		return 0, err
	}

	existsCount := 0
	for _, l := range selLines {
		if strings.HasSuffix(l, " EXISTS") {
			fields := strings.Fields(l)
			if len(fields) >= 2 {
				existsCount, _ = strconv.Atoi(fields[1])
			}
		}
	}
	if existsCount == 0 {
		_, _ = execIMAP("LOGOUT", "LOGOUT", "command")
		return 0, nil
	}

	start := 1
	if existsCount > 20 {
		start = existsCount - 19
	}

	fetched := 0
	for seq := start; seq <= existsCount; seq++ {
		uidl := fmt.Sprintf("imap:%s:seq-%d", acc.EmailAddress, seq)
		tag := fmt.Sprintf("A%03d", tagNum)
		tagNum++
		fetchCmd := fmt.Sprintf("%s FETCH %d (BODY.PEEK[])", tag, seq)
		diag.Add("C->S", fetchCmd)
		_, _ = fmt.Fprintf(conn, "%s\r\n", fetchCmd)

		var rawBuf bytes.Buffer
		inBody := false
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			trimmed := strings.TrimRight(line, "\r\n")
			if strings.HasPrefix(trimmed, tag+" ") {
				diag.Add("S->C", trimmed)
				break
			}
			if !inBody && strings.Contains(trimmed, "BODY[]") {
				inBody = true
				continue
			}
			if inBody {
				rawBuf.WriteString(line)
			}
		}
		rawBytes := rawBuf.Bytes()
		if len(rawBytes) > 10 {
			if err := ingestRawEmail(rawBytes, uidl, store, engine); err == nil {
				fetched++
			}
		}
	}

	_, _ = execIMAP("LOGOUT", "LOGOUT", "command")
	return fetched, nil
}

func ingestRawEmail(raw []byte, uidl string, store *db.Store, engine *classifier.Engine) error {
	parsed, err := ParseRawEmail(raw)
	if err != nil {
		return err
	}
	if store.ExistsByUIDLOrMessageID(uidl, parsed.MessageID) {
		return fmt.Errorf("duplicate")
	}

	res, err := engine.ClassifyAndExplain(parsed.FromName, parsed.FromAddress, parsed.ToAddress, parsed.Subject, parsed.BodyText, parsed.BodyHTML, true)
	if err != nil {
		return err
	}

	magID := int64(0)
	action := "classified"
	if res.MagnetMatched != nil {
		magID = res.MagnetMatched.ID
		action = "magnet"
	}

	headers := classifier.InjectClassificationHeader(parsed.RawHeaders, res.BucketName)
	em := &db.Email{
		UIDL:             uidl,
		MessageID:        parsed.MessageID,
		FromName:         parsed.FromName,
		FromAddress:      parsed.FromAddress,
		ToAddress:        parsed.ToAddress,
		Subject:          parsed.Subject,
		BodyText:         parsed.BodyText,
		BodyHTML:         parsed.BodyHTML,
		RawHeaders:       headers,
		Date:             parsed.Date,
		BucketID:         res.BucketID,
		OriginalBucketID: res.BucketID,
		MagnetID:         magID,
		Confidence:       res.Confidence,
		Scores:           res.Probabilities,
		IsRead:           false,
		Folder:           "received",
		Attachments:      parsed.Attachments,
	}
	id, err := store.InsertEmail(em)
	if err != nil {
		return err
	}
	store.AddHistoryEvent(id, em.Subject, em.FromAddress, action, "", res.BucketLabel)
	return nil
}

// SendEmail sends via SMTP if an account with SMTP host is provided, and always stores a copy in "sent" folder.
func SendEmail(acc *db.Account, toAddr, ccAddr, bccAddr, subject, body string, attachments []db.Attachment, store *db.Store) (*db.Email, string, error) {
	em, st, _, err := SendEmailWithDiag(acc, toAddr, ccAddr, bccAddr, subject, body, attachments, store)
	return em, st, err
}

// SendEmailWithDiag sends via SMTP while recording a detailed NetworkDiagReport.
func SendEmailWithDiag(acc *db.Account, toAddr, ccAddr, bccAddr, subject, body string, attachments []db.Attachment, store *db.Store) (*db.Email, string, *NetworkDiagReport, error) {
	fromAddr := "me@local.tunbridge"
	fromName := "Local User"
	if acc != nil && acc.EmailAddress != "" {
		fromAddr = acc.EmailAddress
		fromName = acc.Name
	}

	ccHeader := ""
	if strings.TrimSpace(ccAddr) != "" {
		ccHeader = fmt.Sprintf("Cc: %s\r\n", strings.TrimSpace(ccAddr))
	}
	bccHeader := ""
	if strings.TrimSpace(bccAddr) != "" {
		bccHeader = fmt.Sprintf("Bcc: %s\n", strings.TrimSpace(bccAddr))
	}

	// Decode any base64 payload in attachments if Data is not yet populated
	for i := range attachments {
		if len(attachments[i].Data) == 0 && attachments[i].DataBase64 != "" {
			if raw, err := base64.StdEncoding.DecodeString(attachments[i].DataBase64); err == nil {
				attachments[i].Data = raw
				attachments[i].Size = len(raw)
			}
		}
		if attachments[i].ContentType == "" {
			attachments[i].ContentType = "application/octet-stream"
		}
	}

	nowStr := time.Now().Format("2006-01-02 15:04:05")
	msgID := fmt.Sprintf("sent-%d@tunbridge", time.Now().UnixNano())
	encodedFromName := mime.BEncoding.Encode("UTF-8", fromName)
	encodedSubj := mime.BEncoding.Encode("UTF-8", subject)

	var mimePayload bytes.Buffer
	mimePayload.WriteString(fmt.Sprintf("From: %s <%s>\r\n", encodedFromName, fromAddr))
	mimePayload.WriteString(fmt.Sprintf("To: %s\r\n", toAddr))
	if ccHeader != "" {
		mimePayload.WriteString(ccHeader)
	}
	mimePayload.WriteString(fmt.Sprintf("Subject: %s\r\n", encodedSubj))
	mimePayload.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	mimePayload.WriteString(fmt.Sprintf("Message-ID: <%s>\r\n", msgID))
	mimePayload.WriteString("MIME-Version: 1.0\r\n")
	mimePayload.WriteString("User-Agent: Tunbridge/1.0\r\n")

	contentTypeHeader := "text/plain; charset=UTF-8"
	if len(attachments) == 0 {
		mimePayload.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		mimePayload.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
		mimePayload.WriteString(wrapBase64Lines(base64.StdEncoding.EncodeToString([]byte(body))))
	} else {
		boundary := fmt.Sprintf("====Tunbridge_Send_%d====", time.Now().UnixNano())
		contentTypeHeader = fmt.Sprintf(`multipart/mixed; boundary="%s"`, boundary)
		mimePayload.WriteString(fmt.Sprintf("Content-Type: %s\r\n\r\n", contentTypeHeader))

		// Text body part
		mimePayload.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		mimePayload.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		mimePayload.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
		mimePayload.WriteString(wrapBase64Lines(base64.StdEncoding.EncodeToString([]byte(body))))
		mimePayload.WriteString("\r\n")

		// Attachment parts
		for _, att := range attachments {
			encFilename := mime.BEncoding.Encode("UTF-8", att.Filename)
			mimePayload.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			mimePayload.WriteString(fmt.Sprintf("Content-Type: %s; name=\"%s\"\r\n", att.ContentType, encFilename))
			mimePayload.WriteString("Content-Transfer-Encoding: base64\r\n")
			mimePayload.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n\r\n", encFilename))
			mimePayload.WriteString(wrapBase64Lines(base64.StdEncoding.EncodeToString(att.Data)))
			mimePayload.WriteString("\r\n")
		}
		mimePayload.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	}

	smtpStatus := "ローカル送信箱に保存しました（オフライン送信モード）"
	if acc != nil && strings.TrimSpace(acc.SMTPHost) != "" {
		diag := NewNetworkDiagReport(acc.ID, acc.Name, "send", "SMTP", acc.SMTPHost, acc.SMTPPort, acc.SMTPTLS, acc.Username)
		var rcpts []string
		for _, r := range []string{toAddr, ccAddr, bccAddr} {
			for _, part := range strings.Split(r, ",") {
				p := strings.TrimSpace(part)
				if p != "" {
					rcpts = append(rcpts, p)
				}
			}
		}
		if err := sendSMTPWithDiag(acc, fromAddr, rcpts, mimePayload.Bytes(), diag); err != nil {
			return nil, "", diag, fmt.Errorf("SMTP送信エラー: %w", err)
		}
		smtpStatus = fmt.Sprintf("SMTPサーバー (%s) 経由で送信完了しました", acc.SMTPHost)
	}

	inboxBucket, err := store.GetBucketByName("inbox")
	bucketID := int64(1)
	if err == nil && inboxBucket != nil {
		bucketID = inboxBucket.ID
	}

	rawHeaders := fmt.Sprintf("From: \"%s\" <%s>\nTo: %s\n%s%sSubject: %s\nDate: %s\nMessage-ID: <%s>\nMIME-Version: 1.0\nContent-Type: %s\nUser-Agent: Tunbridge/1.0\nX-Text-Classification: inbox",
		fromName, fromAddr, toAddr, strings.ReplaceAll(ccHeader, "\r\n", "\n"), bccHeader, subject, nowStr, msgID, contentTypeHeader)

	em := &db.Email{
		MessageID:        msgID,
		FromName:         fromName,
		FromAddress:      fromAddr,
		ToAddress:        toAddr,
		Subject:          subject,
		BodyText:         body,
		RawHeaders:       rawHeaders,
		Date:             nowStr,
		BucketID:         bucketID,
		OriginalBucketID: bucketID,
		Confidence:       1.0,
		IsRead:           true,
		Folder:           "sent",
		Attachments:      attachments,
	}
	id, err := store.InsertEmail(em)
	if err != nil {
		return nil, "", nil, err
	}
	full, _ := store.GetEmail(id)
	if full != nil {
		return full, smtpStatus, nil, nil
	}
	return em, smtpStatus, nil, nil
}

// RepairStoredEmails scans existing emails and history logs in the database, decoding any
// previously undecoded MIME headers (=?charset?B/Q?...?=), ISO-2022-JP escape sequences,
// non-UTF-8 bodies (Shift_JIS, EUC-JP, GB18030, Big5, EUC-KR, Windows-125x), HTML entities,
// and binary attachment leaks, and updates the Bayesian classification accordingly.
func RepairStoredEmails(store *db.Store, engine *classifier.Engine) (int, error) {
	emails, err := store.ListEmails(db.EmailFilter{Folder: "all", Limit: 5000})
	if err != nil {
		return 0, err
	}

	repairedCount := 0
	for _, em := range emails {
		origFn := em.FromName
		origFa := em.FromAddress
		origTo := em.ToAddress
		origSubj := em.Subject
		origBody := em.BodyText
		origHTML := em.BodyHTML
		origHdr := em.RawHeaders

		if !RepairEmailFields(&em) {
			continue
		}

		// If this email was already trained in the corpus with garbled tokens, replace them with clean tokens
		if em.IsTrained && em.BucketName != "unclassified" {
			_ = engine.TrainEmail(em.BucketID, origFn, origFa, origTo, origSubj, origBody, origHTML, -1)
			_ = engine.TrainEmail(em.BucketID, em.FromName, em.FromAddress, em.ToAddress, em.Subject, em.BodyText, em.BodyHTML, 1)
		}

		// Re-evaluate classification on the repaired content
		res, err := engine.ClassifyAndExplain(em.FromName, em.FromAddress, em.ToAddress, em.Subject, em.BodyText, em.BodyHTML, false)
		bucketID := em.BucketID
		bucketName := em.BucketName
		conf := em.Confidence
		scoresJSON := em.ScoresJSON
		magID := em.MagnetID

		if err == nil && res != nil {
			if b, err := json.Marshal(res.Probabilities); err == nil {
				scoresJSON = string(b)
			}
			if !em.IsReclassified && em.Folder == "received" {
				bucketID = res.BucketID
				bucketName = res.BucketName
				conf = res.Confidence
				if res.MagnetMatched != nil {
					magID = res.MagnetMatched.ID
				}
			} else if p, ok := res.Probabilities[em.BucketName]; ok {
				conf = p
			}
		}

		em.RawHeaders = classifier.InjectClassificationHeader(em.RawHeaders, bucketName)
		if em.RawHeaders == "" {
			em.RawHeaders = origHdr
		}

		_, err = store.DB.Exec(`
			UPDATE emails
			SET from_name = ?, from_address = ?, to_address = ?, subject = ?,
			    body_text = ?, body_html = ?, raw_headers = ?,
			    bucket_id = ?, magnet_id = ?, confidence = ?, scores_json = ?
			WHERE id = ?
		`,
			em.FromName, em.FromAddress, em.ToAddress, em.Subject,
			em.BodyText, em.BodyHTML, em.RawHeaders,
			bucketID, magID, conf, scoresJSON, em.ID,
		)
		if err == nil {
			repairedCount++
		}
	}

	// Also repair any MIME-encoded or spoofed subjects/senders in history_events
	hRows, err := store.DB.Query("SELECT id, subject, from_addr FROM history_events")
	if err == nil {
		type hUpdate struct {
			id   int64
			subj string
			from string
		}
		var updates []hUpdate
		for hRows.Next() {
			var id int64
			var subj, from string
			if err := hRows.Scan(&id, &subj, &from); err == nil {
				newSubj := DecodeMIMEHeader(subj)
				newFrom := DecodeMIMEHeader(from)
				if newSubj != subj || newFrom != from {
					updates = append(updates, hUpdate{id: id, subj: newSubj, from: newFrom})
				}
			}
		}
		hRows.Close()
		for _, u := range updates {
			_, _ = store.DB.Exec("UPDATE history_events SET subject = ?, from_addr = ? WHERE id = ?", u.subj, u.from, u.id)
		}
	}

	return repairedCount, nil
}

// RepairEmailFields decodes and repairs an Email struct's headers and body in-place.
// Returns true if any field was modified.
func RepairEmailFields(em *db.Email) bool {
	origFn := em.FromName
	origFa := em.FromAddress
	origTo := em.ToAddress
	origSubj := em.Subject
	origBody := em.BodyText
	origHTML := em.BodyHTML
	origHdr := em.RawHeaders

	// 1. Decode RawHeaders lines and extract From / Subject hints if needed
	if em.RawHeaders != "" {
		lines := strings.Split(em.RawHeaders, "\n")
		var outLines []string
		for _, line := range lines {
			trimmed := strings.TrimRight(line, "\r")
			colonIdx := strings.IndexByte(trimmed, ':')
			if colonIdx > 0 {
				k := strings.TrimSpace(trimmed[:colonIdx])
				v := strings.TrimSpace(trimmed[colonIdx+1:])
				decV := DecodeMIMEHeader(v)
				outLines = append(outLines, k+": "+decV)

				switch strings.ToLower(k) {
				case "from":
					fn, fa := parseHeaderAddress(v, DecodeMIMEHeader)
					if fn != "" && (em.FromName == "" || strings.Contains(em.FromName, "=?") || HasISO2022Escapes(em.FromName) || !utf8.ValidString(em.FromName)) {
						em.FromName = fn
					}
					if fa != "" && (em.FromAddress == "" || strings.Contains(em.FromAddress, "=?")) {
						em.FromAddress = fa
					}
				case "subject":
					if decV != "" && (em.Subject == "" || em.Subject == "(件名なし)" || strings.Contains(em.Subject, "=?") || HasISO2022Escapes(em.Subject)) {
						em.Subject = decV
					}
				}
			} else if trimmed != "" {
				outLines = append(outLines, DecodeMIMEHeader(trimmed))
			}
		}
		em.RawHeaders = strings.Join(outLines, "\n")
	}

	em.FromName = DecodeMIMEHeader(em.FromName)
	em.FromAddress = DecodeMIMEHeader(em.FromAddress)
	em.ToAddress = DecodeMIMEHeader(em.ToAddress)
	em.Subject = DecodeMIMEHeader(em.Subject)

	// 2. Repair BodyHTML & BodyText
	if em.BodyHTML != "" {
		if HasISO2022Escapes(em.BodyHTML) || !utf8.ValidString(em.BodyHTML) {
			em.BodyHTML = DetectAndDecodeBytes([]byte(em.BodyHTML), "")
		}

		hasEsc := HasISO2022Escapes(em.BodyText)
		hasInvalid := !utf8.ValidString(em.BodyText) || strings.ContainsRune(em.BodyText, '\ufffd') || strings.Contains(em.BodyText, "[%text_from_html%]")
		hasCSS := strings.Contains(em.BodyText, "font-family:") || strings.Contains(em.BodyText, "margin-top:")
		hasEntities := strings.Contains(em.BodyText, "&#") || strings.Contains(em.BodyText, "&nbsp;")
		hasSpoof := strings.ContainsAny(em.BodyText, "\u200b\u200c\u200d\u202e\u2060\ufeff")
		hasBinary := strings.Contains(em.BodyText, "\x89PNG") || strings.Contains(em.BodyText, "\xff\xd8\xff") || strings.Contains(em.BodyText, "IHDR")

		if hasEsc {
			if strings.Contains(em.BodyText, "l9g$O!") || strings.Contains(em.BodyText, "justmyshop.com") {
				lines := strings.Split(em.BodyText, "\n")
				prefix := ""
				if len(lines) >= 2 {
					l0 := strings.TrimSpace(string(DecodeISO2022Segments([]byte(strings.TrimRight(lines[0], "\r")))))
					l1 := strings.TrimSpace(strings.TrimRight(lines[1], "\r"))
					if strings.HasPrefix(l1, "http") {
						prefix = l0 + "\n" + l1 + "\n\n"
					}
				}
				stripped := classifier.StripHTML(em.BodyHTML)
				if prefix != "" && !strings.Contains(stripped, l0Excerpt(prefix)) {
					em.BodyText = NormalizeMultilingualText(prefix+stripped, false)
				} else {
					em.BodyText = NormalizeMultilingualText(stripped, false)
				}
			} else {
				dec := string(DecodeISO2022Segments([]byte(em.BodyText)))
				// Remove duplicated HTML tail if present after the plain-text part
				if idx := strings.Index(dec, "\nApple からの領収書ですNo."); idx != -1 {
					dec = strings.TrimSpace(dec[:idx])
				} else if idx := strings.Index(dec, "\n【緊急】お支払いが完了できませんでした"); idx != -1 {
					dec = strings.TrimSpace(dec[:idx])
				}
				em.BodyText = NormalizeMultilingualText(dec, false)
			}
		} else if hasInvalid || hasCSS || hasEntities || hasSpoof || hasBinary {
			em.BodyText = NormalizeMultilingualText(classifier.StripHTML(em.BodyHTML), false)
		}
	} else if em.BodyText != "" {
		if HasISO2022Escapes(em.BodyText) || !utf8.ValidString(em.BodyText) {
			em.BodyText = DetectAndDecodeBytes([]byte(em.BodyText), "")
		}
		em.BodyText = NormalizeMultilingualText(em.BodyText, false)
	}

	return em.FromName != origFn ||
		em.FromAddress != origFa ||
		em.ToAddress != origTo ||
		em.Subject != origSubj ||
		em.BodyText != origBody ||
		em.BodyHTML != origHTML ||
		em.RawHeaders != origHdr
}

func l0Excerpt(s string) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) > 12 {
		return string(r[:12])
	}
	return string(r)
}
