package email

import (
    "crypto/tls"
    "fmt"
    "net"
    "net/smtp"
    "os"
    "strconv"
    "strings"
)

type EmailService struct {
    host     string
    port     int
    username string
    password string
    from     string
}

func NewEmailService() *EmailService {
    host := os.Getenv("SMTP_HOST")
    portStr := os.Getenv("SMTP_PORT")
    username := os.Getenv("SMTP_USER")
    password := os.Getenv("SMTP_PASS")
    
    if host == "" || portStr == "" || username == "" || password == "" {
        panic("SMTP configuration missing. Please set SMTP_HOST, SMTP_PORT, SMTP_USER, and SMTP_PASS")
    }
    
    port, err := strconv.Atoi(portStr)
    if err != nil {
        panic(fmt.Sprintf("Invalid SMTP_PORT: %s", portStr))
    }
    
    return &EmailService{
        host:     host,
        port:     port,
        username: username,
        password: password,
        from:     username, // Use username as from address
    }
}

func (e *EmailService) SendEmail(to, subject, htmlBody string) error {
    // Create the email message
    message := e.buildMessage(to, subject, htmlBody)
    
    // Connect to SMTP server
    addr := fmt.Sprintf("%s:%d", e.host, e.port)
    
    // Create TLS config
    tlsConfig := &tls.Config{
        InsecureSkipVerify: false,
        ServerName:         e.host,
    }
    
    // Connect to server
    conn, err := net.Dial("tcp", addr)
    if err != nil {
        return fmt.Errorf("failed to connect to SMTP server: %w", err)
    }
    defer conn.Close()
    
    // Create SMTP client
    client, err := smtp.NewClient(conn, e.host)
    if err != nil {
        return fmt.Errorf("failed to create SMTP client: %w", err)
    }
    defer client.Quit()
    
    // Start TLS
    if ok, _ := client.Extension("STARTTLS"); ok {
        if err = client.StartTLS(tlsConfig); err != nil {
            return fmt.Errorf("failed to start TLS: %w", err)
        }
    }
    
    // Authenticate
    auth := smtp.PlainAuth("", e.username, e.password, e.host)
    if err = client.Auth(auth); err != nil {
        return fmt.Errorf("authentication failed: %w", err)
    }
    
    // Set sender
    if err = client.Mail(e.from); err != nil {
        return fmt.Errorf("failed to set sender: %w", err)
    }
    
    // Set recipient
    if err = client.Rcpt(to); err != nil {
        return fmt.Errorf("failed to set recipient: %w", err)
    }
    
    // Send message
    writer, err := client.Data()
    if err != nil {
        return fmt.Errorf("failed to get data writer: %w", err)
    }
    
    _, err = writer.Write([]byte(message))
    if err != nil {
        return fmt.Errorf("failed to write message: %w", err)
    }
    
    err = writer.Close()
    if err != nil {
        return fmt.Errorf("failed to close writer: %w", err)
    }
    
    return nil
}

func (e *EmailService) buildMessage(to, subject, htmlBody string) string {
    var msg strings.Builder
    
    // Headers
    msg.WriteString(fmt.Sprintf("From: SUDO Kanban <%s>\r\n", e.from))
    msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
    msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
    msg.WriteString("MIME-Version: 1.0\r\n")
    msg.WriteString("Content-Type: multipart/alternative; boundary=\"boundary123\"\r\n")
    msg.WriteString("\r\n")
    
    // Plain text version
    msg.WriteString("--boundary123\r\n")
    msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
    msg.WriteString("\r\n")
    msg.WriteString(e.htmlToPlainText(htmlBody))
    msg.WriteString("\r\n\r\n")
    
    // HTML version
    msg.WriteString("--boundary123\r\n")
    msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
    msg.WriteString("\r\n")
    msg.WriteString(e.wrapHTML(htmlBody))
    msg.WriteString("\r\n\r\n")
    
    msg.WriteString("--boundary123--\r\n")
    
    return msg.String()
}

func (e *EmailService) htmlToPlainText(html string) string {
    // Simple HTML to text conversion
    text := html
    text = strings.ReplaceAll(text, "<br>", "\n")
    text = strings.ReplaceAll(text, "<br/>", "\n")
    text = strings.ReplaceAll(text, "<br />", "\n")
    text = strings.ReplaceAll(text, "</p>", "\n\n")
    text = strings.ReplaceAll(text, "</h1>", "\n")
    text = strings.ReplaceAll(text, "</h2>", "\n")
    text = strings.ReplaceAll(text, "</h3>", "\n")
    text = strings.ReplaceAll(text, "</div>", "\n")
    
    // Remove HTML tags
    for strings.Contains(text, "<") && strings.Contains(text, ">") {
        start := strings.Index(text, "<")
        end := strings.Index(text[start:], ">")
        if end != -1 {
            text = text[:start] + text[start+end+1:]
        } else {
            break
        }
    }
    
    // Clean up whitespace
    lines := strings.Split(text, "\n")
    var cleanLines []string
    for _, line := range lines {
        trimmed := strings.TrimSpace(line)
        if trimmed != "" {
            cleanLines = append(cleanLines, trimmed)
        }
    }
    
    return strings.Join(cleanLines, "\n")
}

func (e *EmailService) wrapHTML(body string) string {
    return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SUDO Kanban</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: white;
            text-align: center;
            padding: 30px 20px;
            border-radius: 8px 8px 0 0;
        }
        .content {
            background: #ffffff;
            padding: 30px;
            border: 1px solid #e5e7eb;
            border-top: none;
        }
        .footer {
            background: #f9fafb;
            padding: 20px;
            text-align: center;
            color: #6b7280;
            font-size: 14px;
            border: 1px solid #e5e7eb;
            border-top: none;
            border-radius: 0 0 8px 8px;
        }
        .code {
            background: #f3f4f6;
            padding: 15px;
            border-radius: 6px;
            text-align: center;
            margin: 20px 0;
            font-family: monospace;
            font-size: 24px;
            font-weight: bold;
            color: #1f2937;
            letter-spacing: 4px;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1 style="margin: 0; font-size: 28px;">SUDO Kanban</h1>
        <p style="margin: 10px 0 0 0; opacity: 0.9;">Suck It Up and Do It</p>
    </div>
    <div class="content">
        %s
    </div>
    <div class="footer">
        <p>This email was sent by SUDO Kanban Board</p>
        <p>If you didn't request this, please ignore this email.</p>
    </div>
</body>
</html>`, body)
}

// SendInvitationEmail sends an invitation email to a new team member
func (e *EmailService) SendInvitationEmail(to, boardName, inviterName, inviteURL string) error {
    subject := fmt.Sprintf("You've been invited to join '%s' on SUDO Kanban", boardName)
    
    body := fmt.Sprintf(`
        <h2>You've been invited to collaborate!</h2>
        <p><strong>%s</strong> has invited you to join the board "<strong>%s</strong>" on SUDO Kanban.</p>
        <p style="margin: 30px 0;">
            <a href="%s" style="background: #2563eb; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; display: inline-block;">
                Join Board
            </a>
        </p>
        <p>If the button doesn't work, copy and paste this link into your browser:</p>
        <p style="word-break: break-all; color: #6b7280;">%s</p>
        <br>
        <p>Get ready to organize and get things done!</p>
        <p>The SUDO Team</p>
    `, inviterName, boardName, inviteURL, inviteURL)
    
    return e.SendEmail(to, subject, body)
}

// SendTaskAssignmentEmail sends an email when a task is assigned
func (e *EmailService) SendTaskAssignmentEmail(to, taskTitle, boardName, assignerName, taskURL string) error {
    subject := fmt.Sprintf("Task assigned: %s", taskTitle)
    
    body := fmt.Sprintf(`
        <h2>You've been assigned a new task!</h2>
        <p><strong>%s</strong> has assigned you the task "<strong>%s</strong>" in the board "<strong>%s</strong>".</p>
        <p style="margin: 30px 0;">
            <a href="%s" style="background: #059669; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; display: inline-block;">
                View Task
            </a>
        </p>
        <p>Time to SUDO and get it done!</p>
        <p>The SUDO Team</p>
    `, assignerName, taskTitle, boardName, taskURL)
    
    return e.SendEmail(to, subject, body)
}

// SendDeadlineReminderEmail sends a reminder for upcoming deadlines
func (e *EmailService) SendDeadlineReminderEmail(to, taskTitle, boardName, deadline, taskURL string) error {
    subject := fmt.Sprintf("Deadline reminder: %s", taskTitle)
    
    body := fmt.Sprintf(`
        <h2>Deadline reminder!</h2>
        <p>Your task "<strong>%s</strong>" in the board "<strong>%s</strong>" is due on <strong>%s</strong>.</p>
        <p style="margin: 30px 0;">
            <a href="%s" style="background: #dc2626; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; display: inline-block;">
                View Task
            </a>
        </p>
        <p>Don't let deadlines slip - SUDO and finish it!</p>
        <p>The SUDO Team</p>
    `, taskTitle, boardName, deadline, taskURL)
    
    return e.SendEmail(to, subject, body)
}