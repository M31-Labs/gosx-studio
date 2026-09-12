package sitehost

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
	"sync"
	"time"
)

// mail.go is how the site tells its owner something happened.
//
// Nothing in Studio could send email, so a lead sat in an inbox nobody was
// told about. The default host sends through one of three transports, chosen
// by a single URL, and never silently: every attempt is recorded, the last
// result is shown in Settings, and a test button proves the setup before a
// real message depends on it.
//
//   smtp://user:pass@smtp.example.com:587?from=hello@example.com
//   smtps://user:pass@smtp.example.com:465?from=hello@example.com
//   resend://re_xxx?from=hello@example.com
//   postmark://server-token?from=hello@example.com

// Mail is one message to send.
type Mail struct {
	To      string
	Subject string
	Text    string
	ReplyTo string
}

// Mailer sends mail. Tests and embedders supply their own.
type Mailer interface {
	Send(ctx context.Context, mail Mail) error
	Describe() string
}

// mailStatus is the last thing that happened, shown in Settings.
type mailStatus struct {
	mu       sync.Mutex
	last     time.Time
	lastErr  string
	attempts int
	sent     int
}

func (s *mailStatus) record(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.last = time.Now()
	s.attempts++
	if err != nil {
		s.lastErr = err.Error()
		return
	}
	s.lastErr = ""
	s.sent++
}

func (s *mailStatus) snapshot() (time.Time, string, int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.last, s.lastErr, s.attempts, s.sent
}

// ParseMailURL turns a transport URL into a Mailer.
func ParseMailURL(raw string) (Mailer, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("mail: %w", err)
	}
	from := strings.TrimSpace(parsed.Query().Get("from"))
	if from == "" {
		return nil, errors.New("mail: the URL needs ?from=<address the email is sent from>")
	}
	switch parsed.Scheme {
	case "smtp", "smtps":
		host := parsed.Hostname()
		port := parsed.Port()
		if host == "" {
			return nil, errors.New("mail: smtp URL needs a host")
		}
		if port == "" {
			port = "587"
			if parsed.Scheme == "smtps" {
				port = "465"
			}
		}
		user := ""
		pass := ""
		if parsed.User != nil {
			user = parsed.User.Username()
			pass, _ = parsed.User.Password()
		}
		return &smtpMailer{host: host, port: port, user: user, pass: pass, from: from, implicitTLS: parsed.Scheme == "smtps"}, nil
	case "resend":
		key := firstNonEmpty(parsed.Host, parsed.User.String())
		if parsed.User != nil {
			key = parsed.User.Username()
		}
		if key == "" {
			key = parsed.Host
		}
		if key == "" {
			return nil, errors.New("mail: resend URL needs the API key, resend://re_xxx?from=...")
		}
		return &httpMailer{name: "Resend", endpoint: "https://api.resend.com/emails", key: key, from: from, shape: "resend"}, nil
	case "postmark":
		key := parsed.Host
		if parsed.User != nil && parsed.User.Username() != "" {
			key = parsed.User.Username()
		}
		if key == "" {
			return nil, errors.New("mail: postmark URL needs the server token, postmark://token?from=...")
		}
		return &httpMailer{name: "Postmark", endpoint: "https://api.postmarkapp.com/email", key: key, from: from, shape: "postmark"}, nil
	default:
		return nil, fmt.Errorf("mail: unknown transport %q (use smtp, smtps, resend, or postmark)", parsed.Scheme)
	}
}

// ---------- SMTP ----------

type smtpMailer struct {
	host, port, user, pass, from string
	implicitTLS                  bool
}

func (m *smtpMailer) Describe() string { return "SMTP via " + m.host + ", sending from " + m.from }

func (m *smtpMailer) Send(ctx context.Context, mail Mail) error {
	address := net.JoinHostPort(m.host, m.port)
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	var conn net.Conn
	var err error
	if m.implicitTLS {
		conn, err = tls.DialWithDialer(dialer, "tcp", address, &tls.Config{ServerName: m.host, MinVersion: tls.VersionTLS12})
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", address)
	}
	if err != nil {
		return err
	}
	defer conn.Close()
	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return err
	}
	defer client.Close()
	if !m.implicitTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: m.host, MinVersion: tls.VersionTLS12}); err != nil {
				return err
			}
		}
	}
	if m.user != "" {
		if err := client.Auth(smtp.PlainAuth("", m.user, m.pass, m.host)); err != nil {
			return err
		}
	}
	if err := client.Mail(m.from); err != nil {
		return err
	}
	if err := client.Rcpt(mail.To); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write([]byte(formatMessage(m.from, mail))); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func formatMessage(from string, mail Mail) string {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + mail.To + "\r\n")
	if mail.ReplyTo != "" {
		b.WriteString("Reply-To: " + mail.ReplyTo + "\r\n")
	}
	b.WriteString("Subject: " + sanitizeHeader(mail.Subject) + "\r\n")
	b.WriteString("Date: " + time.Now().UTC().Format(time.RFC1123Z) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n")
	b.WriteString(strings.ReplaceAll(mail.Text, "\n", "\r\n"))
	return b.String()
}

func sanitizeHeader(value string) string {
	return strings.NewReplacer("\r", " ", "\n", " ").Replace(value)
}

// ---------- HTTPS APIs (Resend, Postmark) ----------

type httpMailer struct {
	name, endpoint, key, from, shape string
	client                           *http.Client
}

func (m *httpMailer) Describe() string { return m.name + ", sending from " + m.from }

func (m *httpMailer) Send(ctx context.Context, mail Mail) error {
	var payload map[string]any
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.endpoint, nil)
	if err != nil {
		return err
	}
	switch m.shape {
	case "resend":
		payload = map[string]any{"from": m.from, "to": []string{mail.To}, "subject": mail.Subject, "text": mail.Text}
		if mail.ReplyTo != "" {
			payload["reply_to"] = mail.ReplyTo
		}
		req.Header.Set("Authorization", "Bearer "+m.key)
	case "postmark":
		payload = map[string]any{"From": m.from, "To": mail.To, "Subject": mail.Subject, "TextBody": mail.Text, "MessageStream": "outbound"}
		if mail.ReplyTo != "" {
			payload["ReplyTo"] = mail.ReplyTo
		}
		req.Header.Set("X-Postmark-Server-Token", m.key)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.ContentLength = int64(len(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := m.client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<10))
		return fmt.Errorf("%s replied %d: %s", m.name, resp.StatusCode, strings.TrimSpace(string(detail)))
	}
	return nil
}

// ---------- the host's use of it ----------

const notifyEmailKey = "notifyEmail"

// notifyAddress is where owner notifications go: the Settings override, or
// the contact email the wizard collected.
func (h *Host) notifyAddress() string {
	settings := h.settings()
	return firstNonEmpty(settings.Metadata[notifyEmailKey], settings.Metadata["contactEmail"])
}

// notify sends in the background with a bounded wait, records the outcome,
// and never blocks the request that triggered it.
func (h *Host) notify(mail Mail) {
	if h.mailer == nil || strings.TrimSpace(mail.To) == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		err := h.mailer.Send(ctx, mail)
		h.mailStatus.record(err)
		if err != nil {
			log.Printf("gosx-site: email to %s failed: %v", mail.To, err)
		}
	}()
}

// sendNow sends synchronously; the Settings test button uses it so the
// owner sees the result on the same page.
func (h *Host) sendNow(mail Mail) error {
	if h.mailer == nil {
		return errors.New("no email transport is configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	err := h.mailer.Send(ctx, mail)
	h.mailStatus.record(err)
	return err
}

func (h *Host) newMessageMail(message Message, base string) Mail {
	siteTitle := firstNonEmpty(h.settings().Title, h.opts.SiteTitle)
	var b strings.Builder
	b.WriteString("Someone sent a message through your website.\n\n")
	b.WriteString("From: " + message.Name + " <" + message.Email + ">\n")
	if message.Page != "" {
		b.WriteString("Page: " + base + message.Page + "\n")
	}
	b.WriteString("\n" + message.Body + "\n\n")
	b.WriteString("Reply to this email to answer them, or see all messages at " + base + "/admin/messages\n")
	return Mail{
		To:      h.notifyAddress(),
		Subject: "New message from " + message.Name + " — " + siteTitle,
		Text:    b.String(),
		ReplyTo: message.Email,
	}
}
