package mail

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

type NetLogLine struct {
	Time string `json:"time"`
	Tag  string `json:"tag"` // "DNS", "TCP", "TLS", "S->C", "C->S", "ERROR", "INFO"
	Text string `json:"text"`
}

type NetworkDiagReport struct {
	AccountID       int64        `json:"account_id"`
	AccountName     string       `json:"account_name"`
	Direction       string       `json:"direction"` // "receive" or "send"
	Protocol        string       `json:"protocol"`  // "POP3", "IMAP4", "SMTP"
	Host            string       `json:"host"`
	Port            int          `json:"port"`
	UseTLS          bool         `json:"use_tls"`
	Username        string       `json:"username"`
	ResolvedIPs     []string     `json:"resolved_ips"`
	SocketConnected bool         `json:"socket_connected"`
	LocalAddr       string       `json:"local_addr"`
	RemoteAddr      string       `json:"remote_addr"`
	TLSCompleted    bool         `json:"tls_completed"`
	TLSInfo         string       `json:"tls_info"`
	ErrorStage      string       `json:"error_stage"` // "dns", "tcp_connect", "tls_handshake", "greeting", "auth", "command"
	ErrorMessage    string       `json:"error_message"`
	DiagnosisJA     string       `json:"diagnosis_ja"`
	DiagnosisEN     string       `json:"diagnosis_en"`
	Lines           []NetLogLine `json:"lines"`
}

func NewNetworkDiagReport(accID int64, accName, direction, protocol, host string, port int, useTLS bool, username string) *NetworkDiagReport {
	return &NetworkDiagReport{
		AccountID:   accID,
		AccountName: accName,
		Direction:   direction,
		Protocol:    strings.ToUpper(protocol),
		Host:        host,
		Port:        port,
		UseTLS:      useTLS,
		Username:    username,
		Lines:       make([]NetLogLine, 0, 16),
	}
}

func (d *NetworkDiagReport) Add(tag, text string) {
	d.Lines = append(d.Lines, NetLogLine{
		Time: time.Now().Format("15:04:05.000"),
		Tag:  tag,
		Text: text,
	})
}

func (d *NetworkDiagReport) SetFailure(stage string, err error) {
	d.ErrorStage = stage
	if err != nil {
		d.ErrorMessage = err.Error()
	}
	switch stage {
	case "dns":
		d.DiagnosisJA = fmt.Sprintf(
			"サーバーアドレス（ホスト名 %q）の名前解決（DNS）に失敗したため、ソケットを接続できませんでした。サーバーのホスト名に誤字・脱字がないか、またはネットワーク接続を確認してください。",
			d.Host,
		)
		d.DiagnosisEN = fmt.Sprintf(
			"DNS lookup failed for hostname %q (socket could not connect). Please verify that the server hostname is spelled correctly and your network is connected.",
			d.Host,
		)
	case "tcp_connect":
		d.DiagnosisJA = fmt.Sprintf(
			"%s:%d へのTCPソケット接続に失敗しました（ソケット未接続：接続拒否またはタイムアウト）。ポート番号（POP3: 995/110, IMAP: 993/143, SMTP: 587/465）が間違っていないか、サーバーアドレスやファイアウォール設定を確認してください。",
			d.Host, d.Port,
		)
		d.DiagnosisEN = fmt.Sprintf(
			"TCP socket connection to %s:%d failed (Socket not connected: connection refused or timed out). Please check if the port number (POP3: 995/110, IMAP: 993/143, SMTP: 587/465) and server address are correct.",
			d.Host, d.Port,
		)
	case "tls_handshake":
		d.DiagnosisJA = fmt.Sprintf(
			"TCPソケット（%s）は正常に接続できましたが、SSL/TLS暗号化ハンドシェイクに失敗しました。非SSLポート（110 / 143 / 587 等）に対して暗黙的TLSを指定していないか、ポート番号とTLS設定の組み合わせを確認してください。",
			d.RemoteAddr,
		)
		d.DiagnosisEN = fmt.Sprintf(
			"TCP socket (%s) connected, but the SSL/TLS handshake failed. Please check if SSL/TLS settings match the port number (e.g. implicit TLS on 995/993/465 vs plain/STARTTLS on 110/143/587).",
			d.RemoteAddr,
		)
	case "greeting":
		d.DiagnosisJA = fmt.Sprintf(
			"ソケットは接続されましたが、%sサーバーからの初期応答（Greeting）が不正または切断されました。別のプロトコルのポート番号を指定していないか、または平文ポートにSSLなしで接続する設定が合っているか確認してください。",
			d.Protocol,
		)
		d.DiagnosisEN = fmt.Sprintf(
			"Socket connected, but the initial %s server greeting was invalid or closed immediately. Check if the port belongs to a different protocol or requires SSL/TLS.",
			d.Protocol,
		)
	case "auth":
		d.DiagnosisJA = fmt.Sprintf(
			"ソケット接続およびサーバーとの通信は正常ですが、アカウント認証（ユーザー名 %q またはパスワード）がサーバーに拒否されました。ログインユーザー名・パスワード（Gmail / Yahoo / iCloud 等の場合はアプリパスワード）・POP/IMAPアクセス許可設定を確認してください。",
			d.Username,
		)
		d.DiagnosisEN = fmt.Sprintf(
			"Socket & server connection succeeded, but account authentication failed for user %q. Please verify your login username, password (or App Password), and server POP/IMAP access settings.",
			d.Username,
		)
	default:
		d.DiagnosisJA = fmt.Sprintf(
			"サーバー接続後に %s プロトコルコマンドでエラーが発生しました。下の送受信プロトコルログでサーバーからの応答内容を確認してください。",
			d.Protocol,
		)
		d.DiagnosisEN = fmt.Sprintf(
			"A %s protocol command error occurred after connecting. Please check the server response in the network log below.",
			d.Protocol,
		)
	}
}

func tlsVersionName(v uint16) string {
	switch v {
	case tls.VersionTLS13:
		return "TLS 1.3"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS10:
		return "TLS 1.0"
	default:
		return fmt.Sprintf("TLS 0x%04x", v)
	}
}

func dialMailServerWithDiag(host string, port int, useTLS bool, diag *NetworkDiagReport) (net.Conn, error) {
	trimmedHost := strings.TrimSpace(host)
	diag.Add("DNS", fmt.Sprintf("Resolving host %q...", trimmedHost))
	startDNS := time.Now()
	ips, dnsErr := net.LookupHost(trimmedHost)
	if dnsErr != nil {
		diag.Add("ERROR", fmt.Sprintf("DNS resolution failed (%v): %v", time.Since(startDNS).Round(time.Millisecond), dnsErr))
		diag.SetFailure("dns", fmt.Errorf("ホスト名 %q の名前解決に失敗しました: %w", trimmedHost, dnsErr))
		return nil, dnsErr
	}
	diag.ResolvedIPs = ips
	diag.Add("DNS", fmt.Sprintf("Resolved %q -> [%s] (%v)", trimmedHost, strings.Join(ips, ", "), time.Since(startDNS).Round(time.Millisecond)))

	addr := net.JoinHostPort(trimmedHost, strconv.Itoa(port))
	diag.Add("TCP", fmt.Sprintf("Dialing TCP socket %s (timeout 10s)...", addr))
	startTCP := time.Now()
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	rawConn, tcpErr := dialer.Dial("tcp", addr)
	if tcpErr != nil {
		diag.Add("ERROR", fmt.Sprintf("TCP socket connection failed (%v): %v", time.Since(startTCP).Round(time.Millisecond), tcpErr))
		diag.SetFailure("tcp_connect", fmt.Errorf("%s へのTCPソケット接続に失敗しました: %w", addr, tcpErr))
		return nil, tcpErr
	}

	diag.SocketConnected = true
	if rawConn.LocalAddr() != nil {
		diag.LocalAddr = rawConn.LocalAddr().String()
	}
	if rawConn.RemoteAddr() != nil {
		diag.RemoteAddr = rawConn.RemoteAddr().String()
	}
	diag.Add("TCP", fmt.Sprintf("Socket connected (%v)! Local=%s -> Remote=%s", time.Since(startTCP).Round(time.Millisecond), diag.LocalAddr, diag.RemoteAddr))

	if useTLS {
		diag.Add("TLS", fmt.Sprintf("Starting SSL/TLS handshake (SNI=%q)...", trimmedHost))
		startTLS := time.Now()
		_ = rawConn.SetDeadline(time.Now().Add(12 * time.Second))
		tlsConn := tls.Client(rawConn, &tls.Config{ServerName: trimmedHost})
		if tlsErr := tlsConn.Handshake(); tlsErr != nil {
			_ = rawConn.Close()
			diag.Add("ERROR", fmt.Sprintf("SSL/TLS handshake failed (%v): %v", time.Since(startTLS).Round(time.Millisecond), tlsErr))
			diag.SetFailure("tls_handshake", fmt.Errorf("SSL/TLSハンドシェイク失敗: %w", tlsErr))
			return nil, tlsErr
		}
		_ = tlsConn.SetDeadline(time.Time{})
		st := tlsConn.ConnectionState()
		diag.TLSCompleted = true
		diag.TLSInfo = fmt.Sprintf("%s / %s", tlsVersionName(st.Version), tls.CipherSuiteName(st.CipherSuite))
		diag.Add("TLS", fmt.Sprintf("SSL/TLS handshake OK (%v): %s", time.Since(startTLS).Round(time.Millisecond), diag.TLSInfo))
		return tlsConn, nil
	}

	diag.Add("INFO", "Using plain TCP connection (TLS disabled)")
	return rawConn, nil
}

func sendSMTPWithDiag(acc *db.Account, fromAddr string, rawRcpts []string, payload []byte, diag *NetworkDiagReport) error {
	implicitTLS := acc.SMTPPort == 465
	conn, err := dialMailServerWithDiag(acc.SMTPHost, acc.SMTPPort, implicitTLS, diag)
	if err != nil {
		return fmt.Errorf("SMTP接続エラー: %w", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))

	c, err := smtp.NewClient(conn, strings.TrimSpace(acc.SMTPHost))
	if err != nil {
		diag.Add("ERROR", fmt.Sprintf("SMTP greeting failed: %v", err))
		diag.SetFailure("greeting", fmt.Errorf("SMTPグリーティング失敗: %w", err))
		return err
	}
	defer c.Close()
	diag.Add("S->C", "220 SMTP server greeting received")

	diag.Add("C->S", "EHLO localhost")
	if err := c.Hello("localhost"); err != nil {
		diag.Add("ERROR", fmt.Sprintf("EHLO failed: %v", err))
		diag.SetFailure("greeting", fmt.Errorf("SMTP EHLO失敗: %w", err))
		return err
	}
	diag.Add("S->C", "250 EHLO OK")

	if !implicitTLS && acc.SMTPTLS {
		if ok, _ := c.Extension("STARTTLS"); ok {
			diag.Add("C->S", "STARTTLS")
			startTLS := time.Now()
			if err := c.StartTLS(&tls.Config{ServerName: strings.TrimSpace(acc.SMTPHost)}); err != nil {
				diag.Add("ERROR", fmt.Sprintf("STARTTLS handshake failed (%v): %v", time.Since(startTLS).Round(time.Millisecond), err))
				diag.SetFailure("tls_handshake", fmt.Errorf("SMTP STARTTLSハンドシェイク失敗: %w", err))
				return err
			}
			if st, ok := c.TLSConnectionState(); ok {
				diag.TLSCompleted = true
				diag.TLSInfo = fmt.Sprintf("STARTTLS %s / %s", tlsVersionName(st.Version), tls.CipherSuiteName(st.CipherSuite))
				diag.Add("TLS", fmt.Sprintf("STARTTLS handshake OK (%v): %s", time.Since(startTLS).Round(time.Millisecond), diag.TLSInfo))
			}
		} else {
			diag.Add("INFO", "Server does not advertise STARTTLS; continuing on current connection")
		}
	}

	if strings.TrimSpace(acc.Username) != "" {
		diag.Add("C->S", fmt.Sprintf("AUTH PLAIN (user=%q, password=******** masked, %d chars)", acc.Username, len(acc.Password)))
		auth := smtp.PlainAuth("", acc.Username, acc.Password, strings.TrimSpace(acc.SMTPHost))
		if err := c.Auth(auth); err != nil {
			diag.Add("ERROR", fmt.Sprintf("SMTP authentication failed: %v", err))
			diag.SetFailure("auth", fmt.Errorf("SMTP認証エラー: %w", err))
			return err
		}
		diag.Add("S->C", "235 2.7.0 Authentication successful")
	}

	cleanFrom := extractPureEmailAddr(fromAddr)
	diag.Add("C->S", fmt.Sprintf("MAIL FROM:<%s>", cleanFrom))
	if err := c.Mail(cleanFrom); err != nil {
		diag.Add("ERROR", fmt.Sprintf("MAIL FROM rejected: %v", err))
		diag.SetFailure("command", fmt.Errorf("MAIL FROMエラー: %w", err))
		return err
	}
	diag.Add("S->C", "250 2.1.0 Sender OK")

	for _, r := range rawRcpts {
		cleanRcpt := extractPureEmailAddr(r)
		if cleanRcpt == "" {
			continue
		}
		diag.Add("C->S", fmt.Sprintf("RCPT TO:<%s>", cleanRcpt))
		if err := c.Rcpt(cleanRcpt); err != nil {
			diag.Add("ERROR", fmt.Sprintf("RCPT TO <%s> rejected: %v", cleanRcpt, err))
			diag.SetFailure("command", fmt.Errorf("宛先 (%s) エラー: %w", cleanRcpt, err))
			return err
		}
		diag.Add("S->C", fmt.Sprintf("250 2.1.5 Recipient <%s> OK", cleanRcpt))
	}

	diag.Add("C->S", fmt.Sprintf("DATA (%d bytes)", len(payload)))
	w, err := c.Data()
	if err != nil {
		diag.Add("ERROR", fmt.Sprintf("DATA command failed: %v", err))
		diag.SetFailure("command", fmt.Errorf("SMTP DATAエラー: %w", err))
		return err
	}
	if _, err := w.Write(payload); err != nil {
		_ = w.Close()
		diag.Add("ERROR", fmt.Sprintf("Writing message body failed: %v", err))
		diag.SetFailure("command", err)
		return err
	}
	if err := w.Close(); err != nil {
		diag.Add("ERROR", fmt.Sprintf("Message delivery rejected: %v", err))
		diag.SetFailure("command", err)
		return err
	}
	diag.Add("S->C", "250 2.0.0 Message accepted for delivery")

	diag.Add("C->S", "QUIT")
	_ = c.Quit()
	return nil
}

func extractPureEmailAddr(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if a, err := mail.ParseAddress(raw); err == nil && a.Address != "" {
		return a.Address
	}
	if idx := strings.LastIndex(raw, "<"); idx != -1 {
		rest := raw[idx+1:]
		if end := strings.Index(rest, ">"); end != -1 {
			return strings.TrimSpace(rest[:end])
		}
	}
	return strings.Trim(raw, "<> ")
}

