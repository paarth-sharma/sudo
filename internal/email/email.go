package email

import (
    "fmt"
    "net/smtp"
    "os"
)

type EmailService struct {
    smtpHost     string
    smtpPort     string
    smtpUsername string
    smtpPassword string
    fromEmail    string
    fromName     string
}

func NewEmailService() *EmailService {
    return &EmailService{
        smtpHost:     getEnvOrDefault("SMTP_HOST", "smtp.gmail.com"),
        smtpPort:     getEnvOrDefault("SMTP_PORT", "465"),
        smtpUsername: getEnvOrDefault("SMTP_USERNAME", ""),
        smtpPassword: getEnvOrDefault("SMTP_PASSWORD", ""),
        fromEmail:    getEnvOrDefault("FROM_EMAIL", "noreply@kanban.app"),
        fromName:     getEnvOrDefault("FROM_NAME", "SUDO Kanban Board"),
    }
}

func (e *EmailService) SendOTP(to, otp string) error {
    if e.smtpUsername == "" || e.smtpPassword == "" {
        // For development, log the OTP instead of sending email
        fmt.Printf("\n🔐 OTP for %s: %s\n", to, otp)
        return nil
    }
    
    subject := "Your Login Code for SUDO Kanban Board"
    body := e.buildOTPEmailBody(otp)
    
    return e.sendEmail(to, subject, body)
}

func (e *EmailService) SendInvitation(to, inviterName, boardName, inviteLink string) error {
    if e.smtpUsername == "" || e.smtpPassword == "" {
        // For development, log the invitation instead of sending email
        fmt.Printf("\n📧 Board invitation for %s: %s invited you to '%s'\n", to, inviterName, boardName)
        return nil
    }
    
    subject := fmt.Sprintf("You've been invited to join '%s' on SUDO Kanban Board", boardName)
    body := e.buildInvitationEmailBody(inviterName, boardName, inviteLink)
    
    return e.sendEmail(to, subject, body)
}

func (e *EmailService) SendEmail(to, subject, body string) error {
    if e.smtpUsername == "" || e.smtpPassword == "" {
        // For development, log the email instead of sending
        fmt.Printf("\n📧 EMAIL TO: %s\n", to)
        fmt.Printf("📧 SUBJECT: %s\n", subject)
        fmt.Printf("📧 BODY: %s\n", body)
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
    
    err := smtp.SendMail(addr, auth, e.fromEmail, []string{to}, []byte(message))
    if err != nil {
        return fmt.Errorf("failed to send email: %w", err)
    }
    
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
            <div class="logo">🔲 SUDO Kanban Board</div>
        </div>
        
        <h1>Your Login Code</h1>
        
        <p>Hello!</p>
        
        <p>Use the following 6-digit code to sign in to your SUDO Kanban Board account:</p>
        
        <div class="otp-container">
            <div class="otp-code">%s</div>
            <p style="margin: 10px 0 0 0; color: #6b7280;">Enter this code in your browser</p>
        </div>
        
        <div class="warning">
            <strong>⚠️ Security Notice:</strong>
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
            <div class="logo">🔲 SUDO Kanban Board</div>
        </div>
        
        <h1>You've been invited to collaborate!</h1>
        
        <p>Hi there!</p>
        
        <p><strong>%s</strong> has invited you to join their Kanban board and start collaborating together.</p>
        
        <div class="invite-box">
            <div>📋 You're invited to join:</div>
            <div class="board-name">"%s"</div>
            <a href="%s" class="cta-button">Accept Invitation</a>
        </div>
        
        <div class="features">
            <h3>🚀 What you can do:</h3>
            <div class="feature">✅ Create and manage tasks</div>
            <div class="feature">🔄 Track project progress in real-time</div>
            <div class="feature">👥 Collaborate with team members</div>
            <div class="feature">📊 Organize work with drag & drop</div>
            <div class="feature">⚡ Get instant updates and notifications</div>
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

func getEnvOrDefault(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}