package operations

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/nas-ai/api/src/config"
	"github.com/resend/resend-go/v2"
	"github.com/sirupsen/logrus"
)

// EmailService handles email operations via Resend
type EmailService struct {
	client      *resend.Client
	fromAddress string
	frontendURL string
	logger      *logrus.Logger
}

// NewEmailService creates a new email service
func NewEmailService(cfg *config.Config, logger *logrus.Logger) *EmailService {
	client := resend.NewClient(cfg.ResendAPIKey)

	return &EmailService{
		client:      client,
		fromAddress: cfg.EmailFrom,
		frontendURL: cfg.FrontendURL,
		logger:      logger,
	}
}

// SendVerificationEmail sends an email verification link
func (s *EmailService) SendVerificationEmail(to, username, token string) error {
	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", s.frontendURL, token)

	htmlBody := s.renderVerificationHTML(username, verifyURL)
	textBody := s.renderVerificationText(username, verifyURL)

	params := &resend.SendEmailRequest{
		From:    s.fromAddress,
		To:      []string{to},
		Subject: "Verify your NAS.AI email address",
		Html:    htmlBody,
		Text:    textBody,
	}

	sent, err := s.client.Emails.Send(params)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"to":    to,
			"error": err.Error(),
		}).Error("Failed to send verification email")
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"to":       to,
		"email_id": sent.Id,
	}).Info("Verification email sent successfully")

	return nil
}

// SendPasswordResetEmail sends a password reset link
func (s *EmailService) SendPasswordResetEmail(to, username, token string) error {
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.frontendURL, token)

	htmlBody := s.renderPasswordResetHTML(username, resetURL)
	textBody := s.renderPasswordResetText(username, resetURL)

	params := &resend.SendEmailRequest{
		From:    s.fromAddress,
		To:      []string{to},
		Subject: "Reset your NAS.AI password",
		Html:    htmlBody,
		Text:    textBody,
	}

	sent, err := s.client.Emails.Send(params)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"to":    to,
			"error": err.Error(),
		}).Error("Failed to send password reset email")
		return fmt.Errorf("failed to send password reset email: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"to":       to,
		"email_id": sent.Id,
	}).Info("Password reset email sent successfully")

	return nil
}

// SendWelcomeEmail sends a welcome email after email verification
func (s *EmailService) SendWelcomeEmail(to, username string) error {
	htmlBody := s.renderWelcomeHTML(username)
	textBody := s.renderWelcomeText(username)

	params := &resend.SendEmailRequest{
		From:    s.fromAddress,
		To:      []string{to},
		Subject: "Welcome to NAS.AI!",
		Html:    htmlBody,
		Text:    textBody,
	}

	sent, err := s.client.Emails.Send(params)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"to":    to,
			"error": err.Error(),
		}).Error("Failed to send welcome email")
		return fmt.Errorf("failed to send welcome email: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"to":       to,
		"email_id": sent.Id,
	}).Info("Welcome email sent successfully")

	return nil
}

// SendGenericEmail sends a generic email (text/html)
func (s *EmailService) SendGenericEmail(to, subject, body string) error {
	params := &resend.SendEmailRequest{
		From:    s.fromAddress,
		To:      []string{to},
		Subject: subject,
		Html:    body, // Treating body as HTML for flexibility
		Text:    body, // Fallback
	}

	sent, err := s.client.Emails.Send(params)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"to":    to,
			"error": err.Error(),
		}).Error("Failed to send generic email")
		return fmt.Errorf("failed to send generic email: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"to":       to,
		"email_id": sent.Id,
	}).Info("Generic email sent successfully")

	return nil
}

// renderVerificationHTML renders the email verification HTML template
func (s *EmailService) renderVerificationHTML(username, verifyURL string) string {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .button { display: inline-block; padding: 12px 24px; background-color: #007bff; color: #ffffff; text-decoration: none; border-radius: 4px; margin: 20px 0; }
        .footer { margin-top: 30px; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Welcome to NAS.AI, {{.Username}}!</h1>
        <p>Thank you for signing up. Please verify your email address by clicking the button below:</p>
        <a href="{{.VerifyURL}}" class="button">Verify Email Address</a>
        <p>Or copy and paste this link into your browser:</p>
        <p><a href="{{.VerifyURL}}">{{.VerifyURL}}</a></p>
        <p><strong>This link expires in 24 hours.</strong></p>
        <div class="footer">
            <p>If you didn't create an account with NAS.AI, please ignore this email.</p>
            <p>&copy; 2025 NAS.AI. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`
	t := template.Must(template.New("verification").Parse(tmpl))
	var buf bytes.Buffer
	t.Execute(&buf, map[string]string{"Username": username, "VerifyURL": verifyURL})
	return buf.String()
}

// renderVerificationText renders the plain text version
func (s *EmailService) renderVerificationText(username, verifyURL string) string {
	return fmt.Sprintf(`Welcome to NAS.AI, %s!

Thank you for signing up. Please verify your email address by visiting this link:

%s

This link expires in 24 hours.

If you didn't create an account with NAS.AI, please ignore this email.

© 2025 NAS.AI. All rights reserved.
`, username, verifyURL)
}

// renderPasswordResetHTML renders the password reset HTML template
func (s *EmailService) renderPasswordResetHTML(username, resetURL string) string {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .button { display: inline-block; padding: 12px 24px; background-color: #dc3545; color: #ffffff; text-decoration: none; border-radius: 4px; margin: 20px 0; }
        .footer { margin-top: 30px; font-size: 12px; color: #666; }
        .warning { background-color: #fff3cd; border-left: 4px solid #ffc107; padding: 12px; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Password Reset Request</h1>
        <p>Hello {{.Username}},</p>
        <p>You requested to reset your password for your NAS.AI account. Click the button below to reset it:</p>
        <a href="{{.ResetURL}}" class="button">Reset Password</a>
        <p>Or copy and paste this link into your browser:</p>
        <p><a href="{{.ResetURL}}">{{.ResetURL}}</a></p>
        <p><strong>This link expires in 1 hour.</strong></p>
        <div class="warning">
            <strong>⚠️ Important:</strong> If you didn't request this password reset, please ignore this email. Your password will remain unchanged.
        </div>
        <div class="footer">
            <p>&copy; 2025 NAS.AI. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`
	t := template.Must(template.New("password_reset").Parse(tmpl))
	var buf bytes.Buffer
	t.Execute(&buf, map[string]string{"Username": username, "ResetURL": resetURL})
	return buf.String()
}

// renderPasswordResetText renders the plain text version
func (s *EmailService) renderPasswordResetText(username, resetURL string) string {
	return fmt.Sprintf(`Password Reset Request

Hello %s,

You requested to reset your password for your NAS.AI account. Visit this link to reset it:

%s

This link expires in 1 hour.

⚠️ IMPORTANT: If you didn't request this password reset, please ignore this email. Your password will remain unchanged.

© 2025 NAS.AI. All rights reserved.
`, username, resetURL)
}

// renderWelcomeHTML renders the welcome email HTML template
func (s *EmailService) renderWelcomeHTML(username string) string {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { text-align: center; padding: 20px 0; }
        .features { background-color: #f8f9fa; padding: 20px; border-radius: 8px; margin: 20px 0; }
        .footer { margin-top: 30px; font-size: 12px; color: #666; text-align: center; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🎉 Welcome to NAS.AI!</h1>
        </div>
        <p>Hello {{.Username}},</p>
        <p>Your email has been successfully verified! You now have full access to your NAS.AI account.</p>
        <div class="features">
            <h3>What you can do now:</h3>
            <ul>
                <li>📁 Upload and manage your files</li>
                <li>🔒 Secure cloud storage</li>
                <li>🚀 Fast file access from anywhere</li>
                <li>🔐 Enterprise-grade security</li>
            </ul>
        </div>
        <p>Get started by logging in to your account!</p>
        <div class="footer">
            <p>Need help? Contact us at support@nas.ai</p>
            <p>&copy; 2025 NAS.AI. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`
	t := template.Must(template.New("welcome").Parse(tmpl))
	var buf bytes.Buffer
	t.Execute(&buf, map[string]string{"Username": username})
	return buf.String()
}

// renderWelcomeText renders the plain text version
func (s *EmailService) renderWelcomeText(username string) string {
	return fmt.Sprintf(`🎉 Welcome to NAS.AI!

Hello %s,

Your email has been successfully verified! You now have full access to your NAS.AI account.

What you can do now:
- 📁 Upload and manage your files
- 🔒 Secure cloud storage
- 🚀 Fast file access from anywhere
- 🔐 Enterprise-grade security

Get started by logging in to your account!

Need help? Contact us at support@nas.ai

© 2025 NAS.AI. All rights reserved.
`, username)
}
