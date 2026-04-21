package notifications

import (
	"fmt"
	"time"
)

// EmailTemplate provides formatted HTML and text bodies for various email types.
type EmailTemplate struct {
	Subject  string
	TextBody string
	HtmlBody string
}

// VerifyEmailTemplate generates a formatted verification email template.
func VerifyEmailTemplate(verificationURL string, expiresAt time.Time) EmailTemplate {
	expirationTime := expiresAt.Format("02 Jan 2006, 15:04 MST")

	textBody := fmt.Sprintf(`Welcome to ManaWise!

Verify your email address to start analyzing EDH decks with AI.

Click here to verify:
%s

This link expires at: %s

If you didn't create this account, you can safely ignore this email.

---
ManaWise Team
🔮 Mana-Wise AI`, verificationURL, expirationTime)

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en" xmlns="http://www.w3.org/1999/xhtml">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Verify Your ManaWise Account</title>
    <style type="text/css">
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: 'Segoe UI', system-ui, -apple-system, sans-serif;
            background: linear-gradient(135deg, rgba(124,92,191,0.1) 0%%, rgba(229,162,42,0.05) 100%%), #0e0e14;
            color: #e8e8f0;
            line-height: 1.6;
        }
        .container {
            max-width: 580px;
            margin: 0 auto;
            padding: 20px;
        }
        .card {
            background: linear-gradient(180deg, #16161f 0%%, rgba(22,22,31,0.8) 100%%);
            border: 1px solid #2a2a3a;
            border-radius: 12px;
            padding: 40px 32px;
            box-shadow: 0 4px 24px rgba(0,0,0,0.5);
        }
        .header {
            text-align: center;
            padding-bottom: 30px;
            border-bottom: 1px solid #2a2a3a;
            margin-bottom: 30px;
        }
        .logo {
            font-size: 32px;
            font-weight: 700;
            letter-spacing: -0.5px;
            margin-bottom: 8px;
        }
        .logo-main { color: #e8e8f0; }
        .logo-accent { color: #9b7fe0; }
        .tagline {
            color: #8888aa;
            font-size: 13px;
            letter-spacing: 0.05em;
            text-transform: uppercase;
        }
        .content {
            padding: 20px 0;
        }
        .greeting {
            font-size: 20px;
            font-weight: 600;
            color: #e8e8f0;
            margin-bottom: 16px;
        }
        .description {
            color: #b8b8cc;
            font-size: 14px;
            line-height: 1.7;
            margin-bottom: 28px;
        }
        .cta-button {
            display: inline-block;
            padding: 14px 36px;
            background: linear-gradient(135deg, #7c5cbf 0%%, #9b7fe0 100%%);
            color: white;
            text-decoration: none;
            border-radius: 8px;
            font-weight: 600;
            font-size: 15px;
            letter-spacing: 0.02em;
            transition: all 0.2s ease;
            box-shadow: 0 4px 12px rgba(124,92,191,0.3);
            border: none;
            cursor: pointer;
            margin: 24px 0;
        }
        .cta-button:hover {
            transform: translateY(-2px);
            box-shadow: 0 6px 20px rgba(124,92,191,0.4);
        }
        .link-section {
            margin-top: 24px;
            padding: 16px;
            background: rgba(124,92,191,0.08);
            border-left: 3px solid #9b7fe0;
            border-radius: 4px;
        }
        .link-section-label {
            color: #8888aa;
            font-size: 12px;
            text-transform: uppercase;
            letter-spacing: 0.04em;
            margin-bottom: 8px;
            display: block;
            font-weight: 600;
        }
        .verification-link {
            color: #9b7fe0;
            font-size: 12px;
            word-break: break-all;
            font-family: 'Courier New', monospace;
            line-height: 1.5;
        }
        .expiration {
            color: #e5a22a;
            font-size: 13px;
            font-weight: 500;
            margin-top: 20px;
            padding: 12px;
            background: rgba(229,162,42,0.08);
            border-radius: 6px;
        }
        .footer {
            border-top: 1px solid #2a2a3a;
            padding-top: 24px;
            margin-top: 32px;
            text-align: center;
            color: #8888aa;
            font-size: 12px;
        }
        .footer-logo {
            font-size: 16px;
            margin: 8px 0;
            color: #9b7fe0;
        }
        .footer-note {
            margin-top: 12px;
            color: #666699;
            font-size: 11px;
        }
        .safe-note {
            color: #8888aa;
            font-size: 13px;
            margin-top: 20px;
            font-style: italic;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="card">
            <div class="header">
                <div class="logo">
                    <span class="logo-main">🔮 Mana</span><span class="logo-accent">Wise</span>
                </div>
                <div class="tagline">EDH Deck Analysis with AI</div>
            </div>
            
            <div class="content">
                <p class="greeting">Welcome to ManaWise!</p>
                <p class="description">
                    Verify your email address to unlock powerful AI-driven analysis tools for your Commander decks.
                    Get instant insights on power level, mana curves, and strategic suggestions.
                </p>
                
                <a href="%s" class="cta-button" style="color: white;">Verify Email Address</a>
                
                <div class="link-section">
                    <span class="link-section-label">Or copy this link:</span>
                    <div class="verification-link">%s</div>
                </div>
                
                <div class="expiration">
                    ⏱️ This verification link expires on<br>
                    <strong>%s</strong>
                </div>
                
                <p class="safe-note">
                    If you didn't create this account, you can safely ignore this email.
                </p>
            </div>
            
            <div class="footer">
                <p>ManaWise Team</p>
                <p class="footer-logo">🔮 Mana-Wise AI</p>
                <p class="footer-note">Empowering MTG Commanders since 2026</p>
            </div>
        </div>
    </div>
</body>
</html>`, verificationURL, verificationURL, expirationTime)

	return EmailTemplate{
		Subject:  "Verify your ManaWise account",
		TextBody: textBody,
		HtmlBody: htmlBody,
	}
}

// ResetPasswordTemplate generates a formatted password reset email template.
func ResetPasswordTemplate(resetURL string, expiresAt time.Time) EmailTemplate {
	expirationTime := expiresAt.Format("02 Jan 2006, 15:04 MST")

	textBody := fmt.Sprintf(`Password Reset Request

We received a request to reset your ManaWise account password.

Click here to reset your password:
%s

This link expires at: %s

If you didn't request this, your account is still secure.
Your password won't change unless you click the link above.

---
ManaWise Team
🔮 Mana-Wise AI`, resetURL, expirationTime)

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en" xmlns="http://www.w3.org/1999/xhtml">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Reset Your ManaWise Password</title>
    <style type="text/css">
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: 'Segoe UI', system-ui, -apple-system, sans-serif;
            background: linear-gradient(135deg, rgba(124,92,191,0.1) 0%%, rgba(229,162,42,0.05) 100%%), #0e0e14;
            color: #e8e8f0;
            line-height: 1.6;
        }
        .container {
            max-width: 580px;
            margin: 0 auto;
            padding: 20px;
        }
        .card {
            background: linear-gradient(180deg, #16161f 0%%, rgba(22,22,31,0.8) 100%%);
            border: 1px solid #2a2a3a;
            border-radius: 12px;
            padding: 40px 32px;
            box-shadow: 0 4px 24px rgba(0,0,0,0.5);
        }
        .header {
            text-align: center;
            padding-bottom: 30px;
            border-bottom: 1px solid #2a2a3a;
            margin-bottom: 30px;
        }
        .logo {
            font-size: 32px;
            font-weight: 700;
            letter-spacing: -0.5px;
            margin-bottom: 8px;
        }
        .logo-main { color: #e8e8f0; }
        .logo-accent { color: #9b7fe0; }
        .tagline {
            color: #8888aa;
            font-size: 13px;
            letter-spacing: 0.05em;
            text-transform: uppercase;
        }
        .content {
            padding: 20px 0;
        }
        .greeting {
            font-size: 20px;
            font-weight: 600;
            color: #e8e8f0;
            margin-bottom: 16px;
        }
        .description {
            color: #b8b8cc;
            font-size: 14px;
            line-height: 1.7;
            margin-bottom: 28px;
        }
        .cta-button {
            display: inline-block;
            padding: 14px 36px;
            background: linear-gradient(135deg, #7c5cbf 0%%, #9b7fe0 100%%);
            color: white;
            text-decoration: none;
            border-radius: 8px;
            font-weight: 600;
            font-size: 15px;
            letter-spacing: 0.02em;
            transition: all 0.2s ease;
            box-shadow: 0 4px 12px rgba(124,92,191,0.3);
            border: none;
            cursor: pointer;
            margin: 24px 0;
        }
        .cta-button:hover {
            transform: translateY(-2px);
            box-shadow: 0 6px 20px rgba(124,92,191,0.4);
        }
        .link-section {
            margin-top: 24px;
            padding: 16px;
            background: rgba(124,92,191,0.08);
            border-left: 3px solid #9b7fe0;
            border-radius: 4px;
        }
        .link-section-label {
            color: #8888aa;
            font-size: 12px;
            text-transform: uppercase;
            letter-spacing: 0.04em;
            margin-bottom: 8px;
            display: block;
            font-weight: 600;
        }
        .reset-link {
            color: #9b7fe0;
            font-size: 12px;
            word-break: break-all;
            font-family: 'Courier New', monospace;
            line-height: 1.5;
        }
        .expiration {
            color: #e5a22a;
            font-size: 13px;
            font-weight: 500;
            margin-top: 20px;
            padding: 12px;
            background: rgba(229,162,42,0.08);
            border-radius: 6px;
        }
        .security-warning {
            background: rgba(229,85,85,0.12);
            border: 1px solid rgba(229,85,85,0.35);
            border-radius: 6px;
            padding: 14px;
            margin: 20px 0;
            color: #ff9999;
            font-size: 13px;
            line-height: 1.6;
        }
        .security-warning strong {
            display: block;
            margin-bottom: 6px;
            color: #ff7777;
        }
        .footer {
            border-top: 1px solid #2a2a3a;
            padding-top: 24px;
            margin-top: 32px;
            text-align: center;
            color: #8888aa;
            font-size: 12px;
        }
        .footer-logo {
            font-size: 16px;
            margin: 8px 0;
            color: #9b7fe0;
        }
        .footer-note {
            margin-top: 12px;
            color: #666699;
            font-size: 11px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="card">
            <div class="header">
                <div class="logo">
                    <span class="logo-main">🔮 Mana</span><span class="logo-accent">Wise</span>
                </div>
                <div class="tagline">EDH Deck Analysis with AI</div>
            </div>
            
            <div class="content">
                <p class="greeting">Password Reset Request</p>
                <p class="description">
                    We received a request to reset your ManaWise account password.
                    Click the button below to create a new secure password.
                </p>
                
                <a href="%s" class="cta-button" style="color: white;">Reset Password</a>
                
                <div class="link-section">
                    <span class="link-section-label">Or copy this link:</span>
                    <div class="reset-link">%s</div>
                </div>
                
                <div class="expiration">
                    ⏱️ This reset link expires on<br>
                    <strong>%s</strong>
                </div>
                
                <div class="security-warning">
                    <strong>⚠️ Didn't request this?</strong>
                    Your account is still secure. Your password won't change unless you click the link above.
                </div>
            </div>
            
            <div class="footer">
                <p>ManaWise Team</p>
                <p class="footer-logo">🔮 Mana-Wise AI</p>
                <p class="footer-note">Keep your account secure</p>
            </div>
        </div>
    </div>
</body>
</html>`, resetURL, resetURL, expirationTime)

	return EmailTemplate{
		Subject:  "ManaWise password reset",
		TextBody: textBody,
		HtmlBody: htmlBody,
	}
}
