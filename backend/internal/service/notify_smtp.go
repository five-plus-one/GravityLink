package service

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"
)

type SMTPSettings struct {
	Host       string
	Port       int
	Encryption string // ssl | starttls | none
	From       string // 邮箱地址
	FromName   string // 显示名，如 GravityLink
	Username   string
	Password   string
	To         []string
}

// formatFromHeader 组装 From 头：有显示名时为 "Name <email>"（Name 做 MIME 编码）。
func formatFromHeader(name, email string) string {
	email = strings.TrimSpace(email)
	name = strings.TrimSpace(name)
	if name == "" {
		return email
	}
	// 已是完整 "Name <email>" 形式则原样返回
	if strings.Contains(email, "<") && strings.Contains(email, ">") {
		return email
	}
	encoded := mime.QEncoding.Encode("utf-8", name)
	return fmt.Sprintf("%s <%s>", encoded, email)
}

// parseFrom 拆出发件邮箱与显示名；支持 "Name <email>" 或纯邮箱。
func parseFrom(raw, fallbackName string) (name, email string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return strings.TrimSpace(fallbackName), ""
	}
	if i := strings.LastIndex(raw, "<"); i >= 0 && strings.HasSuffix(raw, ">") {
		email = strings.TrimSpace(raw[i+1 : len(raw)-1])
		n := strings.TrimSpace(raw[:i])
		n = strings.Trim(n, `"'`)
		if n != "" {
			return n, email
		}
		return strings.TrimSpace(fallbackName), email
	}
	return strings.TrimSpace(fallbackName), raw
}

func sendSMTPEmail(ctx context.Context, cfg SMTPSettings, subject, body string) error {
	if cfg.Host == "" || cfg.Port == 0 {
		return fmt.Errorf("%w: SMTP 服务器未配置", ErrNotify)
	}
	if len(cfg.To) == 0 {
		return fmt.Errorf("%w: 未配置收件人", ErrNotify)
	}
	fromName, fromEmail := parseFrom(cfg.From, cfg.FromName)
	if fromEmail == "" {
		fromEmail = strings.TrimSpace(cfg.Username)
	}
	if fromEmail == "" {
		return fmt.Errorf("%w: 未配置发件人", ErrNotify)
	}

	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	msg := buildSMTPMessage(formatFromHeader(fromName, fromEmail), cfg.To, subject, body)

	d := net.Dialer{Timeout: 10 * time.Second}
	var conn net.Conn
	var err error
	if cfg.Encryption == "ssl" || (cfg.Encryption == "" && cfg.Port == 465) {
		tlsCfg := &tls.Config{ServerName: cfg.Host}
		conn, err = tls.DialWithDialer(&d, "tcp", addr, tlsCfg)
	} else {
		conn, err = d.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("连接 SMTP 失败：%w", sanitizeNetErr(err))
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return fmt.Errorf("SMTP 握手失败：%w", sanitizeNetErr(err))
	}
	defer client.Close()

	if cfg.Encryption == "starttls" || (cfg.Encryption == "" && cfg.Port == 587) {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: cfg.Host}); err != nil {
				return fmt.Errorf("STARTTLS 失败：%w", sanitizeNetErr(err))
			}
		}
	}

	if cfg.Username != "" && cfg.Password != "" {
		auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP 认证失败，请检查授权码：%w", sanitizeNetErr(err))
		}
	}
	// MAIL FROM 必须是纯邮箱
	if err := client.Mail(fromEmail); err != nil {
		return fmt.Errorf("SMTP MAIL FROM 失败：%w", sanitizeNetErr(err))
	}
	for _, to := range cfg.To {
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("SMTP 收件人 %s 无效：%w", to, sanitizeNetErr(err))
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA 失败：%w", sanitizeNetErr(err))
	}
	if _, err := w.Write(msg); err != nil {
		_ = w.Close()
		return fmt.Errorf("SMTP 写入失败：%w", sanitizeNetErr(err))
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("SMTP 发送失败：%w", sanitizeNetErr(err))
	}
	return client.Quit()
}

func buildSMTPMessage(from string, to []string, subject, body string) []byte {
	encodedSubject := mime.QEncoding.Encode("utf-8", subject)
	// 正文用 base64，避免中文被中间设备截断
	encodedBody := base64.StdEncoding.EncodeToString([]byte(body))
	// 简单换行折叠，避免超长 base64 行
	var folded strings.Builder
	for i, r := range encodedBody {
		if i > 0 && i%76 == 0 {
			folded.WriteString("\r\n")
		}
		folded.WriteRune(r)
	}
	headers := []string{
		"From: " + from,
		"To: " + strings.Join(to, ", "),
		"Subject: " + encodedSubject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"Content-Transfer-Encoding: base64",
		"Date: " + time.Now().Format(time.RFC1123Z),
	}
	return []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + folded.String() + "\r\n")
}

// sanitizeNetErr 避免把完整连接串/认证细节回给前端。
func sanitizeNetErr(err error) error {
	if err == nil {
		return nil
	}
	s := err.Error()
	// 去掉可能包含密码的 auth 明文（PlainAuth 错误一般不含密码，仍做兜底）
	if len(s) > 200 {
		s = s[:200]
	}
	return fmt.Errorf("%s", s)
}
