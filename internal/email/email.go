package email

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"time"
)

type EmailService struct {
	smtpHost     string
	smtpPort     string
	smtpUsername string
	smtpPassword string
	fromEmail    string
	fromName     string
	resendAPIKey string
	useResend    bool
}

func NewEmailService() *EmailService {
	resendKey := os.Getenv("RESEND_API_KEY")

	return &EmailService{
		smtpHost:     getEnvOrDefault("SMTP_HOST", "smtp.gmail.com"),
		smtpPort:     getEnvOrDefault("SMTP_PORT", "465"),
		smtpUsername: getEnvOrDefault("SMTP_USERNAME", ""),
		smtpPassword: getEnvOrDefault("SMTP_PASSWORD", ""),
		fromEmail:    getEnvOrDefault("FROM_EMAIL", "send@sudo-kanban.co.in"),
		fromName:     getEnvOrDefault("FROM_NAME", "SUDO Kanban Board"),
		resendAPIKey: resendKey,
		useResend:    resendKey != "",
	}
}

func (e *EmailService) SendOTP(to, otp string) error {
	// Use Resend if API key is configured
	if e.useResend {
		log.Printf("[EMAIL] Sending OTP to %s via Resend API", to)
		subject := "Your Login Code for SUDO Kanban Board"
		body := e.buildOTPEmailBody(otp)
		return e.sendViaResend(to, subject, body)
	}

	// Fall back to SMTP if configured
	if e.smtpUsername == "" || e.smtpPassword == "" {
		// For development, log the OTP instead of sending email
		log.Printf("[EMAIL] Dev mode - OTP for %s: %s", to, otp)
		return nil
	}

	log.Printf("[EMAIL] Attempting to send OTP to %s via %s:%s", to, e.smtpHost, e.smtpPort)

	subject := "Your Login Code for SUDO Kanban Board"
	body := e.buildOTPEmailBody(otp)

	err := e.sendEmail(to, subject, body)
	if err != nil {
		log.Printf("[EMAIL] Failed to send OTP to %s: %v", to, err)
		return err
	}

	log.Printf("[EMAIL] Successfully sent OTP to %s", to)
	return nil
}

func (e *EmailService) SendInvitation(to, inviterName, boardName, inviteLink string) error {
	subject := fmt.Sprintf("You've been invited to join '%s' on SUDO Kanban Board", boardName)
	body := e.buildInvitationEmailBody(inviterName, boardName, inviteLink)

	// Use Resend if API key is configured
	if e.useResend {
		log.Printf("[EMAIL] Sending invitation to %s via Resend API", to)
		return e.sendViaResend(to, subject, body)
	}

	// Fall back to SMTP if configured
	if e.smtpUsername == "" || e.smtpPassword == "" {
		// For development, log the invitation instead of sending email
		fmt.Printf("\nBoard invitation for %s: %s invited you to '%s'\n", to, inviterName, boardName)
		return nil
	}

	return e.sendEmail(to, subject, body)
}

func (e *EmailService) SendEmail(to, subject, body string) error {
	// Use Resend if API key is configured
	if e.useResend {
		log.Printf("[EMAIL] Sending email to %s via Resend API", to)
		return e.sendViaResend(to, subject, body)
	}

	// Fall back to SMTP if configured
	if e.smtpUsername == "" || e.smtpPassword == "" {
		// For development, log the email instead of sending
		fmt.Printf("\nEMAIL TO: %s\n", to)
		fmt.Printf("SUBJECT: %s\n", subject)
		fmt.Printf("BODY: %s\n", body)
		return nil
	}

	return e.sendEmail(to, subject, body)
}

func (e *EmailService) sendEmail(to, subject, body string) error {
	from := fmt.Sprintf("%s <%s>", e.fromName, e.fromEmail)

	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		from, to, subject, body)

	auth := smtp.PlainAuth("", e.smtpUsername, e.smtpPassword, e.smtpHost)
	addr := fmt.Sprintf("%s:%s", e.smtpHost, e.smtpPort)

	// Use TLS for port 587 (STARTTLS)
	if e.smtpPort == "587" {
		return e.sendEmailWithTLS(addr, auth, e.fromEmail, []string{to}, []byte(message))
	}

	// Use SSL/TLS for port 465
	if e.smtpPort == "465" {
		return e.sendEmailWithSSL(addr, auth, e.fromEmail, []string{to}, []byte(message))
	}

	// Fallback to basic SMTP for other ports
	err := smtp.SendMail(addr, auth, e.fromEmail, []string{to}, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func (e *EmailService) sendEmailWithTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	// Connect to SMTP server with 30 second timeout
	log.Printf("[EMAIL] Connecting to %s...", addr)
	conn, err := net.DialTimeout("tcp", addr, 30*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	// Set deadline for entire operation (60 seconds total)
	if deadlineErr := conn.SetDeadline(time.Now().Add(60 * time.Second)); deadlineErr != nil {
		return fmt.Errorf("failed to set deadline: %w", deadlineErr)
	}

	log.Printf("[EMAIL] Creating SMTP client...")
	// Create SMTP client
	client, err := smtp.NewClient(conn, e.smtpHost)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer func() {
		_ = client.Quit()
	}()

	log.Printf("[EMAIL] Starting TLS...")
	// Start TLS
	tlsConfig := &tls.Config{
		ServerName: e.smtpHost,
		MinVersion: tls.VersionTLS12,
	}

	if err = client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("failed to start TLS: %w", err)
	}

	log.Printf("[EMAIL] Authenticating...")
	// Authenticate
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	log.Printf("[EMAIL] Setting sender...")
	// Set sender
	if err = client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	log.Printf("[EMAIL] Setting recipient...")
	// Set recipients
	for _, recipient := range to {
		if err = client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient: %w", err)
		}
	}

	log.Printf("[EMAIL] Sending message data...")
	// Send message
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	log.Printf("[EMAIL] Message sent successfully")
	return nil
}

func (e *EmailService) sendEmailWithSSL(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	log.Printf("[EMAIL] Connecting to %s via SSL/TLS...", addr)

	// Create TLS config
	tlsConfig := &tls.Config{
		ServerName: e.smtpHost,
		MinVersion: tls.VersionTLS12,
	}

	// Connect directly with TLS (for port 465)
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect via SSL/TLS: %w", err)
	}
	defer conn.Close()

	// Set deadline for entire operation
	if deadlineErr := conn.SetDeadline(time.Now().Add(60 * time.Second)); deadlineErr != nil {
		return fmt.Errorf("failed to set deadline: %w", deadlineErr)
	}

	log.Printf("[EMAIL] Creating SMTP client...")
	client, err := smtp.NewClient(conn, e.smtpHost)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer func() {
		_ = client.Quit()
	}()

	log.Printf("[EMAIL] Authenticating...")
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	log.Printf("[EMAIL] Setting sender...")
	if err = client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	log.Printf("[EMAIL] Setting recipient...")
	for _, recipient := range to {
		if err = client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient: %w", err)
		}
	}

	log.Printf("[EMAIL] Sending message data...")
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	log.Printf("[EMAIL] Message sent successfully via SSL/TLS")
	return nil
}

func (e *EmailService) buildOTPEmailBody(otp string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Your Login Code</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; background-color: #f4f4f4; margin: 0; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background-color: #ffffff; padding: 40px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .header { text-align: center; margin-bottom: 30px; }
        .logo { font-size: 24px; font-weight: bold; color: #2563eb; }
        .otp-container { background-color: #f8fafc; padding: 30px; border-radius: 8px; text-align: center; margin: 30px 0; border: 2px solid #e2e8f0; }
        .otp-code { font-size: 36px; font-weight: bold; letter-spacing: 8px; color: #1e40af; font-family: 'Courier New', monospace; margin: 10px 0; }
        .warning { background-color: #fef3cd; border-left: 4px solid #fbbf24; padding: 15px; margin: 20px 0; border-radius: 4px; }
        .footer { text-align: center; margin-top: 30px; padding-top: 20px; border-top: 1px solid #e5e7eb; color: #6b7280; font-size: 14px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="logo">SUDO Kanban Board</div>
        </div>
        
        <h1>Your Login Code</h1>
        
        <p>Hello!</p>
        
        <p>Use the following 6-digit code to sign in to your SUDO Kanban Board account:</p>
        
        <div class="otp-container">
            <div class="otp-code">%s</div>
            <p style="margin: 10px 0 0 0; color: #6b7280;">Enter this code in your browser</p>
        </div>
        
        <div class="warning">
            <strong>Security Notice:</strong>
            <ul style="margin: 10px 0 0 0; padding-left: 20px;">
                <li>This code expires in 10 minutes</li>
                <li>Don't share this code with anyone</li>
                <li>We'll never ask for this code via phone or email</li>
            </ul>
        </div>
        
        <p>If you didn't request this code, you can safely ignore this email.</p>
        
        <div class="footer">
            <p>© 2025 SUDO Kanban Board. All rights reserved.</p>
            <p>This is an automated message, please do not reply to this email.</p>
        </div>
    </div>
</body>
</html>`, otp)
}

func (e *EmailService) buildInvitationEmailBody(inviterName, boardName, inviteLink string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Board Invitation</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; background-color: #f4f4f4; margin: 0; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background-color: #ffffff; padding: 40px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .header { text-align: center; margin-bottom: 30px; }
        .logo { font-size: 24px; font-weight: bold; color: #2563eb; }
        .invite-box { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 30px; border-radius: 8px; text-align: center; margin: 30px 0; }
        .board-name { font-size: 24px; font-weight: bold; margin: 10px 0; }
        .cta-button { display: inline-block; background-color: #10b981; color: white; padding: 15px 30px; text-decoration: none; border-radius: 6px; font-weight: bold; margin: 20px 0; transition: background-color 0.3s; }
        .cta-button:hover { background-color: #059669; }
        .features { background-color: #f8fafc; padding: 20px; border-radius: 6px; margin: 20px 0; }
        .feature { margin: 10px 0; }
        .footer { text-align: center; margin-top: 30px; padding-top: 20px; border-top: 1px solid #e5e7eb; color: #6b7280; font-size: 14px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="logo">SUDO Kanban Board</div>
        </div>
        
        <h1>You've been invited to collaborate!</h1>
        
        <p>Hi there!</p>
        
        <p><strong>%s</strong> has invited you to join their Kanban board and start collaborating together.</p>
        
        <div class="invite-box">
            <div>You're invited to join:</div>
            <div class="board-name">"%s"</div>
            <a href="%s" class="cta-button">Accept Invitation</a>
        </div>
        
        <div class="features">
            <h3>What you can do:</h3>
            <div class="feature">Create and manage tasks</div>
            <div class="feature">Track project progress in real-time</div>
            <div class="feature">Collaborate with team members</div>
            <div class="feature">Organize work with drag & drop</div>
            <div class="feature">Get instant updates and notifications</div>
        </div>
        
        <p>Click the button above to get started, or copy and paste this link into your browser:</p>
        <p style="word-break: break-all; background-color: #f3f4f6; padding: 10px; border-radius: 4px; font-family: monospace;">%s</p>
        
        <p>Looking forward to seeing you on the board!</p>
        
        <div class="footer">
            <p>© 2025 SUDO Kanban Board. All rights reserved.</p>
            <p>If you don't want to receive these invitations, please contact the sender directly.</p>
        </div>
    </div>
</body>
</html>`, inviterName, boardName, inviteLink, inviteLink)
}

func (e *EmailService) sendViaResend(to, subject, body string) error {
	type ResendEmail struct {
		From    string   `json:"from"`
		To      []string `json:"to"`
		Subject string   `json:"subject"`
		HTML    string   `json:"html"`
	}

	email := ResendEmail{
		From:    fmt.Sprintf("%s <%s>", e.fromName, e.fromEmail),
		To:      []string{to},
		Subject: subject,
		HTML:    body,
	}

	jsonData, err := json.Marshal(email)
	if err != nil {
		return fmt.Errorf("failed to marshal email: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", e.resendAPIKey))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Printf("[EMAIL] Resend API error (status %d): %s", resp.StatusCode, string(bodyBytes))
		return fmt.Errorf("resend API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	log.Printf("[EMAIL] Successfully sent email via Resend to %s", to)
	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
