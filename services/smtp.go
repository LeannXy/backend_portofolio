package services

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"
)

type EmailMessage struct {
	To      string
	ReplyTo string
	Subject string
	HTML    string
	Text    string
}

type EmailSender interface {
	Send(context.Context, EmailMessage) error
}

type SMTPMailer struct {
	host        string
	port        string
	username    string
	password    string
	encryption  string
	fromAddress string
	fromName    string
}

func NewSMTPMailerFromEnv() (*SMTPMailer, error) {
	host := strings.TrimSpace(os.Getenv("MAIL_HOST"))
	port := strings.TrimSpace(os.Getenv("MAIL_PORT"))
	username := strings.TrimSpace(os.Getenv("MAIL_USERNAME"))
	password := strings.TrimSpace(os.Getenv("MAIL_PASSWORD"))
	encryption := strings.ToLower(strings.TrimSpace(os.Getenv("MAIL_ENCRYPTION")))
	fromAddress := strings.TrimSpace(os.Getenv("MAIL_FROM_ADDRESS"))
	fromName := strings.TrimSpace(os.Getenv("MAIL_FROM_NAME"))

	if host == "" && port == "" && username == "" && password == "" &&
		encryption == "" && fromAddress == "" && fromName == "" {
		return nil, nil
	}
	if host == "" || username == "" || password == "" || !validEmailAddress(username) ||
		!validSenderAddress(fromAddress) {
		return nil, fmt.Errorf("MAIL_HOST, valid MAIL_USERNAME, MAIL_PASSWORD, and MAIL_FROM_ADDRESS are required")
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return nil, fmt.Errorf("MAIL_PORT must be a valid TCP port")
	}
	if encryption != "tls" && encryption != "ssl" {
		return nil, fmt.Errorf("MAIL_ENCRYPTION must be tls (STARTTLS) or ssl")
	}
	if strings.ContainsAny(host, "\r\n") || strings.ContainsAny(fromName, "\r\n") {
		return nil, fmt.Errorf("SMTP host and sender name cannot contain line breaks")
	}

	return &SMTPMailer{
		host:        host,
		port:        port,
		username:    username,
		password:    password,
		encryption:  encryption,
		fromAddress: fromAddress,
		fromName:    fromName,
	}, nil
}

func (mailer *SMTPMailer) Send(ctx context.Context, message EmailMessage) error {
	if mailer == nil {
		return fmt.Errorf("email provider is not configured")
	}
	if !validEmailAddress(message.To) || message.Subject == "" || message.HTML == "" ||
		strings.ContainsAny(message.Subject, "\r\n") {
		return fmt.Errorf("email message is incomplete or invalid")
	}
	if message.ReplyTo != "" && !validEmailAddress(message.ReplyTo) {
		return fmt.Errorf("email reply address is invalid")
	}

	rawMessage, err := mailer.formatMessage(message)
	if err != nil {
		return fmt.Errorf("format email message: %w", err)
	}

	dialer := &net.Dialer{Timeout: 12 * time.Second}
	connection, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(mailer.host, mailer.port))
	if err != nil {
		return fmt.Errorf("connect to SMTP server: %w", err)
	}
	defer connection.Close()

	deadline := time.Now().Add(12 * time.Second)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	if err := connection.SetDeadline(deadline); err != nil {
		return fmt.Errorf("set SMTP connection deadline: %w", err)
	}

	if mailer.encryption == "ssl" {
		tlsConnection := tls.Client(connection, &tls.Config{ServerName: mailer.host, MinVersion: tls.VersionTLS12})
		if err := tlsConnection.HandshakeContext(ctx); err != nil {
			return fmt.Errorf("establish SMTP TLS connection: %w", err)
		}
		connection = tlsConnection
	}

	client, err := smtp.NewClient(connection, mailer.host)
	if err != nil {
		return fmt.Errorf("start SMTP session: %w", err)
	}
	defer client.Close()

	if mailer.encryption == "tls" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return fmt.Errorf("SMTP server does not support STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{ServerName: mailer.host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("establish SMTP STARTTLS connection: %w", err)
		}
	}
	if ok, _ := client.Extension("AUTH"); !ok {
		return fmt.Errorf("SMTP server does not support authentication")
	}
	if err := client.Auth(smtp.PlainAuth("", mailer.username, mailer.password, mailer.host)); err != nil {
		return fmt.Errorf("authenticate with SMTP server: %w", err)
	}

	from, err := mail.ParseAddress(mailer.fromAddress)
	if err != nil {
		return fmt.Errorf("parse sender address: %w", err)
	}
	to, err := mail.ParseAddress(message.To)
	if err != nil {
		return fmt.Errorf("parse recipient address: %w", err)
	}
	if err := client.Mail(from.Address); err != nil {
		return fmt.Errorf("set SMTP sender: %w", err)
	}
	if err := client.Rcpt(to.Address); err != nil {
		return fmt.Errorf("set SMTP recipient: %w", err)
	}
	dataWriter, err := client.Data()
	if err != nil {
		return fmt.Errorf("begin SMTP message: %w", err)
	}
	if _, err := dataWriter.Write(rawMessage); err != nil {
		_ = dataWriter.Close()
		return fmt.Errorf("write SMTP message: %w", err)
	}
	if err := dataWriter.Close(); err != nil {
		return fmt.Errorf("finish SMTP message: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("close SMTP session: %w", err)
	}
	return nil
}

func (mailer *SMTPMailer) formatMessage(message EmailMessage) ([]byte, error) {
	from, err := mail.ParseAddress(mailer.fromAddress)
	if err != nil {
		return nil, err
	}
	if mailer.fromName != "" {
		from = &mail.Address{Name: mailer.fromName, Address: from.Address}
	}
	to, err := mail.ParseAddress(message.To)
	if err != nil {
		return nil, err
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, part := range []struct {
		contentType string
		content     string
	}{
		{"text/plain; charset=UTF-8", message.Text},
		{"text/html; charset=UTF-8", message.HTML},
	} {
		header := make(map[string][]string)
		header["Content-Type"] = []string{part.contentType}
		header["Content-Transfer-Encoding"] = []string{"quoted-printable"}
		partWriter, err := writer.CreatePart(header)
		if err != nil {
			return nil, err
		}
		encodedWriter := quotedprintable.NewWriter(partWriter)
		if _, err := io.WriteString(encodedWriter, part.content); err != nil {
			return nil, err
		}
		if err := encodedWriter.Close(); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	var output bytes.Buffer
	fmt.Fprintf(&output, "From: %s\r\n", from.String())
	fmt.Fprintf(&output, "To: %s\r\n", to.String())
	if message.ReplyTo != "" {
		replyTo, err := mail.ParseAddress(message.ReplyTo)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&output, "Reply-To: %s\r\n", replyTo.String())
	}
	fmt.Fprintf(&output, "Subject: %s\r\n", mime.QEncoding.Encode("UTF-8", message.Subject))
	output.WriteString("MIME-Version: 1.0\r\n")
	fmt.Fprintf(&output, "Content-Type: multipart/alternative; boundary=%q\r\n\r\n", writer.Boundary())
	output.Write(body.Bytes())
	return output.Bytes(), nil
}

func validEmailAddress(value string) bool {
	if value == "" || len(value) > 254 || strings.ContainsAny(value, "\r\n") {
		return false
	}
	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value
}

func validSenderAddress(value string) bool {
	if value == "" || len(value) > 254 || strings.ContainsAny(value, "\r\n") {
		return false
	}
	_, err := mail.ParseAddress(value)
	return err == nil
}
